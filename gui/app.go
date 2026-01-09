package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/1amkhush/torrentium/pkg/config"
	"github.com/1amkhush/torrentium/pkg/db"
	"github.com/1amkhush/torrentium/pkg/logger"
	"github.com/1amkhush/torrentium/pkg/p2p"
	"github.com/1amkhush/torrentium/pkg/torrentium_client"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"

	"github.com/dustin/go-humanize"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ========== Data Types for GUI ==========

// LocalFileInfo represents a shared file for the GUI
type LocalFileInfo struct {
	CID       string `json:"cid"`
	Filename  string `json:"filename"`
	FileSize  int64  `json:"fileSize"`
	FilePath  string `json:"filePath"`
	SizeHuman string `json:"sizeHuman"`
	CreatedAt string `json:"createdAt"`
}

// PeerInfo represents a connected peer for the GUI
type PeerInfo struct {
	PeerID    string   `json:"peerId"`
	Addresses []string `json:"addresses"`
	Connected bool     `json:"connected"`
}

// NetworkStatus represents overall network status
type NetworkStatus struct {
	PeerID           string   `json:"peerId"`
	ListenAddresses  []string `json:"listenAddresses"`
	ConnectedPeers   int      `json:"connectedPeers"`
	DHTRoutingTable  int      `json:"dhtRoutingTable"`
	SharedFilesCount int      `json:"sharedFilesCount"`
	IsConnected      bool     `json:"isConnected"`
}

// StatsData represents combined statistics
type StatsData struct {
	TotalUploaded      int64   `json:"totalUploaded"`
	TotalDownloaded    int64   `json:"totalDownloaded"`
	UploadedHuman      string  `json:"uploadedHuman"`
	DownloadedHuman    string  `json:"downloadedHuman"`
	ChunksServed       int     `json:"chunksServed"`
	PeersServed        int     `json:"peersServed"`
	FilesDownloaded    int     `json:"filesDownloaded"`
	FilesShared        int     `json:"filesShared"`
	Ratio              float64 `json:"ratio"`
	MaxUploadRate      int64   `json:"maxUploadRate"`
	MaxUploadRateHuman string  `json:"maxUploadRateHuman"`
}

// DownloadInfo represents a downloaded file
type DownloadInfo struct {
	CID          string `json:"cid"`
	Filename     string `json:"filename"`
	FileSize     int64  `json:"fileSize"`
	SizeHuman    string `json:"sizeHuman"`
	DownloadPath string `json:"downloadPath"`
	DownloadedAt string `json:"downloadedAt"`
	Status       string `json:"status"`
}

// SearchResult represents a search result
type SearchResult struct {
	CID       string `json:"cid"`
	Filename  string `json:"filename"`
	Providers int    `json:"providers"`
}

// UploadProgressInfo represents upload progress for a file
type UploadProgressInfo struct {
	CID           string `json:"cid"`
	BytesUploaded int64  `json:"bytesUploaded"`
	UploadedHuman string `json:"uploadedHuman"`
	ChunksServed  int    `json:"chunksServed"`
	PeersServed   int    `json:"peersServed"`
	AvgSpeed      string `json:"avgSpeed"`
}

// ConfigData represents configuration for the GUI
type ConfigData struct {
	DownloadDir        string `json:"downloadDir"`
	MaxUploadRate      int64  `json:"maxUploadRate"`
	MaxUploadRateHuman string `json:"maxUploadRateHuman"`
	LogLevel           string `json:"logLevel"`
	DatabasePath       string `json:"databasePath"`
}

// ========== App Struct ==========

// App struct holds all application state
type App struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	client    *torrentium_client.Client
	host      host.Host
	dht       *dht.IpfsDHT
	database  *sql.DB
	repo      *db.Repository
	cfg       *config.Config
	initOnce  sync.Once
	initErr   error
	ready     bool
	readyMu   sync.RWMutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Initialize in background to not block UI
	go a.initializeTorrentium()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.client != nil {
		a.client.Close()
	}
	if a.host != nil {
		a.host.Close()
	}
	if a.database != nil {
		a.database.Close()
	}
	if a.ctxCancel != nil {
		a.ctxCancel()
	}
}

