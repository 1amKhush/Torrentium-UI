package torrentium_client

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	webRTC "github.com/1amkhush/torrentium/pkg/client"
	"github.com/1amkhush/torrentium/pkg/config"
	db "github.com/1amkhush/torrentium/pkg/db"
	"github.com/1amkhush/torrentium/pkg/logger"

	"github.com/dustin/go-humanize"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pion/webrtc/v3"
	"github.com/schollz/progressbar/v3"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

const (
	// SignalingProtocolID is the protocol ID for WebRTC signaling
	SignalingProtocolID = "/torrentium/webrtc-signaling/1.0"

	// DefaultPieceSize is the default size of file pieces (1 MiB)
	// Can be overridden via config
	DefaultPieceSize = 1 << 20

	// MaxProviders is the maximum number of providers to query from DHT
	MaxProviders = 10

	// MaxChunk is the maximum size of data chunks sent over WebRTC (16 KiB)
	MaxChunk = 16 * 1024

	// MaxParallelDownloads is the maximum number of parallel piece downloads
	MaxParallelDownloads = 3

	// PieceTimeout is the timeout for downloading a single piece
	PieceTimeout = 300 * time.Second

	// RetransmissionTimeout is the timeout before retransmitting unacked chunks
	RetransmissionTimeout = 5 * time.Second

	// KeepAliveInterval is the interval for sending keep-alive messages
	KeepAliveInterval = 15 * time.Second

	// PingInterval is the interval for sending ping messages
	PingInterval = 10 * time.Second

	// MaxRTT is the maximum round-trip time
	MaxRTT = 500 * time.Millisecond

	// MinDelay is the minimum delay
	MinDelay = 0

	// MaxDelay is the maximum delay
	MaxDelay = 100 * time.Millisecond

	// ExponentialBackoffBase is the base for exponential backoff
	ExponentialBackoffBase = 1 * time.Second

	// MaxBackoff is the maximum backoff duration
	MaxBackoff = 32 * time.Second

	// MaxUploadRate is the default max upload rate (0 = unlimited)
	MaxUploadRate = 0
)

// getConfig returns the configuration values from global config or defaults
func getConfig() *config.Config {
	cfg := config.Global()
	if cfg == nil {
		return config.DefaultConfig()
	}
	return cfg
}

// PieceAvailability tracks which peers have a specific piece
type PieceAvailability struct {
	PieceIndex  int
	PeerCount   int       // How many peers have this piece
	HavingPeers []peer.ID // Which peers have it
}

// UploadStats tracks real-time upload progress for a CID
type UploadStats struct {
	CID           string
	BytesUploaded int64
	ChunksServed  int
	PeersServed   map[peer.ID]int64 // bytes per peer
	StartTime     time.Time
	LastActivity  time.Time
	mu            sync.Mutex
}

type Client struct {
	host             host.Host
	dht              *dht.IpfsDHT
	webRTCPeers      map[peer.ID]*webRTC.SimpleWebRTCPeer
	peersMux         sync.RWMutex
	sharingFiles     map[string]*FileInfo
	activeDownloads  map[string]*DownloadState
	downloadsMux     sync.RWMutex
	db               *db.Repository
	unackedChunks    map[string]map[int64]map[int]controlMessage
	unackedChunksMux sync.RWMutex
	congestionCtrl   map[peer.ID]time.Duration
	pingTimes        map[peer.ID]time.Time
	rttMeasurements  map[peer.ID][]time.Duration
	rttMux           sync.Mutex
	// Upload progress tracking
	uploadProgress   map[string]*UploadStats
	uploadProgressMu sync.RWMutex
	maxUploadRate    int64 // bytes per second, 0 = unlimited
	// Download queue management
	downloadQueue      *DownloadQueue
	adaptiveManager    *AdaptiveDownloadManager
	endgameManager     *EndgameManager
	checkpointer       *DownloadCheckpointer
	dhtRetryHelper     *DHTRetryHelper
}

type FileInfo struct {
	FilePath    string
	Hash        string
	Size        int64
	Name        string
	PieceSz     int64
	IsDirectory bool
	Files       []db.FileEntry // For multi-file support
}

type controlMessage struct {
	Command     string     `json:"command"`
	CID         string     `json:"cid,omitempty"`
	PieceSize   int64      `json:"piece_size,omitempty"`
	TotalSize   int64      `json:"total_size,omitempty"`
	HashHex     string     `json:"hash_hex,omitempty"`
	NumPieces   int64      `json:"num_pieces,omitempty"`
	Pieces      []db.Piece `json:"pieces,omitempty"`
	PieceHash   string     `json:"piece_hash,omitempty"`
	Index       int64      `json:"index,omitempty"`
	Filename    string     `json:"filename,omitempty"`
	ChunkIndex  int        `json:"chunk_index,omitempty"`
	TotalChunks int        `json:"total_chunks,omitempty"`
	Payload     string     `json:"payload,omitempty"`
	Sequence    int        `json:"sequence,omitempty"`
	// Multi-file support
	Files       []db.FileEntry `json:"files,omitempty"`
	IsDirectory bool           `json:"is_directory,omitempty"`
	// Bitfield protocol for rarest-first
	Bitfield  []bool `json:"bitfield,omitempty"`   // Which pieces this peer has
	HavePiece int64  `json:"have_piece,omitempty"` // Single piece announcement
}

type DownloadState struct {
	File            *os.File
	Manifest        controlMessage
	TotalPieces     int
	Pieces          []db.Piece
	Completed       chan bool
	Progress        *progressbar.ProgressBar
	PieceStatus     []bool // true if piece is downloaded
	PieceAssignees  map[int]peer.ID
	pieceBuffers    map[int][][]byte // Buffer to reassemble chunks into pieces
	mu              sync.Mutex
	completedPieces int
	pieceTimers     map[int]*time.Timer // Timers for each piece
	retryCounts     map[int]int         // Retry counts for exponential backoff
	// Rarest-first piece selection
	pieceRarity   map[int]*PieceAvailability // Tracks availability of each piece
	peerBitfields map[peer.ID][]bool         // Bitfields received from peers
	// Multi-file support
	Files         []db.FileEntry // Files to download (for selective download)
	SelectedFiles []int          // Indices of files selected for download (-1 = all)
}

var (
	manifestWaiters = make(map[string]chan controlMessage)
	manifestChMu    sync.Mutex
)

// NewClient creates a new Torrentium client instance
func NewClient(h host.Host, d *dht.IpfsDHT, repo *db.Repository) *Client {
	cfg := getConfig()
	c := &Client{
		host:            h,
		dht:             d,
		webRTCPeers:     make(map[peer.ID]*webRTC.SimpleWebRTCPeer),
		sharingFiles:    make(map[string]*FileInfo),
		activeDownloads: make(map[string]*DownloadState),
		db:              repo,
		unackedChunks:   make(map[string]map[int64]map[int]controlMessage),
		congestionCtrl:  make(map[peer.ID]time.Duration),
		pingTimes:       make(map[peer.ID]time.Time),
		rttMeasurements: make(map[peer.ID][]time.Duration),
		uploadProgress:  make(map[string]*UploadStats),
		maxUploadRate:   cfg.Client.MaxUploadRate,
	}

	// Initialize download queue with configured max parallel downloads
	c.downloadQueue = NewDownloadQueue(cfg.Client.MaxParallelDownloads)
	c.downloadQueue.SetCallbacks(
		func(cid string, status DownloadStatus) {
			log := logger.WithComponent("download_queue")
			log.Info().Str("cid", cid).Str("status", string(status)).Msg("Download status changed")
		},
		nil, nil, nil,
	)

	// Initialize adaptive download manager (uses config internally)
	c.adaptiveManager = NewAdaptiveDownloadManager()

	// Initialize endgame manager (uses config internally)
	c.endgameManager = NewEndgameManager()

	// Initialize DHT retry helper (uses config internally)
	c.dhtRetryHelper = NewDHTRetryHelper()

	// Initialize download checkpointer (uses config internally)
	c.checkpointer = NewDownloadCheckpointer()

	go c.monitorCongestion()
	return c
}

// GetDownloadQueue returns the download queue for management
func (c *Client) GetDownloadQueue() *DownloadQueue {
	return c.downloadQueue
}

// SetAdaptiveDownloadsEnabled enables or disables adaptive parallel downloads
func (c *Client) SetAdaptiveDownloadsEnabled(enabled bool) {
	c.adaptiveManager.SetEnabled(enabled)
}

// SetEndgameModeEnabled enables or disables endgame mode
func (c *Client) SetEndgameModeEnabled(enabled bool) {
	c.endgameManager.SetEnabled(enabled)
}

// SetMaxDownloadRate sets the maximum download rate in bytes per second
func (c *Client) SetMaxDownloadRate(rate int64) {
	c.adaptiveManager.SetMaxBandwidth(rate)
}

// SetMaxUploadRate sets the maximum upload rate in bytes per second.
// A value of 0 means unlimited. This method is thread-safe.
func (c *Client) SetMaxUploadRate(rate int64) {
	c.uploadProgressMu.Lock()
	defer c.uploadProgressMu.Unlock()
	c.maxUploadRate = rate
	if rate == 0 {
		fmt.Println("Upload rate limit removed (unlimited)")
	} else {
		fmt.Printf("Max upload rate set to %s/s\n", humanize.Bytes(uint64(rate)))
	}
}

