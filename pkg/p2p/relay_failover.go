package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/1amkhush/torrentium/pkg/config"
	"github.com/1amkhush/torrentium/pkg/logger"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	relayv2client "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
)

// RelayHealth tracks the health status of a relay server
type RelayHealth struct {
	Address      string
	PeerID       peer.ID
	IsHealthy    bool
	LastCheck    time.Time
	LastSuccess  time.Time
	FailureCount int
	Latency      time.Duration
	Reserved     bool
}

// RelayFailoverManager manages relay server failover
type RelayFailoverManager struct {
	mu            sync.RWMutex
	relays        []*RelayHealth
	currentRelay  *RelayHealth
	host          host.Host
	checkInterval time.Duration
	maxFailures   int
	healthCh      chan *RelayHealth
	stopCh        chan struct{}
	onRelayChange func(oldRelay, newRelay string)
}

// NewRelayFailoverManager creates a new relay failover manager
func NewRelayFailoverManager(h host.Host) *RelayFailoverManager {
	return &RelayFailoverManager{
		relays:        make([]*RelayHealth, 0),
		host:          h,
		checkInterval: 30 * time.Second,
		maxFailures:   3,
		healthCh:      make(chan *RelayHealth, 10),
		stopCh:        make(chan struct{}),
	}
}

// SetOnRelayChange sets the callback for relay changes
func (m *RelayFailoverManager) SetOnRelayChange(fn func(oldRelay, newRelay string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRelayChange = fn
}

// AddRelay adds a relay to the manager
func (m *RelayFailoverManager) AddRelay(address string) error {
	maddr, err := ma.NewMultiaddr(address)
	if err != nil {
		return fmt.Errorf("invalid relay multiaddr: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse relay peer info: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	for _, r := range m.relays {
		if r.PeerID == peerInfo.ID {
			return nil // Already exists
		}
	}

	m.relays = append(m.relays, &RelayHealth{
		Address:      address,
		PeerID:       peerInfo.ID,
		IsHealthy:    false,
		FailureCount: 0,
	})

	logger.Info().Str("address", address).Msg("Added relay to failover manager")
	return nil
}

// AddRelaysFromConfig adds all relays from the configuration
func (m *RelayFailoverManager) AddRelaysFromConfig() error {
	cfg := config.Global()

	for _, addr := range cfg.P2P.RelayAddresses {
		if err := m.AddRelay(addr); err != nil {
			logger.Warn().Err(err).Str("address", addr).Msg("Failed to add relay from config")
		}
	}

	// Also try to fetch from HTTP if configured
	if cfg.P2P.RelayHTTPURL != "" {
		addrs, err := GetRelayAddresses()
		if err == nil {
			for _, addr := range addrs {
				if err := m.AddRelay(addr); err != nil {
					logger.Warn().Err(err).Str("address", addr).Msg("Failed to add relay from HTTP")
				}
			}
		}
	}

	m.mu.RLock()
	count := len(m.relays)
	m.mu.RUnlock()

	if count == 0 {
		return fmt.Errorf("no relays available")
	}

	logger.Info().Int("count", count).Msg("Loaded relays into failover manager")
	return nil
}

// Start starts the failover manager
func (m *RelayFailoverManager) Start(ctx context.Context) error {
	// Initial health check
	if err := m.checkAllRelays(ctx); err != nil {
		return fmt.Errorf("initial relay health check failed: %w", err)
	}

	// Connect to best relay
	if err := m.connectToBestRelay(ctx); err != nil {
		return fmt.Errorf("failed to connect to any relay: %w", err)
	}

	// Start background health monitoring
	go m.monitorRelays(ctx)

	return nil
}

// Stop stops the failover manager
func (m *RelayFailoverManager) Stop() {
	close(m.stopCh)
}

// GetCurrentRelay returns the current active relay
func (m *RelayFailoverManager) GetCurrentRelay() *RelayHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentRelay
}

// GetAllRelays returns all known relays
func (m *RelayFailoverManager) GetAllRelays() []*RelayHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*RelayHealth, len(m.relays))
	copy(result, m.relays)
	return result
}

// GetHealthyRelays returns all healthy relays
func (m *RelayFailoverManager) GetHealthyRelays() []*RelayHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var healthy []*RelayHealth
	for _, r := range m.relays {
		if r.IsHealthy {
			healthy = append(healthy, r)
		}
	}
	return healthy
}

// checkAllRelays checks the health of all relays
func (m *RelayFailoverManager) checkAllRelays(ctx context.Context) error {
	m.mu.RLock()
	relays := make([]*RelayHealth, len(m.relays))
	copy(relays, m.relays)
	m.mu.RUnlock()

	var wg sync.WaitGroup
	for _, relay := range relays {
		wg.Add(1)
		go func(r *RelayHealth) {
			defer wg.Done()
			m.checkRelayHealth(ctx, r)
		}(relay)
	}
	wg.Wait()

	// Check if any relay is healthy
	m.mu.RLock()
	hasHealthy := false
	for _, r := range m.relays {
		if r.IsHealthy {
			hasHealthy = true
			break
		}
	}
	m.mu.RUnlock()

	if !hasHealthy {
		return fmt.Errorf("no healthy relays available")
	}

	return nil
}

