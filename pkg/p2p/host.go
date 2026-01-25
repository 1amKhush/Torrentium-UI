package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/1amkhush/torrentium/pkg/config"
	"github.com/1amkhush/torrentium/pkg/logger"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	relayv2client "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
)

func reserveWithRelay(ctx context.Context, relayAddrStr string, h host.Host) error {
	maddr, err := ma.NewMultiaddr(relayAddrStr)
	if err != nil {
		return fmt.Errorf("invalid relay multiaddr: %w", err)
	}
	relayInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("could not parse relay AddrInfo: %w", err)
	}
	if err := h.Connect(ctx, *relayInfo); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}
	res, err := relayv2client.Reserve(ctx, h, *relayInfo)
	if err != nil {
		return fmt.Errorf("reservation failed: %w", err)
	}
	logger.Info().
		Str("relay", relayAddrStr).
		Time("expires", res.Expiration).
		Msg("Reservation with relay successful")
	return nil
}

// NewHost creates a new libp2p host with the given configuration.
// If cfg is nil, it uses the global configuration.
func NewHost(
	ctx context.Context,
	cfg *config.Config,
	onOffer func(offer, remotePeerID string, s network.Stream) (string, error),
) (host.Host, *dht.IpfsDHT, error) {
	if cfg == nil {
		cfg = config.Global()
	}

	log := logger.WithComponent("p2p")

	// Identity key
	priv, err := loadOrGeneratePrivateKey(cfg.P2P.PrivateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load/generate private key: %w", err)
	}

	// Local listen address
	maddr, err := ma.NewMultiaddr(cfg.P2P.ListenAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse listen address '%s': %w", cfg.P2P.ListenAddress, err)
	}

	// Get relay addresses (dynamically if configured)
	relayAddresses, err := GetRelayAddresses()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get relay addresses: %w", err)
	}
	if len(relayAddresses) == 0 {
		return nil, nil, fmt.Errorf("no relay addresses available")
	}
	relayAddrStr := relayAddresses[0]
	log.Info().Str("relay", relayAddrStr).Msg("Using relay address")

	relayMaddr, err := ma.NewMultiaddr(relayAddrStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid relay multiaddr: %w", err)
	}
	relayInfo, err := peer.AddrInfoFromP2pAddr(relayMaddr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid relay peer info: %w", err)
	}

	// Create host with relay + autorelay
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrs(maddr),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{*relayInfo}),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("libp2p peer not initialized: %w", err)
	}

	// Connect to relay
	if err := h.Connect(ctx, *relayInfo); err != nil {
		h.Close()
		return nil, nil, fmt.Errorf("failed to connect to relay: %w", err)
	}
	log.Info().Msg("Connected to relay")

	// Reserve relay slot
	if err := reserveWithRelay(ctx, relayAddrStr, h); err != nil {
		h.Close()
		return nil, nil, fmt.Errorf("relay reservation failed: %w", err)
	}
	log.Info().Msg("Relay reservation successful")

	// DHT setup
	idht, err := dht.New(ctx, h)
	if err != nil {
		h.Close()
		return nil, nil, fmt.Errorf("failed to initialize DHT: %w", err)
	}
	if err := idht.Bootstrap(ctx); err != nil {
		h.Close()
		return nil, nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Register WebRTC signaling protocol (your handler)
	RegisterSignalingProtocol(h, onOffer)

	log.Info().
		Str("peer_id", h.ID().String()).
		Msg("Host created")
	for _, addr := range h.Addrs() {
		log.Info().
			Str("address", fmt.Sprintf("%s/p2p/%s", addr, h.ID())).
			Msg("Listening")
	}

	return h, idht, nil
}

// NewHostWithDefaults creates a host using the default listen address and global config
func NewHostWithDefaults(
	ctx context.Context,
	onOffer func(offer, remotePeerID string, s network.Stream) (string, error),
) (host.Host, *dht.IpfsDHT, error) {
	return NewHost(ctx, nil, onOffer)
}