// initializeTorrentium initializes all torrentium components
func (a *App) initializeTorrentium() {
	a.initOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				a.initErr = fmt.Errorf("panic during initialization: %v", r)
				a.readyMu.Lock()
				a.ready = true // Mark as ready so UI shows error
				a.readyMu.Unlock()
				runtime.EventsEmit(a.ctx, "error", a.initErr.Error())
				runtime.EventsEmit(a.ctx, "ready", true)
			}
		}()

		runtime.EventsEmit(a.ctx, "status", "Loading configuration...")

		// Load configuration
		cfg, err := config.LoadOrDefault()
		if err != nil {
			a.initErr = fmt.Errorf("failed to load config: %w", err)
			a.readyMu.Lock()
			a.ready = true
			a.readyMu.Unlock()
			runtime.EventsEmit(a.ctx, "error", a.initErr.Error())
			runtime.EventsEmit(a.ctx, "ready", true)
			return
		}
		a.cfg = cfg
		config.SetGlobal(cfg)

		// Initialize logger (suppress console output in GUI mode)
		logger.Init(logger.Config{
			Level:        "warn",
			Format:       "json",
			Output:       "stdout",
			EnableCaller: false,
		})

		runtime.EventsEmit(a.ctx, "status", "Initializing database...")

		// Initialize database
		a.database, err = db.InitDB(&cfg.Database)
		if err != nil {
			a.initErr = fmt.Errorf("database init failed: %w", err)
			a.readyMu.Lock()
			a.ready = true
			a.readyMu.Unlock()
			runtime.EventsEmit(a.ctx, "error", a.initErr.Error())
			runtime.EventsEmit(a.ctx, "ready", true)
			return
		}

		// Create context for P2P operations
		bgCtx, cancel := context.WithCancel(context.Background())
		a.ctxCancel = cancel

		runtime.EventsEmit(a.ctx, "status", "Connecting to P2P network...")

		// Create P2P host
		a.host, a.dht, err = p2p.NewHost(bgCtx, cfg, nil)
		if err != nil {
			a.initErr = fmt.Errorf("P2P host failed: %w", err)
			a.readyMu.Lock()
			a.ready = true
			a.readyMu.Unlock()
			runtime.EventsEmit(a.ctx, "error", a.initErr.Error())
			runtime.EventsEmit(a.ctx, "ready", true)
			return
		}

		runtime.EventsEmit(a.ctx, "status", "Bootstrapping DHT...")

		// Bootstrap DHT
		go func() {
			if err := p2p.Bootstrap(bgCtx, a.host, a.dht); err != nil {
				runtime.EventsEmit(a.ctx, "warning", "DHT bootstrap issue: "+err.Error())
			}
		}()

		// Create repository and client
		a.repo = db.NewRepository(a.database)
		a.client = torrentium_client.NewClient(a.host, a.dht, a.repo)

		// Register signaling protocol
		p2p.RegisterSignalingProtocol(a.host, a.client.HandleWebRTCOffer)

		// Start DHT maintenance
		a.client.StartDHTMaintenance()

		// Mark as ready
		a.readyMu.Lock()
		a.ready = true
		a.readyMu.Unlock()

		runtime.EventsEmit(a.ctx, "status", "Ready")
		runtime.EventsEmit(a.ctx, "ready", true)

		// Start periodic status updates
		go a.periodicStatusUpdate()
	})
}

// periodicStatusUpdate sends periodic updates to the frontend
func (a *App) periodicStatusUpdate() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.IsReady() {
				status := a.GetNetworkStatus()
				runtime.EventsEmit(a.ctx, "networkUpdate", status)
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// ========== Public API Methods for GUI ==========

// IsReady returns whether the client is initialized
func (a *App) IsReady() bool {
	a.readyMu.RLock()
	defer a.readyMu.RUnlock()
	return a.ready
}

// GetInitError returns any initialization error
func (a *App) GetInitError() string {
	if a.initErr != nil {
		return a.initErr.Error()
	}
	return ""
}

// GetNetworkStatus returns current network status
func (a *App) GetNetworkStatus() NetworkStatus {
	if !a.IsReady() {
		return NetworkStatus{IsConnected: false}
	}

	addrs := a.host.Addrs()
	listenAddrs := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		listenAddrs = append(listenAddrs, addr.String())
	}

	peers := a.host.Network().Peers()
	sharedFiles, _ := a.repo.GetLocalFiles(context.Background())

	return NetworkStatus{
		PeerID:           a.host.ID().String(),
		ListenAddresses:  listenAddrs,
		ConnectedPeers:   len(peers),
		DHTRoutingTable:  a.dht.RoutingTable().Size(),
		SharedFilesCount: len(sharedFiles),
		IsConnected:      len(peers) > 0,
	}
}

