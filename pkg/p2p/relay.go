// Package p2p provides peer-to-peer networking functionality for Torrentium.
package p2p

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/1amkhush/torrentium/pkg/config"
	"github.com/1amkhush/torrentium/pkg/logger"
)

// RelayInfo represents the information returned by the relay's /info endpoint
type RelayInfo struct {
	PeerID     string   `json:"peer_id"`
	Multiaddrs []string `json:"multiaddrs"`
}

// RelayCache stores cached relay information
type RelayCache struct {
	Info      RelayInfo `json:"info"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	relayCache     *RelayCache
	relayCachePath string
)

func init() {
	// Set up cache path in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	relayCachePath = filepath.Join(homeDir, ".torrentium", "relay_cache.json")
}

// GetRelayAddresses fetches relay addresses, using cache as fallback
func GetRelayAddresses() ([]string, error) {
	cfg := config.Global()

	// If relay addresses are explicitly configured, use them
	if len(cfg.P2P.RelayAddresses) > 0 {
		return cfg.P2P.RelayAddresses, nil
	}

	// If no HTTP URL configured, return empty
	if cfg.P2P.RelayHTTPURL == "" {
		return nil, fmt.Errorf("no relay configuration: set RelayHTTPURL or RelayAddresses")
	}

	// Try to fetch from HTTP endpoint
	info, err := fetchRelayInfo(cfg.P2P.RelayHTTPURL)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to fetch relay info from HTTP, trying cache")

		// Try to load from cache
		cached, cacheErr := loadRelayCache()
		if cacheErr != nil {
			return nil, fmt.Errorf("failed to fetch relay info and no cache available: %w", err)
		}

		// Use cached info
		logger.Info().Str("peer_id", cached.Info.PeerID).Msg("Using cached relay info")
		return cached.Info.Multiaddrs, nil
	}

	// Save to cache for future use
	if err := saveRelayCache(info); err != nil {
		logger.Warn().Err(err).Msg("Failed to save relay cache")
	}

	logger.Info().Str("peer_id", info.PeerID).Msg("Fetched relay info")
	return info.Multiaddrs, nil
}

// fetchRelayInfo fetches relay information from the HTTP endpoint
func fetchRelayInfo(url string) (*RelayInfo, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var info RelayInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if info.PeerID == "" {
		return nil, fmt.Errorf("invalid relay info: missing peer_id")
	}

	return &info, nil
}

// loadRelayCache loads cached relay information from disk
func loadRelayCache() (*RelayCache, error) {
	data, err := os.ReadFile(relayCachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache RelayCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Check if cache is too old (more than 7 days)
	if time.Since(cache.UpdatedAt) > 7*24*time.Hour {
		return nil, fmt.Errorf("cache is too old")
	}

	return &cache, nil
}

// saveRelayCache saves relay information to disk cache
func saveRelayCache(info *RelayInfo) error {
	cache := RelayCache{
		Info:      *info,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(relayCachePath), 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(relayCachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// RefreshRelayInfo forces a refresh of relay information
func RefreshRelayInfo() error {
	cfg := config.Global()

	if cfg.P2P.RelayHTTPURL == "" {
		return fmt.Errorf("no relay HTTP URL configured")
	}

	info, err := fetchRelayInfo(cfg.P2P.RelayHTTPURL)
	if err != nil {
		return err
	}

	if err := saveRelayCache(info); err != nil {
		logger.Warn().Err(err).Msg("Failed to save relay cache")
	}

	logger.Info().Str("peer_id", info.PeerID).Msg("Refreshed relay info")
	return nil
}
