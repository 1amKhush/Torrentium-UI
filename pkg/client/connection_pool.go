package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/1amkhush/torrentium/pkg/logger"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PooledConnection wraps a WebRTC peer connection with pool metadata
type PooledConnection struct {
	Peer      *SimpleWebRTCPeer
	PeerID    peer.ID
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int
	InUse     bool
	mu        sync.Mutex
}

// ConnectionPool manages a pool of reusable WebRTC peer connections
type ConnectionPool struct {
	connections    map[peer.ID]*PooledConnection
	mu             sync.RWMutex
	maxConnections int
	maxIdleTime    time.Duration
	maxAge         time.Duration
	onMessage      func(msg interface{}, peer *SimpleWebRTCPeer)
	onClose        func(peerID peer.ID)
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
}

// ConnectionPoolConfig holds configuration for the connection pool
type ConnectionPoolConfig struct {
	MaxConnections  int           // Maximum number of pooled connections
	MaxIdleTime     time.Duration // Max time a connection can be idle before cleanup
	MaxAge          time.Duration // Max age of a connection before forced renewal
	CleanupInterval time.Duration // How often to run cleanup
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxConnections:  20,
		MaxIdleTime:     5 * time.Minute,
		MaxAge:          30 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config ConnectionPoolConfig, onMessage func(msg interface{}, peer *SimpleWebRTCPeer), onClose func(peerID peer.ID)) *ConnectionPool {
	if config.MaxConnections <= 0 {
		config.MaxConnections = 20
	}
	if config.MaxIdleTime <= 0 {
		config.MaxIdleTime = 5 * time.Minute
	}
	if config.MaxAge <= 0 {
		config.MaxAge = 30 * time.Minute
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Minute
	}

	pool := &ConnectionPool{
		connections:    make(map[peer.ID]*PooledConnection),
		maxConnections: config.MaxConnections,
		maxIdleTime:    config.MaxIdleTime,
		maxAge:         config.MaxAge,
		onClose:        onClose,
		stopCleanup:    make(chan struct{}),
	}

	// Start cleanup routine
	pool.cleanupTicker = time.NewTicker(config.CleanupInterval)
	go pool.cleanupRoutine()

	return pool
}

// GetOrCreate retrieves an existing connection or creates a new one
func (p *ConnectionPool) GetOrCreate(ctx context.Context, peerID peer.ID, createFunc func() (*SimpleWebRTCPeer, error)) (*PooledConnection, error) {
	log := logger.WithComponent("connection-pool")

	// First, try to get existing connection
	p.mu.RLock()
	if pooled, exists := p.connections[peerID]; exists {
		p.mu.RUnlock()

		pooled.mu.Lock()
		if pooled.Peer != nil && pooled.Peer.GetConnectionState() == ConnectionStateConnected && !pooled.InUse {
			pooled.InUse = true
			pooled.LastUsed = time.Now()
			pooled.UseCount++
			pooled.mu.Unlock()

			log.Debug().Str("peer_id", peerID.String()).Int("use_count", pooled.UseCount).Msg("Reusing pooled connection")
			return pooled, nil
		}
		pooled.mu.Unlock()

		// Connection exists but is not usable, remove it
		p.Remove(peerID)
	} else {
		p.mu.RUnlock()
	}

	// Check if we have room for a new connection
	p.mu.Lock()
	if len(p.connections) >= p.maxConnections {
		// Try to evict least recently used idle connection
		if !p.evictLRU() {
			p.mu.Unlock()
			return nil, fmt.Errorf("connection pool is full and no connections can be evicted")
		}
	}
	p.mu.Unlock()

	// Create new connection
	log.Debug().Str("peer_id", peerID.String()).Msg("Creating new pooled connection")

	webrtcPeer, err := createFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	pooled := &PooledConnection{
		Peer:      webrtcPeer,
		PeerID:    peerID,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
		InUse:     true,
	}

	p.mu.Lock()
	p.connections[peerID] = pooled
	p.mu.Unlock()

	log.Info().Str("peer_id", peerID.String()).Int("pool_size", len(p.connections)).Msg("Added new connection to pool")

	return pooled, nil
}

// Get retrieves an existing connection from the pool
func (p *ConnectionPool) Get(peerID peer.ID) *PooledConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if pooled, exists := p.connections[peerID]; exists {
		pooled.mu.Lock()
		if pooled.Peer != nil && pooled.Peer.GetConnectionState() == ConnectionStateConnected {
			pooled.LastUsed = time.Now()
			pooled.mu.Unlock()
			return pooled
		}
		pooled.mu.Unlock()
	}
	return nil
}