// GetMaxUploadRate returns the current maximum upload rate in bytes per second
func (c *Client) GetMaxUploadRate() int64 {
	c.uploadProgressMu.RLock()
	defer c.uploadProgressMu.RUnlock()
	return c.maxUploadRate
}

// Close gracefully shuts down the client, closing all connections and resources
func (c *Client) Close() error {
	log := logger.WithComponent("client")
	log.Info().Msg("Shutting down client...")

	// Stop adaptive download manager
	if c.adaptiveManager != nil {
		c.adaptiveManager.Stop()
	}

	// Clear download queue
	if c.downloadQueue != nil {
		c.downloadQueue.Clear()
	}

	// Close all WebRTC peer connections
	c.peersMux.Lock()
	for peerID, peer := range c.webRTCPeers {
		log.Debug().Str("peer_id", peerID.String()).Msg("Closing WebRTC connection")
		peer.Close()
	}
	c.webRTCPeers = make(map[peer.ID]*webRTC.SimpleWebRTCPeer)
	c.peersMux.Unlock()

	// Cancel active downloads
	c.downloadsMux.Lock()
	for cid, state := range c.activeDownloads {
		log.Debug().Str("cid", cid).Msg("Canceling active download")
		if state.File != nil {
			state.File.Close()
		}
		if state.Completed != nil {
			close(state.Completed)
		}
	}
	c.activeDownloads = make(map[string]*DownloadState)
	c.downloadsMux.Unlock()

	log.Info().Msg("Client shutdown complete")
	return nil
}

func (c *Client) commandLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	c.PrintInstructions()
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}
		cmd, args := parts[0], parts[1:]
		var err error
		switch cmd {
		case "help":
			c.PrintInstructions()
		case "add":
			if len(args) != 1 {
				fmt.Println("Usage: add <path>")
			} else {
				err = c.AddFile(args[0])
			}
		case "list":
			c.ListLocalFiles()
		case "search":
			if len(args) != 1 {
				fmt.Println("Usage: search <cid|text>")
			} else {
				c.CheckConnectionHealth()
				if strings.HasPrefix(args[0], "bafy") || strings.HasPrefix(args[0], "Qm") {
					err = c.EnhancedSearchByCID(args[0])
				} else {
					err = c.SearchByText(args[0])
				}
			}
		case "download":
			if len(args) != 1 {
				fmt.Println("Usage: download <cid>")
			} else {
				err = c.DownloadFile(args[0])
			}
		case "peers":
			c.ListConnectedPeers()
		case "announce":
			if len(args) != 1 {
				fmt.Println("Usage: announce <cid>")
			} else {
				err = c.AnnounceFile(args[0])
			}
		case "health":
			c.CheckConnectionHealth()
		case "debug":
			c.DebugNetworkStatus()
		case "exit":
			return
		default:
			fmt.Println("Unknown command. Type 'help' for available commands.")
		}
		if err != nil {
			logger.Error().Err(err).Msg("Command failed")
		}
	}
}

func (c *Client) PrintInstructions() {
	peerID := c.host.ID().String()
	minWidth := len(" Your Peer ID: "+peerID) + 4 // 4 for border characters
	width := 80
	if minWidth > width {
		width = minWidth + 10
	}

	// Create the box
	topBorder := "┌" + strings.Repeat("─", width-2) + "┐"
	bottomBorder := "└" + strings.Repeat("─", width-2) + "┘"

	// Helper function to create a centered line
	centerLine := func(text string) string {
		if len(text) >= width-4 {
			return "│ " + text[:width-6] + "... │"
		}
		padding := (width - 4 - len(text)) / 2
		leftPad := strings.Repeat(" ", padding)
		rightPad := strings.Repeat(" ", width-4-len(text)-padding)
		return "│ " + leftPad + text + rightPad + " │"
	}

	// Helper function to create a left-aligned line
	leftLine := func(text string) string {
		if len(text) >= width-4 {
			return "│ " + text[:width-6] + "... │"
		}
		rightPad := strings.Repeat(" ", width-4-len(text))
		return "│ " + text + rightPad + " │"
	}

	// Print the fancy box
	fmt.Println()
	fmt.Println(topBorder)
	fmt.Println(centerLine("TORRENTIUM"))
	fmt.Println("│" + strings.Repeat("─", width-2) + "│")
	fmt.Println(centerLine("Available Commands"))
	fmt.Println("│" + strings.Repeat(" ", width-2) + "│")

	// Commands with descriptions
	commands := [][]string{
		{"add <path>", "Share a file or directory on the network"},
		{"list", "List your shared files"},
		{"search <cid|text>", "Search by CID or filename text"},
		{"download <cid>", "Download a file by CID (rarest-first)"},
		{"peers", "Show connected peers"},
		{"stats", "Show upload/download statistics and ratio"},
		{"seeding", "Show seeding (upload) progress"},
		{"maxupload <rate>", "Set max upload rate (bytes/sec or 'unlimited')"},
		{"announce <cid>", "Re-announce a file to DHT"},
		{"health", "Check connection health"},
		{"debug", "Show detailed network debug info"},
		{"help", "Show this help"},
		{"exit", "Exit the application"},
	}

	for _, cmd := range commands {
		cmdText := fmt.Sprintf(" %-20s - %s", cmd[0], cmd[1])
		fmt.Println(leftLine(cmdText))
	}

	fmt.Println("│" + strings.Repeat(" ", width-2) + "│")
	fmt.Println("│" + strings.Repeat("─", width-2) + "│")

	// Network info
	peerID = c.host.ID().String()
	fmt.Println(leftLine(" Your Peer ID: " + peerID))

	// Show listening addresses
	addrs := c.host.Addrs()
	fmt.Println(leftLine(" Listening on:"))

	for i, addr := range addrs {
		if i >= 3 { // Limit to 3 addresses to fit in box
			moreAddrs := len(addrs) - 3
			fmt.Println(leftLine(fmt.Sprintf("   ... and %d more", moreAddrs)))
			break
		}
		addrStr := addr.String()
		if len(addrStr) > width-8 {
			addrStr = addrStr[:width-11] + "..."
		}
		fmt.Println(leftLine("   " + addrStr))
	}

	fmt.Println(bottomBorder)
	fmt.Println()
}

func (c *Client) DebugNetworkStatus() {
	fmt.Println("\n=== Network Debug Info ===")
	fmt.Printf("Our Peer ID: %s\n", c.host.ID())
	fmt.Printf("Our Addresses:\n")
	for _, addr := range c.host.Addrs() {
		fmt.Printf(" %s/p2p/%s\n", addr, c.host.ID())
	}

	peers := c.host.Network().Peers()
	fmt.Printf("\nConnected Peers (%d):\n", len(peers))
	for i, peerID := range peers {
		conn := c.host.Network().ConnsToPeer(peerID)
		if len(conn) > 0 {
			fmt.Printf(" %d. %s\n", i+1, peerID)
			fmt.Printf("    Address: %s\n", conn[0].RemoteMultiaddr())
		}
	}

	routingTableSize := c.dht.RoutingTable().Size()
	fmt.Printf("\nDHT Routing Table Size: %d\n", routingTableSize)

	fmt.Printf("\nShared Files (%d):\n", len(c.sharingFiles))
	for cid, fileInfo := range c.sharingFiles {
		fmt.Printf(" CID: %s\n", cid)
		fmt.Printf(" File: %s\n", fileInfo.Name)
		fmt.Printf(" ---\n")
	}
}

func (c *Client) AnnounceFile(cidStr string) error {
	fileCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}
	fmt.Printf("Re-announcing CID %s to DHT...\n", cidStr)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := c.dht.Provide(ctx, fileCID, true); err != nil {
		return fmt.Errorf("failed to announce: %w", err)
	}
	fmt.Println(" - Successfully announced to DHT")
	return nil
}

func (c *Client) StartDHTMaintenance() {
	log := logger.WithComponent("dht")
	cfg := getConfig()
	go func() {
		ticker := time.NewTicker(cfg.P2P.DHTRefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			log.Debug().Msg("Performing DHT maintenance...")
			c.dht.RefreshRoutingTable()
			peers := c.host.Network().Peers()
			log.Debug().Int("peers", len(peers)).Msg("Peer count check")
			if len(peers) < cfg.P2P.MinPeerCount {
				log.Info().Msg("Low peer count; re-bootstrapping...")
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				// Note: Bootstrap is called via the DHT's internal refresh
				if err := c.dht.Bootstrap(ctx); err != nil {
					log.Warn().Err(err).Msg("DHT re-bootstrap failed")
				}
				cancel()
			}
		}
	}()
}

func (c *Client) CheckConnectionHealth() {
	peers := c.host.Network().Peers()
	fmt.Printf("\n=== Connection Health ===\n")
	fmt.Printf("Connected peers: %d\n", len(peers))
	if len(peers) < 3 {
		fmt.Println(" - Warning: Low peer count. Consider restarting or checking network connectivity.")
	} else {
		fmt.Println(" - Good peer connectivity")
	}

	routingTableSize := c.dht.RoutingTable().Size()
	fmt.Printf("DHT routing table size: %d\n", routingTableSize)
	if routingTableSize < 10 {
		fmt.Println(" - Warning: Small DHT routing table. File discovery may be limited.")
	} else {
		fmt.Println(" - Good DHT connectivity")
	}
}