// checkRelayHealth checks the health of a single relay
func (m *RelayFailoverManager) checkRelayHealth(ctx context.Context, relay *RelayHealth) {
	log := logger.WithComponent("relay-failover")

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()

	// Parse the multiaddr
	maddr, err := ma.NewMultiaddr(relay.Address)
	if err != nil {
		relay.IsHealthy = false
		relay.FailureCount++
		log.Warn().Err(err).Str("address", relay.Address).Msg("Invalid relay address")
		return
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		relay.IsHealthy = false
		relay.FailureCount++
		log.Warn().Err(err).Str("address", relay.Address).Msg("Failed to parse relay peer info")
		return
	}

	// Try to connect
	if err := m.host.Connect(checkCtx, *peerInfo); err != nil {
		relay.IsHealthy = false
		relay.FailureCount++
		log.Debug().Err(err).Str("address", relay.Address).Msg("Relay connection failed")
		return
	}

	relay.Latency = time.Since(start)
	relay.LastCheck = time.Now()
	relay.LastSuccess = time.Now()
	relay.IsHealthy = true
	relay.FailureCount = 0

	log.Debug().
		Str("address", relay.Address).
		Dur("latency", relay.Latency).
		Msg("Relay health check passed")
}

// connectToBestRelay connects to the best available relay
func (m *RelayFailoverManager) connectToBestRelay(ctx context.Context) error {
	log := logger.WithComponent("relay-failover")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Sort relays by health and latency
	var bestRelay *RelayHealth
	for _, r := range m.relays {
		if r.IsHealthy {
			if bestRelay == nil || r.Latency < bestRelay.Latency {
				bestRelay = r
			}
		}
	}

	if bestRelay == nil {
		return fmt.Errorf("no healthy relays available")
	}

	// Try to reserve with the relay
	maddr, err := ma.NewMultiaddr(bestRelay.Address)
	if err != nil {
		return fmt.Errorf("invalid relay multiaddr: %w", err)
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse relay peer info: %w", err)
	}

	// Connect first
	if err := m.host.Connect(ctx, *relayInfo); err != nil {
		bestRelay.IsHealthy = false
		bestRelay.FailureCount++
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	// Reserve
	res, err := relayv2client.Reserve(ctx, m.host, *relayInfo)
	if err != nil {
		bestRelay.IsHealthy = false
		bestRelay.FailureCount++
		return fmt.Errorf("relay reservation failed: %w", err)
	}

	oldRelay := m.currentRelay
	bestRelay.Reserved = true
	m.currentRelay = bestRelay

	log.Info().
		Str("address", bestRelay.Address).
		Time("expires", res.Expiration).
		Msg("Connected to relay")

	// Notify of relay change
	if m.onRelayChange != nil && oldRelay != nil && oldRelay.Address != bestRelay.Address {
		go m.onRelayChange(oldRelay.Address, bestRelay.Address)
	}

	return nil
}

// monitorRelays continuously monitors relay health
func (m *RelayFailoverManager) monitorRelays(ctx context.Context) {
	log := logger.WithComponent("relay-failover")
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			// Check all relays
			m.checkAllRelays(ctx)

			// Check if current relay is still healthy
			m.mu.RLock()
			currentRelay := m.currentRelay
			m.mu.RUnlock()

			if currentRelay != nil && (!currentRelay.IsHealthy || currentRelay.FailureCount >= m.maxFailures) {
				log.Warn().
					Str("address", currentRelay.Address).
					Int("failures", currentRelay.FailureCount).
					Msg("Current relay unhealthy, switching")

				if err := m.connectToBestRelay(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to switch to healthy relay")
				}
			}

			// Re-reserve if needed (reservations expire)
			if currentRelay != nil && currentRelay.IsHealthy {
				go func() {
					if err := m.refreshReservation(ctx, currentRelay); err != nil {
						log.Warn().Err(err).Msg("Failed to refresh relay reservation")
					}
				}()
			}
		}
	}
}

// refreshReservation refreshes the relay reservation
func (m *RelayFailoverManager) refreshReservation(ctx context.Context, relay *RelayHealth) error {
	maddr, err := ma.NewMultiaddr(relay.Address)
	if err != nil {
		return err
	}

	relayInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	_, err = relayv2client.Reserve(ctx, m.host, *relayInfo)
	return err
}

// ForceSwitch forces a switch to a different relay
func (m *RelayFailoverManager) ForceSwitch(ctx context.Context) error {
	m.mu.Lock()
	currentRelay := m.currentRelay
	if currentRelay != nil {
		currentRelay.IsHealthy = false // Mark as unhealthy to force switch
	}
	m.mu.Unlock()

	return m.connectToBestRelay(ctx)
}

// ReportFailure reports a failure for the current relay
func (m *RelayFailoverManager) ReportFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentRelay != nil {
		m.currentRelay.FailureCount++
		logger.Warn().
			Str("address", m.currentRelay.Address).
			Int("failures", m.currentRelay.FailureCount).
			Msg("Relay failure reported")
	}
}

// Stats returns statistics about the relay failover manager
type RelayFailoverStats struct {
	TotalRelays    int
	HealthyRelays  int
	CurrentRelay   string
	CurrentLatency time.Duration
}

// GetStats returns current failover statistics
func (m *RelayFailoverManager) GetStats() RelayFailoverStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := RelayFailoverStats{
		TotalRelays: len(m.relays),
	}

	for _, r := range m.relays {
		if r.IsHealthy {
			stats.HealthyRelays++
		}
	}

	if m.currentRelay != nil {
		stats.CurrentRelay = m.currentRelay.Address
		stats.CurrentLatency = m.currentRelay.Latency
	}

	return stats
}