// Release marks a connection as no longer in use but keeps it in the pool
func (p *ConnectionPool) Release(peerID peer.ID) {
	p.mu.RLock()
	pooled, exists := p.connections[peerID]
	p.mu.RUnlock()

	if exists {
		pooled.mu.Lock()
		pooled.InUse = false
		pooled.LastUsed = time.Now()
		pooled.mu.Unlock()

		logger.Debug().Str("peer_id", peerID.String()).Msg("Released connection back to pool")
	}
}

// Remove removes a connection from the pool and closes it
func (p *ConnectionPool) Remove(peerID peer.ID) {
	p.mu.Lock()
	pooled, exists := p.connections[peerID]
	if exists {
		delete(p.connections, peerID)
	}
	p.mu.Unlock()

	if exists && pooled.Peer != nil {
		pooled.Peer.Close()
		logger.Debug().Str("peer_id", peerID.String()).Msg("Removed connection from pool")
	}
}

// evictLRU evicts the least recently used idle connection
// Must be called with write lock held
func (p *ConnectionPool) evictLRU() bool {
	var oldest *PooledConnection
	var oldestID peer.ID

	for id, pooled := range p.connections {
		pooled.mu.Lock()
		isIdle := !pooled.InUse
		lastUsed := pooled.LastUsed
		pooled.mu.Unlock()

		if isIdle && (oldest == nil || lastUsed.Before(oldest.LastUsed)) {
			oldest = pooled
			oldestID = id
		}
	}

	if oldest != nil {
		delete(p.connections, oldestID)
		if oldest.Peer != nil {
			oldest.Peer.Close()
		}
		logger.Debug().Str("peer_id", oldestID.String()).Msg("Evicted LRU connection from pool")
		return true
	}

	return false
}

// cleanupRoutine periodically cleans up stale connections
func (p *ConnectionPool) cleanupRoutine() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.cleanup()
		case <-p.stopCleanup:
			return
		}
	}
}

// cleanup removes stale and expired connections
func (p *ConnectionPool) cleanup() {
	log := logger.WithComponent("connection-pool")
	now := time.Now()
	var toRemove []peer.ID

	p.mu.RLock()
	for id, pooled := range p.connections {
		pooled.mu.Lock()

		// Check if connection is too old
		if now.Sub(pooled.CreatedAt) > p.maxAge {
			toRemove = append(toRemove, id)
			pooled.mu.Unlock()
			continue
		}

		// Check if connection has been idle too long
		if !pooled.InUse && now.Sub(pooled.LastUsed) > p.maxIdleTime {
			toRemove = append(toRemove, id)
			pooled.mu.Unlock()
			continue
		}

		// Check if connection is dead
		if pooled.Peer != nil && pooled.Peer.GetConnectionState() != ConnectionStateConnected {
			toRemove = append(toRemove, id)
		}

		pooled.mu.Unlock()
	}
	p.mu.RUnlock()

	// Remove stale connections
	for _, id := range toRemove {
		p.Remove(id)
	}

	if len(toRemove) > 0 {
		log.Debug().Int("removed", len(toRemove)).Int("remaining", p.Size()).Msg("Cleaned up stale connections")
	}
}

// Size returns the current number of connections in the pool
func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// ActiveCount returns the number of connections currently in use
func (p *ConnectionPool) ActiveCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, pooled := range p.connections {
		pooled.mu.Lock()
		if pooled.InUse {
			count++
		}
		pooled.mu.Unlock()
	}
	return count
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalConnections: len(p.connections),
		MaxConnections:   p.maxConnections,
	}

	for _, pooled := range p.connections {
		pooled.mu.Lock()
		if pooled.InUse {
			stats.ActiveConnections++
		} else {
			stats.IdleConnections++
		}
		stats.TotalUseCount += pooled.UseCount
		pooled.mu.Unlock()
	}

	return stats
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
	MaxConnections    int
	TotalUseCount     int
}

// Close closes all connections and stops the cleanup routine
func (p *ConnectionPool) Close() {
	// Stop cleanup routine
	close(p.stopCleanup)
	p.cleanupTicker.Stop()

	// Close all connections
	p.mu.Lock()
	for id, pooled := range p.connections {
		if pooled.Peer != nil {
			pooled.Peer.Close()
		}
		delete(p.connections, id)
	}
	p.mu.Unlock()

	logger.Info().Msg("Connection pool closed")
}

// ForEach iterates over all connections in the pool
func (p *ConnectionPool) ForEach(fn func(peerID peer.ID, conn *PooledConnection)) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for id, pooled := range p.connections {
		fn(id, pooled)
	}
}