func (c *Client) AddFile(filePath string) error {
	ctx := context.Background()

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		return c.addDirectory(ctx, filePath)
	}

	return c.addSingleFile(ctx, filePath)
}

// addSingleFile adds a single file to the network
func (c *Client) addSingleFile(ctx context.Context, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	fileHashBytes := hasher.Sum(nil)
	fileHashStr := hex.EncodeToString(fileHashBytes)
	mhash, err := multihash.Encode(fileHashBytes, multihash.SHA2_256)
	if err != nil {
		return fmt.Errorf("failed to create multihash: %w", err)
	}

	fileCID := cid.NewCidV1(cid.Raw, mhash)

	// Create pieces manifest
	pieceSz := int64(DefaultPieceSize)
	numPieces := (info.Size() + pieceSz - 1) / pieceSz
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	for idx := int64(0); idx < numPieces; idx++ {
		offset := idx * pieceSz
		size := min64(pieceSz, info.Size()-offset)
		h := sha256.New()
		if _, err := io.CopyN(h, f, size); err != nil {
			return err
		}
		ph := hex.EncodeToString(h.Sum(nil))
		if err := c.db.UpsertPiece(ctx, fileCID.String(), idx, offset, size, ph, true); err != nil {
			return err
		}
	}

	// Add single file entry for consistency
	if err := c.db.AddFileEntry(ctx, fileCID.String(), info.Name(), info.Size(), 0, fileHashStr); err != nil {
		return fmt.Errorf("failed to store file entry: %w", err)
	}

	if err := c.db.AddLocalFile(ctx, fileCID.String(), info.Name(), info.Size(), filePath, fileHashStr); err != nil {
		return fmt.Errorf("failed to store file metadata: %w", err)
	}

	c.sharingFiles[fileCID.String()] = &FileInfo{
		FilePath:    filePath,
		Hash:        fileHashStr,
		Size:        info.Size(),
		Name:        info.Name(),
		PieceSz:     pieceSz,
		IsDirectory: false,
		Files:       []db.FileEntry{{Path: info.Name(), Size: info.Size(), Offset: 0, FileHash: fileHashStr}},
	}

	logger.Info().Msgf("Announcing file %s with CID %s to DHT...", info.Name(), fileCID.String())
	provideCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := c.dht.Provide(provideCtx, fileCID, true); err != nil {
		logger.Info().Msgf(" - Warning: Failed to announce to DHT: %v", err)
	} else {
		logger.Info().Msg(" - Successfully announced file to DHT")
	}

	fmt.Printf("✓ File '%s' is now being shared\n", info.Name())
	fmt.Printf(" CID: %s\n", fileCID.String())
	fmt.Printf(" Hash: %s\n", fileHashStr)
	fmt.Printf(" Size: %s\n", humanize.Bytes(uint64(info.Size())))
	return nil
}

// addDirectory adds a directory (multi-file torrent) to the network
func (c *Client) addDirectory(ctx context.Context, dirPath string) error {
	dirName := filepath.Base(dirPath)
	var files []db.FileEntry
	var totalSize int64
	var currentOffset int64

	// Walk the directory and collect all files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		// Normalize path separators
		relPath = filepath.ToSlash(relPath)

		// Calculate file hash
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}
		fileHash := hex.EncodeToString(hasher.Sum(nil))

		files = append(files, db.FileEntry{
			Path:     relPath,
			Size:     info.Size(),
			Offset:   currentOffset,
			FileHash: fileHash,
		})

		currentOffset += info.Size()
		totalSize += info.Size()
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found in directory")
	}

	// Calculate combined hash for CID
	combinedHasher := sha256.New()
	for _, fe := range files {
		combinedHasher.Write([]byte(fe.Path))
		combinedHasher.Write([]byte(fe.FileHash))
	}
	combinedHash := combinedHasher.Sum(nil)
	combinedHashStr := hex.EncodeToString(combinedHash)

	mhash, err := multihash.Encode(combinedHash, multihash.SHA2_256)
	if err != nil {
		return fmt.Errorf("failed to create multihash: %w", err)
	}
	dirCID := cid.NewCidV1(cid.Raw, mhash)

	// Create pieces from concatenated file data
	pieceSz := int64(DefaultPieceSize)
	numPieces := (totalSize + pieceSz - 1) / pieceSz

	// Process pieces by reading from files in order
	var currentFileIdx int
	var currentFileOffset int64
	var currentFile *os.File

	for idx := int64(0); idx < numPieces; idx++ {
		offset := idx * pieceSz
		pieceSize := min64(pieceSz, totalSize-offset)

		pieceHasher := sha256.New()
		bytesRemaining := pieceSize

		for bytesRemaining > 0 {
			// Open current file if needed
			if currentFile == nil {
				filePath := filepath.Join(dirPath, filepath.FromSlash(files[currentFileIdx].Path))
				var err error
				currentFile, err = os.Open(filePath)
				if err != nil {
					return fmt.Errorf("failed to open file %s: %w", filePath, err)
				}
				currentFileOffset = 0
			}

			// Calculate how much to read from current file
			fileRemaining := files[currentFileIdx].Size - currentFileOffset
			toRead := min64(bytesRemaining, fileRemaining)

			// Read and hash
			buf := make([]byte, toRead)
			n, err := currentFile.Read(buf)
			if err != nil && err != io.EOF {
				currentFile.Close()
				return fmt.Errorf("failed to read file: %w", err)
			}
			pieceHasher.Write(buf[:n])

			bytesRemaining -= int64(n)
			currentFileOffset += int64(n)

			// Move to next file if current is exhausted
			if currentFileOffset >= files[currentFileIdx].Size {
				currentFile.Close()
				currentFile = nil
				currentFileIdx++
				if currentFileIdx >= len(files) {
					break
				}
			}
		}

		pieceHash := hex.EncodeToString(pieceHasher.Sum(nil))
		if err := c.db.UpsertPiece(ctx, dirCID.String(), idx, offset, pieceSize, pieceHash, true); err != nil {
			return err
		}
	}

	if currentFile != nil {
		currentFile.Close()
	}

	// Store file entries in database
	for _, fe := range files {
		if err := c.db.AddFileEntry(ctx, dirCID.String(), fe.Path, fe.Size, fe.Offset, fe.FileHash); err != nil {
			return fmt.Errorf("failed to store file entry: %w", err)
		}
	}

	// Store as local file (directory)
	if err := c.db.AddLocalFile(ctx, dirCID.String(), dirName, totalSize, dirPath, combinedHashStr); err != nil {
		return fmt.Errorf("failed to store directory metadata: %w", err)
	}

	c.sharingFiles[dirCID.String()] = &FileInfo{
		FilePath:    dirPath,
		Hash:        combinedHashStr,
		Size:        totalSize,
		Name:        dirName,
		PieceSz:     pieceSz,
		IsDirectory: true,
		Files:       files,
	}

	logger.Info().Msgf("Announcing directory %s with CID %s to DHT...", dirName, dirCID.String())
	provideCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := c.dht.Provide(provideCtx, dirCID, true); err != nil {
		logger.Info().Msgf(" - Warning: Failed to announce to DHT: %v", err)
	} else {
		logger.Info().Msg(" - Successfully announced directory to DHT")
	}

	fmt.Printf("✓ Directory '%s' is now being shared\n", dirName)
	fmt.Printf(" CID: %s\n", dirCID.String())
	fmt.Printf(" Total Size: %s\n", humanize.Bytes(uint64(totalSize)))
	fmt.Printf(" Files: %d\n", len(files))
	for i, fe := range files {
		if i < 5 {
			fmt.Printf("   - %s (%s)\n", fe.Path, humanize.Bytes(uint64(fe.Size)))
		} else if i == 5 {
			fmt.Printf("   ... and %d more files\n", len(files)-5)
			break
		}
	}
	return nil
}

func (c *Client) ListLocalFiles() {
	ctx := context.Background()
	files, err := c.db.GetLocalFiles(ctx)
	if err != nil {
		logger.Info().Msgf("Error retrieving files: %v", err)
		return
	}
	if len(files) == 0 {
		fmt.Println(" - No files being shared.")
		return
	}
	fmt.Println("\n=== Your Shared Files ===")
	for _, file := range files {
		fmt.Printf("Name: %s\n", file.Filename)
		fmt.Printf(" CID: %s\n", file.CID)
		fmt.Printf(" Size: %s\n", humanize.Bytes(uint64(file.FileSize)))
		fmt.Printf(" Path: %s\n", file.FilePath)
		fmt.Println(" ---")
	}
}

func (c *Client) SearchByText(q string) error {
	ctx := context.Background()
	matches, err := c.db.SearchByFilename(ctx, q)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		fmt.Printf("Searching for files containing '%s'...\n", q)
		fmt.Println("Note: Direct filename search requires content indexing.")
		fmt.Println("Try using the CID if you have it, or check with known peers.")
		return nil
	}
	fmt.Printf("Local index matches for '%s':\n", q)
	for _, m := range matches {
		fmt.Printf("- %s  CID:%s\n", m.Filename, m.CID)
	}
	return nil
}

