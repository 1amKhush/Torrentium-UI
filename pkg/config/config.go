// Package config provides centralized configuration management for Torrentium.
// It supports loading configuration from YAML files, environment variables,
// and provides sensible defaults for production use.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration options for Torrentium
type Config struct {
	// P2P configuration
	P2P P2PConfig `yaml:"p2p"`

	// WebRTC configuration
	WebRTC WebRTCConfig `yaml:"webrtc"`

	// Database configuration
	Database DatabaseConfig `yaml:"database"`

	// Client configuration
	Client ClientConfig `yaml:"client"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`

	// WebShare configuration for the web share portal
	WebShare WebShareConfig `yaml:"webshare"`
}

// WebShareConfig holds configuration for the web share portal
type WebShareConfig struct {
	// PortalURL is the URL of the web share portal server
	PortalURL string `yaml:"portal_url"`

	// APIKey is the optional API key for publishing to the portal
	APIKey string `yaml:"api_key"`

	// DefaultVisibility is the default visibility for published files (public/unlisted)
	DefaultVisibility string `yaml:"default_visibility"`

	// DefaultExpiration is the default expiration time in hours (0 = never)
	DefaultExpiration int `yaml:"default_expiration"`
}

// P2PConfig holds P2P network configuration
type P2PConfig struct {
	// ListenAddress is the multiaddr to listen on (default: /ip4/0.0.0.0/tcp/0)
	ListenAddress string `yaml:"listen_address"`

	// PrivateKeyPath is the path to the private key file (default: private_key)
	PrivateKeyPath string `yaml:"private_key_path"`

	// RelayHTTPURL is the HTTP URL to fetch relay info (peer ID) dynamically
	RelayHTTPURL string `yaml:"relay_http_url"`

	// RelayAddresses are the circuit relay servers to use for NAT traversal
	// If RelayHTTPURL is set, these are fetched dynamically and cached
	RelayAddresses []string `yaml:"relay_addresses"`

	// BootstrapNodes are the initial nodes to connect to for DHT bootstrap
	BootstrapNodes []string `yaml:"bootstrap_nodes"`

	// MinBootstrapConnections is the minimum number of bootstrap connections required
	MinBootstrapConnections int `yaml:"min_bootstrap_connections"`

	// BootstrapTimeout is the timeout for connecting to bootstrap nodes
	BootstrapTimeout time.Duration `yaml:"bootstrap_timeout"`

	// DHTRefreshInterval is how often to refresh the DHT routing table
	DHTRefreshInterval time.Duration `yaml:"dht_refresh_interval"`

	// MinPeerCount is the minimum number of peers before re-bootstrapping
	MinPeerCount int `yaml:"min_peer_count"`
}