func Bootstrap(ctx context.Context, h host.Host, d *dht.IpfsDHT) error {
	return BootstrapWithConfig(ctx, h, d, nil)
}

// BootstrapWithConfig bootstraps the DHT with the given configuration
func BootstrapWithConfig(ctx context.Context, h host.Host, d *dht.IpfsDHT, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.Global()
	}

	log := logger.WithComponent("bootstrap")
	bootstrapNodes := cfg.P2P.BootstrapNodes
	minRequired := cfg.P2P.MinBootstrapConnections
	timeout := cfg.P2P.BootstrapTimeout

	log.Info().Msg("Connecting to bootstrap nodes...")
	connected := 0

	for i, addrStr := range bootstrapNodes {
		// Stop early if we have enough connections
		if connected >= minRequired {
			log.Info().Int("count", connected).Msg("Sufficient bootstrap connections established")
			break
		}

		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			log.Warn().Err(err).Str("address", addrStr).Msg("Invalid bootstrap address")
			continue
		}

		pi, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Warn().Err(err).Str("address", addrStr).Msg("Failed to parse bootstrap peer info")
			continue
		}

		connectCtx, cancel := context.WithTimeout(ctx, timeout)
		if err := h.Connect(connectCtx, *pi); err != nil {
			log.Debug().Err(err).Str("peer_id", pi.ID.String()).Msg("Failed to connect to bootstrap node")
		} else {
			log.Info().Str("peer_id", pi.ID.String()).Msg("Connected to bootstrap node")
			connected++
		}
		cancel()

		// Small delay between connections to avoid overwhelming
		if i < len(bootstrapNodes)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	if connected < minRequired {
		return fmt.Errorf("insufficient bootstrap connections: got %d, need at least %d", connected, minRequired)
	}

	log.Info().
		Int("connected", connected).
		Int("required", minRequired).
		Msg("Bootstrap connections established")

	// Bootstrap the DHT
	log.Info().Msg("Bootstrapping DHT...")
	if err := d.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Wait for DHT to become ready with better feedback
	log.Info().Msg("Waiting for DHT to become ready...")
	readyTimeout := time.After(45 * time.Second)
	checkTicker := time.NewTicker(5 * time.Second)
	defer checkTicker.Stop()

	for {
		select {
		case <-readyTimeout:
			routingTableSize := d.RoutingTable().Size()
			if routingTableSize > 0 {
				log.Info().Int("routing_table_size", routingTableSize).Msg("DHT partially ready, continuing")
			} else {
				log.Warn().Msg("DHT bootstrap timeout, continuing anyway")
			}
			return nil

		case <-checkTicker.C:
			routingTableSize := d.RoutingTable().Size()
			log.Debug().Int("routing_table_size", routingTableSize).Msg("DHT routing table status")
			if routingTableSize >= 10 {
				log.Info().Int("routing_table_size", routingTableSize).Msg("DHT is ready")
				return nil
			}

		case <-d.RefreshRoutingTable():
			routingTableSize := d.RoutingTable().Size()
			log.Debug().Int("routing_table_size", routingTableSize).Msg("DHT routing table refreshed")
			if routingTableSize >= 5 {
				log.Info().Msg("DHT is ready")
				return nil
			}
		}
	}
}

func loadOrGeneratePrivateKey(keyPath string) (crypto.PrivKey, error) {
	if keyPath == "" {
		keyPath = "private_key"
	}

	log := logger.WithComponent("crypto")

	privBytes, err := os.ReadFile(keyPath)
	if os.IsNotExist(err) {
		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}

		privBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal private key: %w", err)
		}

		if err := os.WriteFile(keyPath, privBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to write private key to file: %w", err)
		}

		log.Info().Str("path", keyPath).Msg("Generated new libp2p private key")
		return priv, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	log.Info().Str("path", keyPath).Msg("Loaded existing libp2p private key")
	return crypto.UnmarshalPrivateKey(privBytes)
}