func (c *Client) EnhancedSearchByCID(cidStr string) error {
	fileCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}
	fmt.Printf("Searching for CID: %s\n", fileCID.String())
	providers, err := c.findProvidersWithTimeout(fileCID, 60*time.Second, MaxProviders)
	if err != nil {
		return fmt.Errorf("provider search failed: %w", err)
	}

	if len(providers) == 0 {
		fmt.Println("No providers found for this CID")
		fmt.Println("This could mean:")
		fmt.Println(" - The file is not being shared")
		fmt.Println(" - The provider is offline")
		fmt.Println(" - Network connectivity issues")
		fmt.Println(" - DHT routing problem")
		return nil
	}

	fmt.Printf("Found %d provider(s):\n", len(providers))
	for i, provider := range providers {
		fmt.Printf(" %d. %s\n", i+1, provider.ID)
		if c.host.Network().Connectedness(provider.ID) == network.Connected {
			fmt.Printf(" - Already connected\n")
		} else {
			fmt.Printf(" - Not connected\n")
		}
	}
	return nil
}

func (c *Client) findProvidersWithTimeout(id cid.Cid, timeout time.Duration, maxProviders int) ([]peer.AddrInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	providersChan := c.dht.FindProvidersAsync(ctx, id, maxProviders)
	var providers []peer.AddrInfo
	var totalFound int

	done := make(chan struct{})
	go func() {
		defer close(done)
		for provider := range providersChan {
			totalFound++
			if provider.ID != c.host.ID() {
				providers = append(providers, provider)
				fmt.Printf(" - Found provider %d: %s\n", len(providers), provider.ID)
				if len(providers) >= maxProviders {
					break
				}
			}
		}
	}()

	select {
	case <-done:
		fmt.Printf("Provider search completed. Found %d total providers, %d unique external providers\n",
			totalFound, len(providers))
	case <-time.After(timeout):
		fmt.Printf("Provider search timed out. Found %d providers so far\n", len(providers))
	}

	return providers, nil
}

func (c *Client) connectToPeer(multiaddrStr string) error {
	addr, err := multiaddr.NewMultiaddr(multiaddrStr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	fmt.Printf("Attempting to connect to peer %s...\n", peerInfo.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := c.host.Connect(ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Printf(" - Successfully connected to peer %s\n", peerInfo.ID)

	c.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour)

	return nil
}

func (c *Client) ListConnectedPeers() {
	peers := c.host.Network().Peers()
	fmt.Printf("\n=== Connected Peers (%d) ===\n", len(peers))
	for _, peerID := range peers {
		conn := c.host.Network().ConnsToPeer(peerID)
		if len(conn) > 0 {
			fmt.Printf("Peer: %s\n", peerID)
			fmt.Printf(" Address: %s\n", conn[0].RemoteMultiaddr())
		}
	}
}

func (c *Client) DownloadFile(cidStr string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileCID, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("invalid CID: %w", err)
	}

	fmt.Printf("Looking for providers of CID: %s\n", fileCID.String())
	providers, err := c.findProvidersWithTimeout(fileCID, 60*time.Second, MaxProviders)
	if err != nil {
		return fmt.Errorf("provider search failed: %w", err)
	}

	if len(providers) == 0 {
		return fmt.Errorf("no providers found")
	}

	fmt.Printf("Found %d providers. Getting file manifest...\n", len(providers))
	relayAddrStr := "/dns4/relay-torrentium-pj9h.onrender.com/tcp/443/wss/p2p/12D3KooWQmD64vYYegz3GVDHtt4KeqkSuBSYoLwGMircrv1Q1TdW"

	var manifest controlMessage
	var firstPeer *webRTC.SimpleWebRTCPeer
	connectedPeers := make(map[peer.ID]*webRTC.SimpleWebRTCPeer)

	for _, p := range providers {
		// Try direct connection first
		peerConn, err := c.initiateWebRTCConnectionWithRetry(p.ID, 1)
		if err != nil {
			// Fallback: relay connection
			logger.Debug().Msg("Direct connection failed, trying relay...")

			// Build circuit address
			circuitStr := fmt.Sprintf("%s/p2p-circuit/p2p/%s", relayAddrStr, p.ID.String())
			circuitMaddr, err := multiaddr.NewMultiaddr(circuitStr)
			if err != nil {
				logger.Warn().Err(err).Msg("Invalid circuit multiaddr")
				continue
			}
			targetInfo := peer.AddrInfo{ID: p.ID, Addrs: []multiaddr.Multiaddr{circuitMaddr}}

			if err := c.host.Connect(ctx, targetInfo); err != nil {
				logger.Info().Msgf("❌ Relay dial failed: %v", err)
				continue
			}

			logger.Info().Msgf("✅ Relay dial to %s successful", p.ID)

			//perform WebRTC handshake
			peerConn, err = c.initiateWebRTCConnectionWithRetry(p.ID, 1)
			if err != nil {
				logger.Info().Msgf("⚠ WebRTC connection via relay failed: %v", err)
				continue
			}
		}

		// Store connected peer
		if peerConn != nil {
			connectedPeers[p.ID] = peerConn
			if firstPeer == nil {
				manifest, err = c.requestManifest(peerConn, cidStr)
				if err == nil {
					firstPeer = peerConn
				} else {
					peerConn.Close()
					delete(connectedPeers, p.ID)
				}
			}
		}
	}

	if firstPeer == nil {
		return fmt.Errorf("failed to connect to any provider to get manifest")
	}

	// Store pieces in the database
	for _, piece := range manifest.Pieces {
		if err := c.db.UpsertPiece(ctx, cidStr, piece.Index, piece.Offset, piece.Size, piece.Hash, false); err != nil {
			logger.Info().Msgf("Failed to store piece info for download: %v", err)
		}
	}

	// Handle multi-file manifest
	isMultiFile := len(manifest.Files) > 1 || manifest.IsDirectory
	var downloadPath, finalPath string

	if isMultiFile {
		// Create directory for multi-file download
		downloadPath = manifest.Filename + ".download"
		finalPath = manifest.Filename
		if err := os.MkdirAll(downloadPath, 0755); err != nil {
			return fmt.Errorf("failed to create download directory: %w", err)
		}
		fmt.Printf("Multi-file torrent with %d files:\n", len(manifest.Files))
		for i, fe := range manifest.Files {
			if i < 5 {
				fmt.Printf("   - %s (%s)\n", fe.Path, humanize.Bytes(uint64(fe.Size)))
			} else if i == 5 {
				fmt.Printf("   ... and %d more files\n", len(manifest.Files)-5)
				break
			}
		}
	} else {
		downloadPath = fmt.Sprintf("%s.download", manifest.Filename)
		finalPath = manifest.Filename
	}

	// Create temp file for piece assembly (even for multi-file, we assemble then split)
	tempPath := downloadPath + ".tmp"
	localFile, err := os.Create(tempPath)
	if err != nil {
		firstPeer.Close()
		return fmt.Errorf("failed to create file: %w", err)
	}

	pieces, _ := c.db.GetPieces(ctx, cidStr)
	if len(pieces) == 0 {
		return fmt.Errorf("failed to retrieve piece information after receiving manifest")
	}

	state := &DownloadState{
		File:            localFile,
		Manifest:        manifest,
		TotalPieces:     int(manifest.NumPieces),
		Pieces:          pieces,
		Completed:       make(chan bool, 1),
		Progress:        progressbar.DefaultBytes(manifest.TotalSize, "downloading..."),
		PieceStatus:     make([]bool, int(manifest.NumPieces)),
		PieceAssignees:  make(map[int]peer.ID),
		pieceBuffers:    make(map[int][][]byte),
		completedPieces: 0,
		pieceTimers:     make(map[int]*time.Timer),
		retryCounts:     make(map[int]int),
		pieceRarity:     make(map[int]*PieceAvailability),
		peerBitfields:   make(map[peer.ID][]bool),
		Files:           manifest.Files,
		SelectedFiles:   nil, // nil means download all
	}

	// Initialize piece rarity (all pieces start with 0 peers)
	for i := 0; i < state.TotalPieces; i++ {
		state.pieceRarity[i] = &PieceAvailability{
			PieceIndex:  i,
			PeerCount:   0,
			HavingPeers: []peer.ID{},
		}
	}

	c.downloadsMux.Lock()
	c.activeDownloads[cidStr] = state
	c.downloadsMux.Unlock()

	// Request bitfields from all connected peers
	fmt.Println("Requesting piece availability from peers...")
	c.requestBitfieldsFromPeers(cidStr, connectedPeers, state)

	// Use rarest-first piece selection
	piecesToDownload := c.selectRarestPieces(state)
	fmt.Printf("Using rarest-first selection for %d pieces\n", len(piecesToDownload))

	// Assign pieces to peers based on rarity and peer availability
	assignments := c.assignPiecesToPeers(state, piecesToDownload, connectedPeers)

	// Parallel downloads
	var wg sync.WaitGroup
	for peerID, pieceIndices := range assignments {
		peerConn := connectedPeers[peerID]
		if peerConn == nil {
			continue
		}

		wg.Add(1)
		go func(conn *webRTC.SimpleWebRTCPeer, indices []int, pid peer.ID) {
			defer wg.Done()
			c.downloadPiecesFromPeer(conn, state, indices)
		}(peerConn, pieceIndices, peerID)
	}

	<-state.Completed
	localFile.Close()

	// Handle multi-file reassembly
	if isMultiFile {
		fmt.Println("Reassembling files...")
		if err := c.reassembleMultiFileDownload(tempPath, downloadPath, manifest.Files); err != nil {
			return fmt.Errorf("failed to reassemble files: %w", err)
		}
		os.Remove(tempPath)
		if err := os.Rename(downloadPath, finalPath); err != nil {
			return fmt.Errorf("failed to rename directory: %w", err)
		}
	} else {
		if err := os.Rename(tempPath, finalPath); err != nil {
			return fmt.Errorf("failed to rename file: %w", err)
		}
	}

	fmt.Printf("\n✅ Download complete. Saved as %s\n", finalPath)
	return nil
}

// requestBitfieldsFromPeers requests bitfields from all connected peers
func (c *Client) requestBitfieldsFromPeers(cidStr string, peers map[peer.ID]*webRTC.SimpleWebRTCPeer, state *DownloadState) {
	var wg sync.WaitGroup
	bitfieldCh := make(chan struct {
		pid      peer.ID
		bitfield []bool
	}, len(peers))

	for pid, peerConn := range peers {
		wg.Add(1)
		go func(pid peer.ID, conn *webRTC.SimpleWebRTCPeer) {
			defer wg.Done()
			req := controlMessage{Command: "REQUEST_BITFIELD", CID: cidStr}
			if err := conn.SendJSONReliable(req); err != nil {
				logger.Info().Msgf("Failed to request bitfield from %s: %v", pid, err)
				return
			}
			// For simplicity, assume peer has all pieces if it's providing
			// In a real implementation, you'd wait for BITFIELD response
			fullBitfield := make([]bool, state.TotalPieces)
			for i := range fullBitfield {
				fullBitfield[i] = true
			}
			bitfieldCh <- struct {
				pid      peer.ID
				bitfield []bool
			}{pid, fullBitfield}
		}(pid, peerConn)
	}

	// Wait with timeout
	go func() {
		wg.Wait()
		close(bitfieldCh)
	}()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case bf, ok := <-bitfieldCh:
			if !ok {
				return
			}
			state.mu.Lock()
			state.peerBitfields[bf.pid] = bf.bitfield
			// Update piece rarity
			for i, has := range bf.bitfield {
				if has {
					state.pieceRarity[i].PeerCount++
					state.pieceRarity[i].HavingPeers = append(state.pieceRarity[i].HavingPeers, bf.pid)
				}
			}
			state.mu.Unlock()
		case <-timeout:
			return
		}
	}
}