// WebRTCConfig holds WebRTC connection configuration
type WebRTCConfig struct {
	// STUNServers are the STUN servers for ICE candidate gathering
	STUNServers []string `yaml:"stun_servers"`

	// TURNServers are optional TURN servers for relay (if STUN fails)
	TURNServers []TURNServer `yaml:"turn_servers,omitempty"`

	// ICEGatheringTimeout is the timeout for ICE candidate gathering
	ICEGatheringTimeout time.Duration `yaml:"ice_gathering_timeout"`

	// ConnectionTimeout is the timeout for establishing WebRTC connections
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`

	// KeepAliveInterval is the interval for sending keep-alive messages
	KeepAliveInterval time.Duration `yaml:"keep_alive_interval"`
}

// TURNServer represents a TURN server configuration
type TURNServer struct {
	URLs       []string `yaml:"urls"`
	Username   string   `yaml:"username,omitempty"`
	Credential string   `yaml:"credential,omitempty"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Path is the path to the SQLite database file
	Path string `yaml:"path"`

	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int `yaml:"max_open_conns"`

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int `yaml:"max_idle_conns"`

	// ConnMaxLifetime is the maximum lifetime of a connection
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`

	// ConnMaxIdleTime is the maximum idle time of a connection
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
}

// ClientConfig holds client-specific configuration
type ClientConfig struct {
	// DefaultPieceSize is the default size of file pieces (default: 1MiB)
	DefaultPieceSize int64 `yaml:"default_piece_size"`

	// MaxProviders is the maximum number of providers to query from DHT
	MaxProviders int `yaml:"max_providers"`

	// MaxChunkSize is the maximum size of data chunks sent over WebRTC
	MaxChunkSize int `yaml:"max_chunk_size"`

	// MaxParallelDownloads is the maximum number of parallel piece downloads
	MaxParallelDownloads int `yaml:"max_parallel_downloads"`

	// MinParallelDownloads is the minimum number of parallel downloads for adaptive mode
	MinParallelDownloads int `yaml:"min_parallel_downloads"`

	// AdaptiveParallelDownloads enables automatic adjustment based on bandwidth
	AdaptiveParallelDownloads bool `yaml:"adaptive_parallel_downloads"`

	// PieceTimeout is the timeout for downloading a single piece
	PieceTimeout time.Duration `yaml:"piece_timeout"`

	// RetransmissionTimeout is the timeout before retransmitting unacked chunks
	RetransmissionTimeout time.Duration `yaml:"retransmission_timeout"`

	// MaxUploadRate is the maximum upload rate in bytes/sec (0 = unlimited)
	MaxUploadRate int64 `yaml:"max_upload_rate"`

	// MaxDownloadRate is the maximum download rate in bytes/sec (0 = unlimited)
	MaxDownloadRate int64 `yaml:"max_download_rate"`

	// DownloadDirectory is the default directory for downloads
	DownloadDirectory string `yaml:"download_directory"`

	// CheckpointInterval is how often to save download state (number of pieces)
	CheckpointInterval int `yaml:"checkpoint_interval"`

	// EnableEndgameMode enables requesting last pieces from multiple peers
	EnableEndgameMode bool `yaml:"enable_endgame_mode"`

	// EndgameThreshold is the percentage of pieces remaining to trigger endgame
	EndgameThreshold float64 `yaml:"endgame_threshold"`

	// MaxConnectionPoolSize is the maximum number of pooled WebRTC connections
	MaxConnectionPoolSize int `yaml:"max_connection_pool_size"`

	// ConnectionPoolIdleTimeout is how long idle connections stay in pool
	ConnectionPoolIdleTimeout time.Duration `yaml:"connection_pool_idle_timeout"`

	// DHTRetryAttempts is the number of retry attempts for DHT operations
	DHTRetryAttempts int `yaml:"dht_retry_attempts"`

	// DHTRetryBackoffBase is the base duration for exponential backoff
	DHTRetryBackoffBase time.Duration `yaml:"dht_retry_backoff_base"`

	// DHTRetryBackoffMax is the maximum backoff duration
	DHTRetryBackoffMax time.Duration `yaml:"dht_retry_backoff_max"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `yaml:"level"`

	// Format is the log format (json, console)
	Format string `yaml:"format"`

	// Output is the log output (stdout, stderr, or file path)
	Output string `yaml:"output"`

	// EnableCaller enables caller information in logs
	EnableCaller bool `yaml:"enable_caller"`
}

var (
	globalConfig *Config
	configMu     sync.RWMutex
	configOnce   sync.Once
)

// DefaultConfig returns a Config with sensible production defaults
func DefaultConfig() *Config {
	return &Config{
		P2P: P2PConfig{
			ListenAddress:  "/ip4/0.0.0.0/tcp/0",
			PrivateKeyPath: "private_key",
			RelayHTTPURL:   "",
			RelayAddresses: []string{
				"/dns4/relay-torrentium-pj9h.onrender.com/tcp/443/wss/p2p/12D3KooWQmD64vYYegz3GVDHtt4KeqkSuBSYoLwGMircrv1Q1TdW",
			},
			BootstrapNodes: []string{
				"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
				"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
				"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zp7VCk8JpNUQLoUPF3HfrDAQGS52a8",
				"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX89HWoNT4gEoNA7MzZqaGzyCu5w",
				"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
				"/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
				"/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
				"/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
			},
			MinBootstrapConnections: 5,
			BootstrapTimeout:        15 * time.Second,
			DHTRefreshInterval:      10 * time.Minute,
			MinPeerCount:            5,
		},
		WebRTC: WebRTCConfig{
			STUNServers: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
				"stun:stun.cloudflare.com:3478",
			},
			TURNServers:         nil,
			ICEGatheringTimeout: 15 * time.Second,
			ConnectionTimeout:   30 * time.Second,
			KeepAliveInterval:   15 * time.Second,
		},
		Database: DatabaseConfig{
			Path:            "./peer.db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		},
		Client: ClientConfig{
			DefaultPieceSize:          1 << 20, // 1 MiB
			MaxProviders:              10,
			MaxChunkSize:              16 * 1024, // 16 KiB
			MaxParallelDownloads:      3,
			MinParallelDownloads:      1,
			AdaptiveParallelDownloads: true,
			PieceTimeout:              300 * time.Second,
			RetransmissionTimeout:     5 * time.Second,
			MaxUploadRate:             0,
			MaxDownloadRate:           0,
			DownloadDirectory:         ".",
			CheckpointInterval:        5, // Save state every 5 pieces
			EnableEndgameMode:         true,
			EndgameThreshold:          0.05, // Last 5% of pieces
			MaxConnectionPoolSize:     20,
			ConnectionPoolIdleTimeout: 5 * time.Minute,
			DHTRetryAttempts:          3,
			DHTRetryBackoffBase:       1 * time.Second,
			DHTRetryBackoffMax:        30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:        "info",
			Format:       "console",
			Output:       "stdout",
			EnableCaller: false,
		},
		WebShare: WebShareConfig{
			PortalURL:         "https://share.torrentium.io",
			APIKey:            "",
			DefaultVisibility: "public",
			DefaultExpiration: 0,
		},
	}
}

