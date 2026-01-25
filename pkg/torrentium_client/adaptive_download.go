package torrentium_client

import (
	"context"
	"sync"
	"time"

	"github.com/1amkhush/torrentium/pkg/config"
	"github.com/1amkhush/torrentium/pkg/logger"
	"github.com/libp2p/go-libp2p/core/peer"
)

// AdaptiveDownloadManager adjusts parallel download count based on bandwidth
type AdaptiveDownloadManager struct {
	mu                   sync.RWMutex
	currentParallel      int
	minParallel          int
	maxParallel          int
	enabled              bool
	bandwidthSamples     []float64
	lastAdjustment       time.Time
	adjustmentInterval   time.Duration
	targetUtilization    float64
	speedHistory         []float64
	lastSpeedMeasurement time.Time
}

// NewAdaptiveDownloadManager creates a new adaptive download manager
func NewAdaptiveDownloadManager() *AdaptiveDownloadManager {
	cfg := config.Global()
	return &AdaptiveDownloadManager{
		currentParallel:    cfg.Client.MaxParallelDownloads,
		minParallel:        cfg.Client.MinParallelDownloads,
		maxParallel:        cfg.Client.MaxParallelDownloads,
		enabled:            cfg.Client.AdaptiveParallelDownloads,
		bandwidthSamples:   make([]float64, 0, 20),
		adjustmentInterval: 30 * time.Second,
		targetUtilization:  0.8, // Target 80% bandwidth utilization
		speedHistory:       make([]float64, 0, 10),
	}
}

// GetParallelCount returns the current number of parallel downloads to use
func (m *AdaptiveDownloadManager) GetParallelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentParallel
}

// SetEnabled enables or disables adaptive mode
func (m *AdaptiveDownloadManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
	if !enabled {
		cfg := config.Global()
		m.currentParallel = cfg.Client.MaxParallelDownloads
	}
}

// SetLimits sets the min and max parallel downloads
func (m *AdaptiveDownloadManager) SetLimits(min, max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.minParallel = min
	m.maxParallel = max
	if m.currentParallel < min {
		m.currentParallel = min
	}
	if m.currentParallel > max {
		m.currentParallel = max
	}
}

// SetMaxBandwidth sets the maximum bandwidth (currently adjusts limits)
func (m *AdaptiveDownloadManager) SetMaxBandwidth(bytesPerSecond int64) {
	// This is a simplified implementation - could be used for more sophisticated throttling
	m.mu.Lock()
	defer m.mu.Unlock()
	// For now, just log the setting - actual bandwidth limiting would require more implementation
	logger.Info().Int64("max_bandwidth", bytesPerSecond).Msg("Max bandwidth set")
}

// Start starts the adaptive manager (no-op for now, adjustments happen on RecordSpeed)
func (m *AdaptiveDownloadManager) Start() {
	// Adjustments happen automatically in RecordSpeed
}

// Stop stops the adaptive manager
func (m *AdaptiveDownloadManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bandwidthSamples = m.bandwidthSamples[:0]
	m.speedHistory = m.speedHistory[:0]
}

// RecordSpeed records a speed measurement for adaptive adjustment
func (m *AdaptiveDownloadManager) RecordSpeed(bytesPerSecond float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return
	}

	// Add to samples
	m.bandwidthSamples = append(m.bandwidthSamples, bytesPerSecond)
	if len(m.bandwidthSamples) > 20 {
		m.bandwidthSamples = m.bandwidthSamples[1:]
	}

	m.speedHistory = append(m.speedHistory, bytesPerSecond)
	if len(m.speedHistory) > 10 {
		m.speedHistory = m.speedHistory[1:]
	}
	m.lastSpeedMeasurement = time.Now()

	// Check if we should adjust
	if time.Since(m.lastAdjustment) >= m.adjustmentInterval {
		m.adjust()
		m.lastAdjustment = time.Now()
	}
}