// selectRarestPieces returns pieces sorted by rarity (rarest first)
func (c *Client) selectRarestPieces(state *DownloadState) []int {
	state.mu.Lock()
	defer state.mu.Unlock()

	// Collect all pieces that need to be downloaded
	var pieces []*PieceAvailability
	for i := 0; i < state.TotalPieces; i++ {
		if !state.PieceStatus[i] {
			pieces = append(pieces, state.pieceRarity[i])
		}
	}

	// Sort by peer count (ascending = rarest first)
	sort.Slice(pieces, func(i, j int) bool {
		// If peer counts are equal, randomize to avoid all peers requesting same piece
		if pieces[i].PeerCount == pieces[j].PeerCount {
			return pieces[i].PieceIndex < pieces[j].PieceIndex
		}
		return pieces[i].PeerCount < pieces[j].PeerCount
	})

	// Extract piece indices
	result := make([]int, len(pieces))
	for i, p := range pieces {
		result[i] = p.PieceIndex
	}

	return result
}

// assignPiecesToPeers distributes pieces to peers, preferring peers who have rarer pieces
func (c *Client) assignPiecesToPeers(state *DownloadState, pieces []int, peers map[peer.ID]*webRTC.SimpleWebRTCPeer) map[peer.ID][]int {
	assignments := make(map[peer.ID][]int)

	state.mu.Lock()
	defer state.mu.Unlock()

	// For each piece (already sorted by rarity), assign to a peer that has it
	for _, pieceIdx := range pieces {
		rarity := state.pieceRarity[pieceIdx]

		// Find best peer for this piece
		var bestPeer peer.ID
		minAssigned := int(^uint(0) >> 1) // Max int

		for _, pid := range rarity.HavingPeers {
			if _, ok := peers[pid]; !ok {
				continue // Peer not connected
			}
			currentAssigned := len(assignments[pid])
			if currentAssigned < minAssigned {
				minAssigned = currentAssigned
				bestPeer = pid
			}
		}

		// If no peer with this piece found, assign to any peer (fallback)
		if bestPeer == "" {
			for pid := range peers {
				if len(assignments[pid]) < minAssigned {
					bestPeer = pid
					break
				}
			}
		}

		if bestPeer != "" {
			assignments[bestPeer] = append(assignments[bestPeer], pieceIdx)
			state.PieceAssignees[pieceIdx] = bestPeer
		}
	}

	return assignments
}

// downloadPiecesFromPeer downloads specific pieces from a peer
func (c *Client) downloadPiecesFromPeer(peer *webRTC.SimpleWebRTCPeer, state *DownloadState, pieces []int) {
	for _, pieceIdx := range pieces {
		state.mu.Lock()
		if state.PieceStatus[pieceIdx] {
			state.mu.Unlock()
			continue
		}
		state.mu.Unlock()

		req := controlMessage{
			Command: "REQUEST_PIECE",
			CID:     state.Manifest.CID,
			Index:   int64(pieceIdx),
		}

		state.mu.Lock()
		state.pieceTimers[pieceIdx] = time.AfterFunc(PieceTimeout, func() {
			logger.Info().Msgf("Piece %d timed out, re-requesting...", pieceIdx)
			c.reRequestPiece(state, pieceIdx)
		})
		state.mu.Unlock()

		if err := peer.SendJSONReliable(req); err != nil {
			logger.Info().Msgf("Failed to request piece %d: %v", pieceIdx, err)
			return
		}
	}
}

// reassembleMultiFileDownload splits the concatenated data into individual files
func (c *Client) reassembleMultiFileDownload(tempPath, outputDir string, files []db.FileEntry) error {
	tempFile, err := os.Open(tempPath)
	if err != nil {
		return err
	}
	defer tempFile.Close()

	for _, fe := range files {
		// Create subdirectories if needed
		filePath := filepath.Join(outputDir, filepath.FromSlash(fe.Path))
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		// Seek to file offset
		if _, err := tempFile.Seek(fe.Offset, io.SeekStart); err != nil {
			return err
		}

		// Create output file
		outFile, err := os.Create(filePath)
		if err != nil {
			return err
		}

		// Copy file data
		if _, err := io.CopyN(outFile, tempFile, fe.Size); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()

		fmt.Printf("   ✓ %s\n", fe.Path)
	}

	return nil
}

func (c *Client) reRequestPiece(state *DownloadState, pieceIndex int) {
	// Re-assign to another peer with backoff
	retryCount := state.retryCounts[pieceIndex]
	backoff := ExponentialBackoffBase * time.Duration(1<<retryCount)
	if backoff > MaxBackoff {
		backoff = MaxBackoff
	}
	time.AfterFunc(backoff, func() {
		// For simplicity, we'll just re-request from any connected peer.
		// A more advanced implementation would select a new peer.
		for _, p := range c.webRTCPeers {
			req := controlMessage{
				Command: "REQUEST_PIECE",
				CID:     state.Manifest.CID,
				Index:   int64(pieceIndex),
			}
			if err := p.SendJSONReliable(req); err == nil {
				logger.Info().Msgf("Re-requested piece %d from a different peer.", pieceIndex)
				return
			}
		}
		logger.Info().Msgf("Failed to re-request piece %d: no available peers.", pieceIndex)
	})
	state.retryCounts[pieceIndex]++
}

func (c *Client) requestManifest(peer *webRTC.SimpleWebRTCPeer, cidStr string) (controlMessage, error) {
	req := controlMessage{Command: "REQUEST_MANIFEST", CID: cidStr}
	if err := peer.SendJSONReliable(req); err != nil {
		return controlMessage{}, err
	}

	manifestCh := make(chan controlMessage, 1)
	manifestChMu.Lock()
	manifestWaiters[cidStr] = manifestCh
	manifestChMu.Unlock()

	defer func() {
		manifestChMu.Lock()
		delete(manifestWaiters, cidStr)
		manifestChMu.Unlock()
	}()

	select {
	case manifest := <-manifestCh:
		return manifest, nil
	case <-time.After(30 * time.Second):
		return controlMessage{}, fmt.Errorf("timed out waiting for manifest")
	}
}