// GetLocalFiles returns all shared files
func (a *App) GetLocalFiles() ([]LocalFileInfo, error) {
	if !a.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}

	files, err := a.repo.GetLocalFiles(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]LocalFileInfo, 0, len(files))
	for _, f := range files {
		result = append(result, LocalFileInfo{
			CID:       f.CID,
			Filename:  f.Filename,
			FileSize:  f.FileSize,
			FilePath:  f.FilePath,
			SizeHuman: humanize.Bytes(uint64(f.FileSize)),
			CreatedAt: f.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	return result, nil
}

// GetConnectedPeers returns list of connected peers
func (a *App) GetConnectedPeers() []PeerInfo {
	if !a.IsReady() {
		return nil
	}

	peers := a.host.Network().Peers()
	result := make([]PeerInfo, 0, len(peers))

	for _, peerID := range peers {
		conns := a.host.Network().ConnsToPeer(peerID)
		addrs := make([]string, 0)
		for _, conn := range conns {
			addrs = append(addrs, conn.RemoteMultiaddr().String())
		}

		result = append(result, PeerInfo{
			PeerID:    peerID.String(),
			Addresses: addrs,
			Connected: a.host.Network().Connectedness(peerID) == network.Connected,
		})
	}
	return result
}

// GetStats returns combined upload/download statistics
func (a *App) GetStats() StatsData {
	if !a.IsReady() {
		return StatsData{}
	}

	ctx := context.Background()

	// Upload stats from DB
	totalUpBytes, totalUpChunks, upPeers, _ := a.repo.GetTotalUploadStats(ctx)

	// Real-time upload stats
	rtStats := a.client.GetUploadStats()
	for _, s := range rtStats {
		totalUpBytes += s.BytesUploaded
		totalUpChunks += s.ChunksServed
	}

	// Download stats
	downloads, _ := a.repo.GetDownloads(ctx)
	var totalDownBytes int64
	for _, d := range downloads {
		totalDownBytes += d.FileSize
	}

	// Shared files
	localFiles, _ := a.repo.GetLocalFiles(ctx)

	// Ratio
	var ratio float64
	if totalDownBytes > 0 {
		ratio = float64(totalUpBytes) / float64(totalDownBytes)
	}

	maxRate := a.client.GetMaxUploadRate()
	maxRateHuman := "Unlimited"
	if maxRate > 0 {
		maxRateHuman = humanize.Bytes(uint64(maxRate)) + "/s"
	}

	return StatsData{
		TotalUploaded:      totalUpBytes,
		TotalDownloaded:    totalDownBytes,
		UploadedHuman:      humanize.Bytes(uint64(totalUpBytes)),
		DownloadedHuman:    humanize.Bytes(uint64(totalDownBytes)),
		ChunksServed:       totalUpChunks,
		PeersServed:        upPeers,
		FilesDownloaded:    len(downloads),
		FilesShared:        len(localFiles),
		Ratio:              ratio,
		MaxUploadRate:      maxRate,
		MaxUploadRateHuman: maxRateHuman,
	}
}

// GetDownloads returns all downloaded files
func (a *App) GetDownloads() ([]DownloadInfo, error) {
	if !a.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}

	downloads, err := a.repo.GetDownloads(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]DownloadInfo, 0, len(downloads))
	for _, d := range downloads {
		result = append(result, DownloadInfo{
			CID:          d.CID,
			Filename:     d.Filename,
			FileSize:     d.FileSize,
			SizeHuman:    humanize.Bytes(uint64(d.FileSize)),
			DownloadPath: d.DownloadPath,
			DownloadedAt: d.DownloadedAt.Format("2006-01-02 15:04"),
			Status:       d.Status,
		})
	}
	return result, nil
}

// GetUploadProgress returns current upload/seeding progress
func (a *App) GetUploadProgress() []UploadProgressInfo {
	if !a.IsReady() {
		return nil
	}

	stats := a.client.GetUploadStats()
	result := make([]UploadProgressInfo, 0, len(stats))

	for _, s := range stats {
		avgSpeed := "N/A"
		if !s.LastActivity.IsZero() && !s.StartTime.IsZero() {
			duration := s.LastActivity.Sub(s.StartTime)
			if duration > 0 {
				speed := float64(s.BytesUploaded) / duration.Seconds()
				avgSpeed = humanize.Bytes(uint64(speed)) + "/s"
			}
		}

		result = append(result, UploadProgressInfo{
			CID:           s.CID,
			BytesUploaded: s.BytesUploaded,
			UploadedHuman: humanize.Bytes(uint64(s.BytesUploaded)),
			ChunksServed:  s.ChunksServed,
			PeersServed:   len(s.PeersServed),
			AvgSpeed:      avgSpeed,
		})
	}
	return result
}

// GetConfig returns current configuration
func (a *App) GetConfig() ConfigData {
	cfg := config.Global()
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	maxRate := int64(0)
	maxRateHuman := "Unlimited"
	if a.client != nil {
		maxRate = a.client.GetMaxUploadRate()
		if maxRate > 0 {
			maxRateHuman = humanize.Bytes(uint64(maxRate)) + "/s"
		}
	}

	return ConfigData{
		DownloadDir:        cfg.Client.DownloadDirectory,
		MaxUploadRate:      maxRate,
		MaxUploadRateHuman: maxRateHuman,
		LogLevel:           cfg.Logging.Level,
		DatabasePath:       cfg.Database.Path,
	}
}