// adjust calculates and applies the optimal parallel count
func (m *AdaptiveDownloadManager) adjust() {
	if len(m.bandwidthSamples) < 5 {
		return // Not enough data
	}

	// Calculate average and variance
	var sum, sumSq float64
	for _, s := range m.bandwidthSamples {
		sum += s
		sumSq += s * s
	}
	avg := sum / float64(len(m.bandwidthSamples))
	variance := (sumSq / float64(len(m.bandwidthSamples))) - (avg * avg)

	// Check trend (are we improving or degrading?)
	recentAvg := avg
	if len(m.speedHistory) >= 5 {
		var recentSum float64
		for _, s := range m.speedHistory[len(m.speedHistory)-5:] {
			recentSum += s
		}
		recentAvg = recentSum / 5
	}

	oldParallel := m.currentParallel
	log := logger.WithComponent("adaptive-download")

	// High variance indicates congestion - reduce parallel downloads
	coefficientOfVariation := 0.0
	if avg > 0 {
		coefficientOfVariation = (variance * variance) / avg
	}

	if coefficientOfVariation > 0.3 {
		// High variance - reduce
		if m.currentParallel > m.minParallel {
			m.currentParallel--
			log.Debug().
				Int("old", oldParallel).
				Int("new", m.currentParallel).
				Float64("cv", coefficientOfVariation).
				Msg("Reduced parallel downloads due to high variance")
		}
	} else if recentAvg > avg*1.1 && m.currentParallel < m.maxParallel {
		// Speed is improving, try increasing
		m.currentParallel++
		log.Debug().
			Int("old", oldParallel).
			Int("new", m.currentParallel).
			Float64("recent_avg", recentAvg).
			Float64("overall_avg", avg).
			Msg("Increased parallel downloads due to improving speed")
	} else if recentAvg < avg*0.8 && m.currentParallel > m.minParallel {
		// Speed is degrading, reduce
		m.currentParallel--
		log.Debug().
			Int("old", oldParallel).
			Int("new", m.currentParallel).
			Float64("recent_avg", recentAvg).
			Float64("overall_avg", avg).
			Msg("Reduced parallel downloads due to degrading speed")
	}

	// Reset samples after adjustment
	m.bandwidthSamples = m.bandwidthSamples[:0]
}

// GetStats returns current adaptive manager statistics
func (m *AdaptiveDownloadManager) GetStats() AdaptiveStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgSpeed float64
	if len(m.speedHistory) > 0 {
		var sum float64
		for _, s := range m.speedHistory {
			sum += s
		}
		avgSpeed = sum / float64(len(m.speedHistory))
	}

	return AdaptiveStats{
		CurrentParallel: m.currentParallel,
		MinParallel:     m.minParallel,
		MaxParallel:     m.maxParallel,
		Enabled:         m.enabled,
		AverageSpeed:    avgSpeed,
		SampleCount:     len(m.bandwidthSamples),
	}
}

// AdaptiveStats contains adaptive download manager statistics
type AdaptiveStats struct {
	CurrentParallel int
	MinParallel     int
	MaxParallel     int
	Enabled         bool
	AverageSpeed    float64
	SampleCount     int
}

// EndgameManager handles endgame mode for fast completion of downloads
type EndgameManager struct {
	mu             sync.RWMutex
	enabled        bool
	threshold      float64 // Percentage of pieces remaining to trigger endgame
	activeEndgames map[string]*EndgameState
}

// EndgameState tracks the state of endgame mode for a download
type EndgameState struct {
	CID             string
	TotalPieces     int
	RemainingPieces []int
	PieceRequests   map[int][]peer.ID // piece -> peers that have been requested
	PieceReceived   map[int]bool      // piece -> received
	mu              sync.Mutex
}

// NewEndgameManager creates a new endgame manager
func NewEndgameManager() *EndgameManager {
	cfg := config.Global()
	return &EndgameManager{
		enabled:        cfg.Client.EnableEndgameMode,
		threshold:      cfg.Client.EndgameThreshold,
		activeEndgames: make(map[string]*EndgameState),
	}
}

// SetEnabled enables or disables endgame mode
func (m *EndgameManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// SetThreshold sets the endgame threshold (percentage of pieces remaining)
func (m *EndgameManager) SetThreshold(threshold float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if threshold > 0 && threshold < 1 {
		m.threshold = threshold
	}
}

// ShouldEnterEndgame checks if endgame mode should be activated
func (m *EndgameManager) ShouldEnterEndgame(cid string, completedPieces, totalPieces int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled || totalPieces == 0 {
		return false
	}

	// Check if already in endgame
	if _, exists := m.activeEndgames[cid]; exists {
		return true
	}

	remainingRatio := float64(totalPieces-completedPieces) / float64(totalPieces)
	return remainingRatio <= m.threshold
}

// StartEndgame initializes endgame mode for a download
func (m *EndgameManager) StartEndgame(cid string, totalPieces int, remainingPieces []int) *EndgameState {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := &EndgameState{
		CID:             cid,
		TotalPieces:     totalPieces,
		RemainingPieces: remainingPieces,
		PieceRequests:   make(map[int][]peer.ID),
		PieceReceived:   make(map[int]bool),
	}

	m.activeEndgames[cid] = state
	logger.Info().
		Str("cid", cid).
		Int("remaining_pieces", len(remainingPieces)).
		Msg("Entered endgame mode")

	return state
}

// GetEndgameState returns the endgame state for a download
func (m *EndgameManager) GetEndgameState(cid string) *EndgameState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeEndgames[cid]
}