func (c *Client) initiateWebRTCConnectionWithRetry(targetPeerID peer.ID, maxRetries int) (*webRTC.SimpleWebRTCPeer, error) {

	// First, test ICE connectivity
	logger.Info().Msgf("Testing ICE connectivity before attempting WebRTC connection...")
	if err := webRTC.TestICEConnectivity(); err != nil {
		logger.Info().Msgf("ICE connectivity test failed: %v", err)
		logger.Info().Msgf("Warning: WebRTC connections may fail due to network restrictions")
	} else {
		logger.Info().Msgf("ICE connectivity test passed")
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			fmt.Printf("Retrying in %v (attempt %d/%d)...\n", backoff, attempt, maxRetries)
			time.Sleep(backoff)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		info := peer.AddrInfo{ID: targetPeerID}
		if pinfo, err := c.dht.FindPeer(ctx, targetPeerID); err == nil && len(pinfo.Addrs) > 0 {
			info = pinfo
		} else {

			lastErr = fmt.Errorf("dht lookup failed: %w", err)
			continue
		}

		if len(info.Addrs) == 0 {
			lastErr = fmt.Errorf("peer %s has no known multiaddrs", targetPeerID)
			continue
		}

		c.host.Peerstore().AddAddrs(info.ID, info.Addrs, time.Hour)
		fmt.Println(c.host.Peerstore())

		if c.host.Network().Connectedness(info.ID) != network.Connected {
			connectCtx, connectCancel := context.WithTimeout(context.Background(), 20*time.Second)
			err := c.host.Connect(connectCtx, info)
			connectCancel()
			if err != nil {
				logger.Info().Msgf("failed to connect to peer %s: %v", info.ID, err)
				fmt.Printf("DHT lookup failed: %v. This could be a network issue now trying connection using relays.\n", err)
				return nil, err
			}

			fmt.Printf("Successfully connected to peer %s\n", info.ID)

			time.Sleep(1 * time.Second)
		}

		webrtcPeer, err := webRTC.NewSimpleWebRTCPeer(c.onDataChannelMessage, c.onWebRTCPeerClose)
		if err != nil {
			lastErr = err
			return nil, err
		}

		offer, err := webrtcPeer.CreateOffer()
		if err != nil {
			webrtcPeer.Close()
			lastErr = err
			return nil, err
		}

		streamCtx, streamCancel := context.WithTimeout(context.Background(), 30*time.Second)
		s, err := c.host.NewStream(streamCtx, targetPeerID, SignalingProtocolID)
		streamCancel()
		if err != nil {
			webrtcPeer.Close()
			lastErr = err
			return nil, err

		}

		webrtcPeer.SetSignalingStream(s)
		encoder := json.NewEncoder(s)
		decoder := json.NewDecoder(s)

		offerMsg := map[string]string{"type": "offer", "data": offer}
		if err := encoder.Encode(offerMsg); err != nil {
			webrtcPeer.Close()
			lastErr = err
			continue
		}

		var answerMsg map[string]string
		if err := decoder.Decode(&answerMsg); err != nil {
			webrtcPeer.Close()
			lastErr = fmt.Errorf("failed to decode answer: %w", err)
			continue
		}

		if answerMsg["type"] == "error" {
			webrtcPeer.Close()
			lastErr = fmt.Errorf("peer returned error: %s", answerMsg["data"])
			continue
		}
		if answerMsg["type"] != "answer" {
			webrtcPeer.Close()
			lastErr = fmt.Errorf("expected answer, got: %s", answerMsg["type"])
			continue
		}

		if err := webrtcPeer.HandleAnswer(answerMsg["data"]); err != nil {
			webrtcPeer.Close()
			lastErr = err
			continue
		}

		if err := webrtcPeer.WaitForConnection(90 * time.Second); err != nil {
			webrtcPeer.Close()
			lastErr = fmt.Errorf("WebRTC connection failed: %w", err)
			continue
		}

		//wait for the data channels to be ready
		if err := webrtcPeer.WaitForDataChannels(10 * time.Second); err != nil {
			webrtcPeer.Close()
			lastErr = fmt.Errorf("data channels did not open in time: %w", err)
			continue
		}

		fmt.Printf("WebRTC connection established with %s\n", targetPeerID)
		c.peersMux.Lock()
		c.webRTCPeers[targetPeerID] = webrtcPeer
		c.peersMux.Unlock()

		return webrtcPeer, nil
	}
	return nil, lastErr
}

// HandleWebRTCOffer handles incoming WebRTC offers from peers
func (c *Client) HandleWebRTCOffer(offer, remotePeerID string, s network.Stream) (string, error) {
	peerID, err := peer.Decode(remotePeerID)
	if err != nil {
		return "", fmt.Errorf("invalid peer ID: %w", err)
	}

	webrtcPeer, err := webRTC.NewSimpleWebRTCPeer(c.onDataChannelMessage, c.onWebRTCPeerClose)
	if err != nil {
		return "", err
	}

	webrtcPeer.SetSignalingStream(s)

	answer, err := webrtcPeer.HandleOffer(offer)
	if err != nil {
		webrtcPeer.Close()
		return "", err
	}

	c.peersMux.Lock()
	c.webRTCPeers[peerID] = webrtcPeer
	c.peersMux.Unlock()

	return answer, nil
}

func (c *Client) onDataChannelMessage(msg webrtc.DataChannelMessage, peer *webRTC.SimpleWebRTCPeer) {
	if !msg.IsString {
		logger.Info().Msgf("Received unexpected binary message, expecting JSON.")
		return
	}
	if len(msg.Data) == 0 {
		return
	}
	var ctrl controlMessage
	if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
		var ping map[string]string
		if err2 := json.Unmarshal(msg.Data, &ping); err2 == nil {
			if ping["type"] == "ping" {
				// Respond to ping
				pong := map[string]string{"type": "pong"}
				peer.SendJSONReliable(pong)
				return
			} else if ping["type"] == "pong" {
				c.handlePong(peer.GetSignalingStream().Conn().RemotePeer())
				return
			}
		}
		logger.Info().Msgf("Failed to unmarshal control message: %v. Raw message: %s", err, string(msg.Data))
		return
	}
	c.handleControlMessage(ctrl, peer)
}

func (c *Client) handleControlMessage(ctrl controlMessage, peer *webRTC.SimpleWebRTCPeer) {
	ctx := context.Background()
	switch ctrl.Command {
	case "REQUEST_MANIFEST":
		c.handleManifestRequest(ctx, ctrl, peer)
	case "MANIFEST":
		manifestChMu.Lock()
		if ch, ok := manifestWaiters[ctrl.CID]; ok {
			ch <- ctrl
		}
		manifestChMu.Unlock()
	case "REQUEST_PIECE":
		go c.handlePieceRequest(ctx, ctrl, peer)
	case "PIECE_CHUNK":
		c.handlePieceChunk(ctrl, peer)
	case "CHUNK_ACK":
		c.handleChunkAck(ctrl)
	case "REQUEST_BITFIELD":
		c.handleBitfieldRequest(ctx, ctrl, peer)
	case "BITFIELD":
		c.handleBitfield(ctrl, peer)
	case "HAVE":
		c.handleHave(ctrl, peer)
	default:
		//do nothing
	}
}