// Load reads configuration from a YAML file, merging with defaults
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Check for config file
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	cfg.applyEnvOverrides()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// LoadOrDefault loads configuration from the default paths or returns defaults
func LoadOrDefault() (*Config, error) {
	// Check common config file locations
	configPaths := []string{
		"config.yaml",
		"config.yml",
		"torrentium.yaml",
		"torrentium.yml",
		filepath.Join(os.Getenv("HOME"), ".torrentium", "config.yaml"),
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}

	// No config file found, use defaults with env overrides
	cfg := DefaultConfig()
	cfg.applyEnvOverrides()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func (c *Config) applyEnvOverrides() {
	// Database
	if v := os.Getenv("TORRENTIUM_DB_PATH"); v != "" {
		c.Database.Path = v
	}
	if v := os.Getenv("SQLITE_DB_PATH"); v != "" { // backward compatibility
		c.Database.Path = v
	}

	// P2P
	if v := os.Getenv("TORRENTIUM_LISTEN_ADDRESS"); v != "" {
		c.P2P.ListenAddress = v
	}
	if v := os.Getenv("TORRENTIUM_PRIVATE_KEY_PATH"); v != "" {
		c.P2P.PrivateKeyPath = v
	}
	if v := os.Getenv("TORRENTIUM_RELAY_ADDRESS"); v != "" {
		c.P2P.RelayAddresses = []string{v}
	}

	// Logging
	if v := os.Getenv("TORRENTIUM_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("TORRENTIUM_LOG_FORMAT"); v != "" {
		c.Logging.Format = v
	}

	// Client
	if v := os.Getenv("TORRENTIUM_DOWNLOAD_DIR"); v != "" {
		c.Client.DownloadDirectory = v
	}
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if c.P2P.ListenAddress == "" {
		return fmt.Errorf("p2p.listen_address is required")
	}
	if len(c.P2P.RelayAddresses) == 0 {
		return fmt.Errorf("at least one relay address is required")
	}
	if len(c.WebRTC.STUNServers) == 0 {
		return fmt.Errorf("at least one STUN server is required")
	}
	if c.Client.DefaultPieceSize <= 0 {
		return fmt.Errorf("client.default_piece_size must be positive")
	}
	if c.Client.MaxChunkSize <= 0 {
		return fmt.Errorf("client.max_chunk_size must be positive")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}
	return nil
}

// Global returns the global configuration instance
func Global() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	if globalConfig == nil {
		return DefaultConfig()
	}
	return globalConfig
}

// SetGlobal sets the global configuration instance
func SetGlobal(cfg *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	globalConfig = cfg
}

// InitGlobal initializes the global configuration once
func InitGlobal(path string) error {
	var initErr error
	configOnce.Do(func() {
		var cfg *Config
		if path != "" {
			cfg, initErr = Load(path)
		} else {
			cfg, initErr = LoadOrDefault()
		}
		if initErr == nil {
			SetGlobal(cfg)
		}
	})
	return initErr
}

// SaveDefault creates a default configuration file at the given path
func SaveDefault(path string) error {
	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