// EndEndgame removes a download from endgame mode
func (m *EndgameManager) EndEndgame(cid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeEndgames, cid)
	logger.Info().Str("cid", cid).Msg("Exited endgame mode")
}

// RecordRequest records that a piece was requested from a peer
func (s *EndgameState) RecordRequest(pieceIndex int, peerID peer.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PieceRequests[pieceIndex] = append(s.PieceRequests[pieceIndex], peerID)
}

// RecordReceived records that a piece was received
func (s *EndgameState) RecordReceived(pieceIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PieceReceived[pieceIndex] = true
}

// IsReceived checks if a piece has been received
func (s *EndgameState) IsReceived(pieceIndex int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.PieceReceived[pieceIndex]
}

// GetPeersToRequest returns peers that haven't been requested for a piece
func (s *EndgameState) GetPeersToRequest(pieceIndex int, allPeers []peer.ID) []peer.ID {
	s.mu.Lock()
	defer s.mu.Unlock()

	requested := make(map[peer.ID]bool)
	for _, p := range s.PieceRequests[pieceIndex] {
		requested[p] = true
	}

	var toRequest []peer.ID
	for _, p := range allPeers {
		if !requested[p] {
			toRequest = append(toRequest, p)
		}
	}
	return toRequest
}

// GetRemainingPieces returns pieces that haven't been received
func (s *EndgameState) GetRemainingPieces() []int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var remaining []int
	for _, p := range s.RemainingPieces {
		if !s.PieceReceived[p] {
			remaining = append(remaining, p)
		}
	}
	return remaining
}

// DHTRetryHelper provides retry logic with exponential backoff for DHT operations
type DHTRetryHelper struct {
	maxAttempts int
	backoffBase time.Duration
	backoffMax  time.Duration
}

// NewDHTRetryHelper creates a new DHT retry helper
func NewDHTRetryHelper() *DHTRetryHelper {
	cfg := config.Global()
	return &DHTRetryHelper{
		maxAttempts: cfg.Client.DHTRetryAttempts,
		backoffBase: cfg.Client.DHTRetryBackoffBase,
		backoffMax:  cfg.Client.DHTRetryBackoffMax,
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func(ctx context.Context) error

// Execute executes a function with retry logic
func (h *DHTRetryHelper) Execute(ctx context.Context, operation string, fn RetryableFunc) error {
	log := logger.WithComponent("dht-retry")
	var lastErr error

	for attempt := 1; attempt <= h.maxAttempts; attempt++ {
		err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				log.Info().
					Str("operation", operation).
					Int("attempts", attempt).
					Msg("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err
		log.Warn().
			Str("operation", operation).
			Int("attempt", attempt).
			Int("max_attempts", h.maxAttempts).
			Err(err).
			Msg("Operation failed, will retry")

		if attempt < h.maxAttempts {
			backoff := h.calculateBackoff(attempt)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Continue to retry
			}
		}
	}

	log.Error().
		Str("operation", operation).
		Int("attempts", h.maxAttempts).
		Err(lastErr).
		Msg("Operation failed after all retries")

	return lastErr
}

// calculateBackoff calculates the backoff duration for an attempt
func (h *DHTRetryHelper) calculateBackoff(attempt int) time.Duration {
	backoff := h.backoffBase * time.Duration(1<<uint(attempt-1))
	if backoff > h.backoffMax {
		backoff = h.backoffMax
	}
	return backoff
}

// DownloadCheckpointer handles saving download state periodically
type DownloadCheckpointer struct {
	interval       int // Number of pieces between checkpoints
	mu             sync.Mutex
	lastCheckpoint map[string]int // CID -> last checkpoint piece count
}

// NewDownloadCheckpointer creates a new checkpointer
func NewDownloadCheckpointer() *DownloadCheckpointer {
	cfg := config.Global()
	return &DownloadCheckpointer{
		interval:       cfg.Client.CheckpointInterval,
		lastCheckpoint: make(map[string]int),
	}
}

// ShouldCheckpoint returns true if a checkpoint should be saved
func (c *DownloadCheckpointer) ShouldCheckpoint(cid string, completedPieces int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	last, exists := c.lastCheckpoint[cid]
	if !exists {
		last = 0
	}

	if completedPieces-last >= c.interval {
		c.lastCheckpoint[cid] = completedPieces
		return true
	}
	return false
}

// Reset resets the checkpoint state for a download
func (c *DownloadCheckpointer) Reset(cid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.lastCheckpoint, cid)
}

// SetInterval sets the checkpoint interval
func (c *DownloadCheckpointer) SetInterval(interval int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if interval > 0 {
		c.interval = interval
	}
}