func (c *Client) handlePieceChunk(ctrl controlMessage, peer *webRTC.SimpleWebRTCPeer) {
	c.downloadsMux.RLock()
	state, ok := c.activeDownloads[ctrl.CID]
	c.downloadsMux.RUnlock()
	if !ok {
		return
	}

	// Send an ACK back to the sender using reliable channel
	ackMsg := controlMessage{
		Command:  "CHUNK_ACK",
		CID:      ctrl.CID,
		Index:    ctrl.Index,
		Sequence: ctrl.Sequence,
	}
	if err := peer.SendJSONReliable(ackMsg); err != nil {
		logger.Info().Msgf("Failed to send ACK for chunk %d of piece %d: %v", ctrl.Sequence, ctrl.Index, err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.PieceStatus[ctrl.Index] {
		return // Already have this piece
	}

	if state.pieceBuffers[int(ctrl.Index)] == nil {
		state.pieceBuffers[int(ctrl.Index)] = make([][]byte, ctrl.TotalChunks)
	}

	chunkData, err := hex.DecodeString(ctrl.Payload)
	if err != nil {
		logger.Info().Msgf("Failed to decode chunk payload: %v", err)
		return
	}

	state.pieceBuffers[int(ctrl.Index)][ctrl.ChunkIndex] = chunkData
	_ = state.Progress.Add(len(chunkData))

	// Check if piece is complete
	isComplete := true
	var pieceSize int
	for _, chunk := range state.pieceBuffers[int(ctrl.Index)] {
		if chunk == nil {
			isComplete = false
			break
		}
		pieceSize += len(chunk)
	}

	if isComplete {
		//Stop the timer for this piece
		if timer, ok := state.pieceTimers[int(ctrl.Index)]; ok {
			timer.Stop()
			delete(state.pieceTimers, int(ctrl.Index))
		}

		// Reassemble and write piece
		pieceData := make([]byte, 0, pieceSize)
		for _, chunk := range state.pieceBuffers[int(ctrl.Index)] {
			pieceData = append(pieceData, chunk...)
		}

		//Verify piece hash
		h := sha256.New()
		h.Write(pieceData)
		hash := hex.EncodeToString(h.Sum(nil))

		if hash != state.Pieces[ctrl.Index].Hash {
			logger.Info().Msgf("Piece %d hash mismatch", ctrl.Index)
			state.pieceBuffers[int(ctrl.Index)] = nil // Clear buffer to retry
			return
		}

		if _, err := state.File.WriteAt(pieceData, state.Pieces[ctrl.Index].Offset); err != nil {
			logger.Info().Msgf("Failed to write piece %d to file: %v", ctrl.Index, err)
			return
		}

		state.PieceStatus[ctrl.Index] = true
		state.completedPieces++
		delete(state.pieceBuffers, int(ctrl.Index))

		if state.completedPieces == state.TotalPieces {
			state.Completed <- true
		}
	}
}

func (c *Client) handleChunkAck(ctrl controlMessage) {
	c.unackedChunksMux.Lock()
	defer c.unackedChunksMux.Unlock()
	if _, ok := c.unackedChunks[ctrl.CID]; ok {
		if _, ok := c.unackedChunks[ctrl.CID][ctrl.Index]; ok {
			delete(c.unackedChunks[ctrl.CID][ctrl.Index], ctrl.Sequence)
			if len(c.unackedChunks[ctrl.CID][ctrl.Index]) == 0 {
				delete(c.unackedChunks[ctrl.CID], ctrl.Index)
			}
		}
		if len(c.unackedChunks[ctrl.CID]) == 0 {
			delete(c.unackedChunks, ctrl.CID)
		}
	}
}

func (c *Client) handlePieceRequest(ctx context.Context, ctrl controlMessage, webrtcPeer *webRTC.SimpleWebRTCPeer) {
	pieces, err := c.db.GetPieces(ctx, ctrl.CID)
	if err != nil || int(ctrl.Index) >= len(pieces) {
		logger.Info().Msgf("Invalid piece request for CID %s, index %d", ctrl.CID, ctrl.Index)
		return
	}

	fileInfo, err := c.db.GetLocalFileByCID(ctx, ctrl.CID)
	if err != nil {
		logger.Info().Msgf("File not found for piece request: %s", ctrl.CID)
		return
	}

	// Check if it's a multi-file torrent
	fileEntries, _ := c.db.GetFileEntries(ctx, ctrl.CID)
	isMultiFile := len(fileEntries) > 1

	var pieceBuffer []byte
	piece := pieces[ctrl.Index]

	if isMultiFile {
		// Read from multiple files based on offset
		pieceBuffer, err = c.readPieceFromMultipleFiles(fileInfo.FilePath, fileEntries, piece.Offset, piece.Size)
	} else {
		file, err := os.Open(fileInfo.FilePath)
		if err != nil {
			logger.Info().Msgf("Failed to open file for piece request: %v", err)
			return
		}
		defer file.Close()

		pieceBuffer = make([]byte, piece.Size)
		_, err = file.ReadAt(pieceBuffer, piece.Offset)
	}

	if err != nil {
		logger.Info().Msgf("Failed to read piece %d: %v", ctrl.Index, err)
		return
	}

	peerID := webrtcPeer.GetSignalingStream().Conn().RemotePeer()

	// Initialize upload stats for this CID if needed
	c.uploadProgressMu.Lock()
	if c.uploadProgress[ctrl.CID] == nil {
		c.uploadProgress[ctrl.CID] = &UploadStats{
			CID:         ctrl.CID,
			PeersServed: make(map[peer.ID]int64),
			StartTime:   time.Now(),
		}
	}
	stats := c.uploadProgress[ctrl.CID]
	c.uploadProgressMu.Unlock()

	totalChunks := (len(pieceBuffer) + MaxChunk - 1) / MaxChunk
	var bytesUploaded int64

	for i := 0; i < totalChunks; i++ {
		start := i * MaxChunk
		end := start + MaxChunk
		if end > len(pieceBuffer) {
			end = len(pieceBuffer)
		}
		chunk := pieceBuffer[start:end]

		chunkMsg := controlMessage{
			Command:     "PIECE_CHUNK",
			CID:         ctrl.CID,
			Index:       ctrl.Index,
			ChunkIndex:  i,
			TotalChunks: totalChunks,
			Payload:     hex.EncodeToString(chunk),
			Sequence:    i,
		}

		// Store the sent chunk and start a retransmission timer
		c.unackedChunksMux.Lock()
		if c.unackedChunks[ctrl.CID] == nil {
			c.unackedChunks[ctrl.CID] = make(map[int64]map[int]controlMessage)
		}
		if c.unackedChunks[ctrl.CID][ctrl.Index] == nil {
			c.unackedChunks[ctrl.CID][ctrl.Index] = make(map[int]controlMessage)
		}
		c.unackedChunks[ctrl.CID][ctrl.Index][i] = chunkMsg
		c.unackedChunksMux.Unlock()
		time.AfterFunc(RetransmissionTimeout, func() { c.retransmitChunk(webrtcPeer, chunkMsg) })

		if err := webrtcPeer.SendJSON(chunkMsg); err != nil {
			logger.Info().Msgf("Failed to send chunk %d of piece %d: %v", i, ctrl.Index, err)
			return
		}

		bytesUploaded += int64(len(chunk))

		// Apply upload rate limiting if configured
		if c.maxUploadRate > 0 {
			delay := time.Duration(float64(len(chunk)) / float64(c.maxUploadRate) * float64(time.Second))
			time.Sleep(delay)
		} else {
			delay := c.congestionCtrl[webrtcPeer.GetSignalingStream().Conn().RemotePeer()]
			time.Sleep(delay)
		}
	}

	// Update upload statistics
	stats.mu.Lock()
	stats.BytesUploaded += bytesUploaded
	stats.ChunksServed += totalChunks
	stats.PeersServed[peerID] += bytesUploaded
	stats.LastActivity = time.Now()
	stats.mu.Unlock()

	// Persist upload stats to database
	if err := c.db.RecordUpload(ctx, ctrl.CID, peerID.String(), bytesUploaded, totalChunks); err != nil {
		logger.Info().Msgf("Failed to record upload stats: %v", err)
	}
}

// readPieceFromMultipleFiles reads piece data that may span multiple files
func (c *Client) readPieceFromMultipleFiles(basePath string, files []db.FileEntry, offset, size int64) ([]byte, error) {
	result := make([]byte, 0, size)
	remaining := size
	currentOffset := offset

	for _, fe := range files {
		fileEnd := fe.Offset + fe.Size

		// Skip files before the offset
		if fileEnd <= currentOffset {
			continue
		}

		// Calculate read position within this file
		fileReadStart := currentOffset - fe.Offset
		if fileReadStart < 0 {
			fileReadStart = 0
		}

		// Calculate how much to read from this file
		availableInFile := fe.Size - fileReadStart
		toRead := min64(remaining, availableInFile)

		// Open and read from file
		filePath := filepath.Join(basePath, filepath.FromSlash(fe.Path))
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, toRead)
		_, err = f.ReadAt(buf, fileReadStart)
		f.Close()
		if err != nil && err != io.EOF {
			return nil, err
		}

		result = append(result, buf...)
		remaining -= toRead
		currentOffset += toRead

		if remaining <= 0 {
			break
		}
	}

	return result, nil
}

func (c *Client) retransmitChunk(peer *webRTC.SimpleWebRTCPeer, chunkMsg controlMessage) {
	c.unackedChunksMux.RLock()
	defer c.unackedChunksMux.RUnlock()
	if _, ok := c.unackedChunks[chunkMsg.CID]; ok {
		if _, ok := c.unackedChunks[chunkMsg.CID][chunkMsg.Index]; ok {
			if _, ok := c.unackedChunks[chunkMsg.CID][chunkMsg.Index][chunkMsg.Sequence]; ok {
				logger.Info().Msgf("Retransmitting chunk %d of piece %d", chunkMsg.Sequence, chunkMsg.Index)
				if err := peer.SendJSON(chunkMsg); err != nil {
					logger.Info().Msgf("Failed to retransmit chunk %d of piece %d: %v", chunkMsg.Sequence, chunkMsg.Index, err)
				}
				// Reset timer
				time.AfterFunc(RetransmissionTimeout, func() { c.retransmitChunk(peer, chunkMsg) })
			}
		}
	}
}

func (c *Client) handleManifestRequest(ctx context.Context, ctrl controlMessage, peer *webRTC.SimpleWebRTCPeer) {
	localFile, err := c.db.GetLocalFileByCID(ctx, ctrl.CID)
	if err != nil {
		logger.Info().Msgf("File not found for manifest: %s", ctrl.CID)
		return
	}

	pieces, err := c.db.GetPieces(ctx, ctrl.CID)
	if err != nil {
		logger.Info().Msgf("Error getting pieces: %v", err)
		return
	}

	// Get file entries for multi-file support
	fileEntries, err := c.db.GetFileEntries(ctx, ctrl.CID)
	if err != nil {
		logger.Info().Msgf("Error getting file entries: %v", err)
		fileEntries = []db.FileEntry{} // Empty for single file
	}

	isDirectory := len(fileEntries) > 1

	manifest := controlMessage{
		Command:     "MANIFEST",
		CID:         ctrl.CID,
		TotalSize:   localFile.FileSize,
		HashHex:     localFile.FileHash,
		NumPieces:   int64(len(pieces)),
		Pieces:      pieces,
		Filename:    localFile.Filename,
		Files:       fileEntries,
		IsDirectory: isDirectory,
	}

	if err := peer.SendJSONReliable(manifest); err != nil {
		logger.Info().Msgf("Error sending manifest: %v", err)
	}
}

// handleBitfieldRequest responds with our bitfield for a CID
func (c *Client) handleBitfieldRequest(ctx context.Context, ctrl controlMessage, peer *webRTC.SimpleWebRTCPeer) {
	pieces, err := c.db.GetPieces(ctx, ctrl.CID)
	if err != nil {
		logger.Info().Msgf("Error getting pieces for bitfield: %v", err)
		return
	}

	bitfield := make([]bool, len(pieces))
	for i, p := range pieces {
		bitfield[i] = p.Have
	}

	response := controlMessage{
		Command:  "BITFIELD",
		CID:      ctrl.CID,
		Bitfield: bitfield,
	}

	if err := peer.SendJSONReliable(response); err != nil {
		logger.Info().Msgf("Error sending bitfield: %v", err)
	}
}

// handleBitfield processes a received bitfield from a peer
func (c *Client) handleBitfield(ctrl controlMessage, peer *webRTC.SimpleWebRTCPeer) {
	peerID := peer.GetSignalingStream().Conn().RemotePeer()

	c.downloadsMux.RLock()
	state, ok := c.activeDownloads[ctrl.CID]
	c.downloadsMux.RUnlock()

	if !ok {
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	state.peerBitfields[peerID] = ctrl.Bitfield

	// Update piece rarity
	for i, has := range ctrl.Bitfield {
		if has && i < len(state.pieceRarity) {
			// Check if peer already counted
			alreadyCounted := false
			for _, p := range state.pieceRarity[i].HavingPeers {
				if p == peerID {
					alreadyCounted = true
					break
				}
			}
			if !alreadyCounted {
				state.pieceRarity[i].PeerCount++
				state.pieceRarity[i].HavingPeers = append(state.pieceRarity[i].HavingPeers, peerID)
			}
		}
	}
}

// handleHave processes a HAVE message (peer announcing they have a specific piece)
func (c *Client) handleHave(ctrl controlMessage, peer *webRTC.SimpleWebRTCPeer) {
	peerID := peer.GetSignalingStream().Conn().RemotePeer()

	c.downloadsMux.RLock()
	state, ok := c.activeDownloads[ctrl.CID]
	c.downloadsMux.RUnlock()

	if !ok {
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	pieceIdx := int(ctrl.HavePiece)
	if pieceIdx >= 0 && pieceIdx < len(state.pieceRarity) {
		// Update bitfield
		if state.peerBitfields[peerID] == nil {
			state.peerBitfields[peerID] = make([]bool, state.TotalPieces)
		}
		if !state.peerBitfields[peerID][pieceIdx] {
			state.peerBitfields[peerID][pieceIdx] = true
			state.pieceRarity[pieceIdx].PeerCount++
			state.pieceRarity[pieceIdx].HavingPeers = append(state.pieceRarity[pieceIdx].HavingPeers, peerID)
		}
	}
}

func (c *Client) onWebRTCPeerClose(peerID peer.ID) {
	logger.Info().Msgf("WebRTC peer disconnected: %s", peerID)
	c.peersMux.Lock()
	delete(c.webRTCPeers, peerID)
	c.peersMux.Unlock()

	// Handle download resumption logic
	c.downloadsMux.Lock()
	defer c.downloadsMux.Unlock()

	for cid, state := range c.activeDownloads {
		for pieceIndex, assignee := range state.PieceAssignees {
			if assignee == peerID {
				logger.Info().Msgf("Peer %s disconnected, re-requesting piece %d for download %s", peerID, pieceIndex, cid)
				// Re-queue the piece for download
				go c.reRequestPiece(state, pieceIndex)
			}
		}
	}
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (c *Client) monitorCongestion() {
	ticker := time.NewTicker(PingInterval)
	for range ticker.C {
		c.peersMux.RLock()
		for pid, peer := range c.webRTCPeers {
			if connState := peer.GetConnectionState(); connState == webRTC.ConnectionStateConnected {
				c.pingTimes[pid] = time.Now()
				ping := map[string]string{"type": "ping"}
				peer.SendJSONReliable(ping)
			}
		}
		c.peersMux.RUnlock()
	}
}

func (c *Client) handlePong(pid peer.ID) {
	if start, ok := c.pingTimes[pid]; ok {
		rtt := time.Since(start)
		c.rttMux.Lock()
		if _, ok := c.rttMeasurements[pid]; !ok {
			c.rttMeasurements[pid] = []time.Duration{}
		}
		c.rttMeasurements[pid] = append(c.rttMeasurements[pid], rtt)
		if len(c.rttMeasurements[pid]) > 10 {
			c.rttMeasurements[pid] = c.rttMeasurements[pid][1:]
		}
		avgRTT := time.Duration(0)
		for _, d := range c.rttMeasurements[pid] {
			avgRTT += d
		}
		avgRTT /= time.Duration(len(c.rttMeasurements[pid]))
		var delay time.Duration = MinDelay
		if avgRTT > MaxRTT {
			delay = MaxDelay
		} else if avgRTT > MaxRTT/2 {
			delay = (MaxDelay - MinDelay) / 2
		}
		c.congestionCtrl[pid] = delay
		c.rttMux.Unlock()
		delete(c.pingTimes, pid)
	}
}

// GetUploadStats returns real-time upload statistics
func (c *Client) GetUploadStats() map[string]*UploadStats {
	c.uploadProgressMu.RLock()
	defer c.uploadProgressMu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*UploadStats)
	for k, v := range c.uploadProgress {
		v.mu.Lock()
		statsCopy := &UploadStats{
			CID:           v.CID,
			BytesUploaded: v.BytesUploaded,
			ChunksServed:  v.ChunksServed,
			PeersServed:   make(map[peer.ID]int64),
			StartTime:     v.StartTime,
			LastActivity:  v.LastActivity,
		}
		for pid, bytes := range v.PeersServed {
			statsCopy.PeersServed[pid] = bytes
		}
		v.mu.Unlock()
		result[k] = statsCopy
	}
	return result
}

// ShowSeedingProgress displays current seeding/upload progress
func (c *Client) ShowSeedingProgress() {
	ctx := context.Background()

	fmt.Println("\n=== Seeding Progress ===")

	// Get real-time stats
	stats := c.GetUploadStats()
	if len(stats) == 0 {
		// Check database for historical stats
		totalBytes, totalChunks, uniquePeers, err := c.db.GetTotalUploadStats(ctx)
		if err != nil || totalBytes == 0 {
			fmt.Println("No upload activity yet.")
			return
		}
		fmt.Printf("Historical Upload Stats:\n")
		fmt.Printf("  Total Uploaded: %s\n", humanize.Bytes(uint64(totalBytes)))
		fmt.Printf("  Chunks Served: %d\n", totalChunks)
		fmt.Printf("  Unique Peers Served: %d\n", uniquePeers)
		return
	}

	// Display real-time stats
	var totalBytes int64
	var totalChunks int
	uniquePeers := make(map[peer.ID]bool)

	for cid, s := range stats {
		s.mu.Lock()
		fmt.Printf("\nCID: %s...\n", cid[:20])
		fmt.Printf("  Uploaded: %s\n", humanize.Bytes(uint64(s.BytesUploaded)))
		fmt.Printf("  Chunks Served: %d\n", s.ChunksServed)
		fmt.Printf("  Peers Served: %d\n", len(s.PeersServed))

		if !s.LastActivity.IsZero() {
			duration := s.LastActivity.Sub(s.StartTime)
			if duration > 0 {
				speed := float64(s.BytesUploaded) / duration.Seconds()
				fmt.Printf("  Avg Speed: %s/s\n", humanize.Bytes(uint64(speed)))
			}
		}

		totalBytes += s.BytesUploaded
		totalChunks += s.ChunksServed
		for pid := range s.PeersServed {
			uniquePeers[pid] = true
		}
		s.mu.Unlock()
	}

	fmt.Println("\n--- Total ---")
	fmt.Printf("  Total Uploaded: %s\n", humanize.Bytes(uint64(totalBytes)))
	fmt.Printf("  Total Chunks: %d\n", totalChunks)
	fmt.Printf("  Unique Peers: %d\n", len(uniquePeers))

	// Show upload rate limit if set
	if c.maxUploadRate > 0 {
		fmt.Printf("  Rate Limit: %s/s\n", humanize.Bytes(uint64(c.maxUploadRate)))
	}
}

// ShowStats displays combined download and upload statistics
func (c *Client) ShowStats() {
	ctx := context.Background()

	fmt.Println("\n╔══════════════════════════════════════╗")
	fmt.Println("║        TORRENTIUM STATISTICS         ║")
	fmt.Println("╠══════════════════════════════════════╣")

	// Upload stats
	totalUpBytes, totalUpChunks, upPeers, _ := c.db.GetTotalUploadStats(ctx)

	// Real-time upload stats
	rtStats := c.GetUploadStats()
	for _, s := range rtStats {
		s.mu.Lock()
		totalUpBytes += s.BytesUploaded
		totalUpChunks += s.ChunksServed
		s.mu.Unlock()
	}

	fmt.Println("║ UPLOADS (Seeding)                    ║")
	fmt.Printf("║   Total Uploaded: %-18s ║\n", humanize.Bytes(uint64(totalUpBytes)))
	fmt.Printf("║   Chunks Served:  %-18d ║\n", totalUpChunks)
	fmt.Printf("║   Peers Served:   %-18d ║\n", upPeers)

	// Download stats
	downloads, _ := c.db.GetDownloads(ctx)
	var totalDownBytes int64
	for _, d := range downloads {
		totalDownBytes += d.FileSize
	}

	fmt.Println("╠══════════════════════════════════════╣")
	fmt.Println("║ DOWNLOADS                            ║")
	fmt.Printf("║   Total Downloaded: %-16s ║\n", humanize.Bytes(uint64(totalDownBytes)))
	fmt.Printf("║   Files Downloaded: %-16d ║\n", len(downloads))

	// Ratio
	fmt.Println("╠══════════════════════════════════════╣")
	var ratio float64
	if totalDownBytes > 0 {
		ratio = float64(totalUpBytes) / float64(totalDownBytes)
	}
	fmt.Printf("║ RATIO: %-30.2f ║\n", ratio)

	// Shared files
	localFiles, _ := c.db.GetLocalFiles(ctx)
	fmt.Println("╠══════════════════════════════════════╣")
	fmt.Printf("║ Files Shared: %-22d ║\n", len(localFiles))

	fmt.Println("╚══════════════════════════════════════╝")
}
