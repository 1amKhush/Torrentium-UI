package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/1amkhush/torrentium/pkg/config"
	db "github.com/1amkhush/torrentium/pkg/db"
	"github.com/1amkhush/torrentium/pkg/logger"
	p2p "github.com/1amkhush/torrentium/pkg/p2p"
	"github.com/1amkhush/torrentium/pkg/torrentium_client"
)

// The commandLoop now takes the client as an argument
func commandLoop(c *torrentium_client.Client) {
	scanner := bufio.NewScanner(os.Stdin)
	printInstructions() // You can move printInstructions to this file as well
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
			printInstructions()
		case "add":
			if len(args) != 1 {
				fmt.Println("Usage: add <path>")
				fmt.Println("  <path> can be a file or directory")
			} else {
				// Call the library function
				err = c.AddFile(args[0])
			}
		case "list":
			c.ListLocalFiles()
		case "search":
			if len(args) != 1 {
				fmt.Println("Usage: search <cid|text>")
			} else {
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
		case "health":
			c.CheckConnectionHealth()
		case "debug":
			c.DebugNetworkStatus()
		case "stats":
			c.ShowStats()
		case "seeding":
			c.ShowSeedingProgress()
		case "maxupload":
			if len(args) != 1 {
				fmt.Println("Usage: maxupload <bytes-per-second|unlimited>")
				fmt.Println("  Example: maxupload 1048576  (1 MB/s)")
				fmt.Println("  Example: maxupload unlimited")
			} else {
				if args[0] == "unlimited" || args[0] == "0" {
					c.SetMaxUploadRate(0)
				} else {
					var rate int64
					_, parseErr := fmt.Sscanf(args[0], "%d", &rate)
					if parseErr != nil {
						fmt.Println("Invalid rate. Use a number (bytes/sec) or 'unlimited'")
					} else {
						c.SetMaxUploadRate(rate)
					}
				}
			}
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

func printInstructions() {
	fmt.Println("\nAvailable Commands:")
	fmt.Println("  add <path>       - Share a file or directory on the network")
	fmt.Println("  list             - List your shared files")
	fmt.Println("  search <cid|text>- Search by CID or filename text")
	fmt.Println("  download <cid>   - Download a file by CID (uses rarest-first)")
	fmt.Println("  peers            - Show connected peers")
	fmt.Println("  stats            - Show upload/download statistics and ratio")
	fmt.Println("  seeding          - Show seeding (upload) progress")
	fmt.Println("  maxupload <rate> - Set max upload rate (bytes/sec or 'unlimited')")
	fmt.Println("  health           - Check connection health")
	fmt.Println("  debug            - Show detailed network debug info")
	fmt.Println("  help             - Show this help")
	fmt.Println("  exit             - Exit the application")
}

func main() {
	// Initialize configuration
	cfg, err := config.LoadOrDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	config.SetGlobal(cfg)

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:        cfg.Logging.Level,
		Format:       cfg.Logging.Format,
		Output:       cfg.Logging.Output,
		EnableCaller: cfg.Logging.EnableCaller,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log := logger.WithComponent("main")
	log.Info().Msg("Starting Torrentium...")

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize database
	database, err := db.InitDB(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Database initialization failed")
	}
	defer database.Close()

	// Create P2P host
	h, d, err := p2p.NewHost(ctx, cfg, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create libp2p host")
	}
	defer h.Close()

	// Bootstrap DHT in background
	go func() {
		if err := p2p.Bootstrap(ctx, h, d); err != nil {
			log.Error().Err(err).Msg("Error bootstrapping DHT")
		}
	}()

	// Create repository and client
	repo := db.NewRepository(database)
	client := torrentium_client.NewClient(h, d, repo)

	// Register signaling protocol with the client's WebRTC offer handler
	p2p.RegisterSignalingProtocol(h, client.HandleWebRTCOffer)

	// Start background tasks
	client.StartDHTMaintenance()

	// Setup graceful shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	// Run command loop in background
	done := make(chan struct{})
	go func() {
		commandLoop(client)
		close(done)
	}()

	// Wait for shutdown signal or command loop exit
	select {
	case sig := <-shutdownCh:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	case <-done:
		log.Info().Msg("Command loop exited")
	}

	// Graceful shutdown
	log.Info().Msg("Shutting down gracefully...")
	cancel() // Cancel context to stop background operations

	// Close client (closes WebRTC peers and active downloads)
	if err := client.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing client")
	}

	// Close host
	if err := h.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing host")
	}

	log.Info().Msg("Shutdown complete")
}