// ========== Action Methods ==========

// AddFile adds a file or directory to share
func (a *App) AddFile(filePath string) (string, error) {
	if !a.IsReady() {
		return "", fmt.Errorf("client not ready")
	}

	err := a.client.AddFile(filePath)
	if err != nil {
		return "", err
	}

	// Get the CID of the added file
	files, _ := a.repo.GetLocalFiles(context.Background())
	for _, f := range files {
		if f.FilePath == filePath {
			runtime.EventsEmit(a.ctx, "fileAdded", LocalFileInfo{
				CID:       f.CID,
				Filename:  f.Filename,
				FileSize:  f.FileSize,
				FilePath:  f.FilePath,
				SizeHuman: humanize.Bytes(uint64(f.FileSize)),
			})
			return f.CID, nil
		}
	}
	return "", nil
}

// SelectFile opens a file dialog and returns the selected path
func (a *App) SelectFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File to Share",
	})
	return selection, err
}

// SelectDirectory opens a directory dialog and returns the selected path
func (a *App) SelectDirectory() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Directory to Share",
	})
	return selection, err
}

// SelectDownloadDirectory opens a dialog to select download directory
func (a *App) SelectDownloadDirectory() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Download Directory",
	})
	return selection, err
}

// DownloadFile downloads a file by CID
func (a *App) DownloadFile(cidStr string) error {
	if !a.IsReady() {
		return fmt.Errorf("client not ready")
	}

	runtime.EventsEmit(a.ctx, "downloadStarted", cidStr)

	go func() {
		err := a.client.DownloadFile(cidStr)
		if err != nil {
			runtime.EventsEmit(a.ctx, "downloadError", map[string]string{
				"cid":   cidStr,
				"error": err.Error(),
			})
		} else {
			runtime.EventsEmit(a.ctx, "downloadComplete", cidStr)
		}
	}()

	return nil
}

// SearchByCID searches for providers of a CID
func (a *App) SearchByCID(cidStr string) (int, error) {
	if !a.IsReady() {
		return 0, fmt.Errorf("client not ready")
	}

	// Use the enhanced search - but capture results instead of printing
	err := a.client.EnhancedSearchByCID(cidStr)
	if err != nil {
		return 0, err
	}
	// Note: The original method prints to stdout. For GUI, we return success.
	return 1, nil
}

// SearchByText searches for files by text in local index
func (a *App) SearchByText(query string) ([]SearchResult, error) {
	if !a.IsReady() {
		return nil, fmt.Errorf("client not ready")
	}

	matches, err := a.repo.SearchByFilename(context.Background(), query)
	if err != nil {
		return nil, err
	}

	result := make([]SearchResult, 0, len(matches))
	for _, m := range matches {
		result = append(result, SearchResult{
			CID:      m.CID,
			Filename: m.Filename,
		})
	}
	return result, nil
}

// AnnounceFile re-announces a file to the DHT
func (a *App) AnnounceFile(cidStr string) error {
	if !a.IsReady() {
		return fmt.Errorf("client not ready")
	}
	return a.client.AnnounceFile(cidStr)
}

// SetMaxUploadRate sets the maximum upload rate
func (a *App) SetMaxUploadRate(rate int64) {
	if a.client != nil {
		a.client.SetMaxUploadRate(rate)
		runtime.EventsEmit(a.ctx, "configUpdated", a.GetConfig())
	}
}

// SetDownloadDirectory sets the download directory
func (a *App) SetDownloadDirectory(dir string) error {
	// Validate directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	cfg := config.Global()
	if cfg != nil {
		cfg.Client.DownloadDirectory = dir
	}
	runtime.EventsEmit(a.ctx, "configUpdated", a.GetConfig())
	return nil
}

// CopyToClipboard copies text to clipboard
func (a *App) CopyToClipboard(text string) error {
	runtime.ClipboardSetText(a.ctx, text)
	return nil
}

// OpenFileLocation opens the file location in file explorer
func (a *App) OpenFileLocation(filePath string) error {
	dir := filepath.Dir(filePath)
	runtime.BrowserOpenURL(a.ctx, "file:///"+dir)
	return nil
}

// GetPeerID returns the local peer ID
func (a *App) GetPeerID() string {
	if !a.IsReady() {
		return ""
	}
	return a.host.ID().String()
}

// RefreshDHT triggers a DHT refresh
func (a *App) RefreshDHT() error {
	if !a.IsReady() {
		return fmt.Errorf("client not ready")
	}
	return a.dht.Bootstrap(context.Background())
}
