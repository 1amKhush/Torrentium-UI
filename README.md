# Torrentium - Decentralized P2P File Sharing System

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/doc/install)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20|%20macOS%20|%20Linux-lightgrey.svg)]()

## Overview

Torrentium is a **fully decentralized peer-to-peer file sharing system** with a modern desktop GUI. Built with Go backend and Vue.js frontend using the Wails framework, it eliminates the need for centralized tracking servers. Unlike traditional BitTorrent systems that rely on trackers, Torrentium uses **libp2p's Distributed Hash Table (DHT)** for peer discovery and **WebRTC** for direct peer-to-peer data transfer.

**🎯 Just download the `.exe` and start sharing files - no installation required!**

## 🌟 Key Features

### Desktop Application

- **Standalone executable** - single `.exe` file, no installation needed
- **Modern GUI** - beautiful Vue.js interface with dark theme
- **Cross-platform** - Windows, macOS, and Linux support
- **Real-time updates** - live connection status and transfer progress

### Decentralized Architecture

- **No central servers required** - fully peer-to-peer operation
- **DHT-based peer discovery** using Kademlia algorithm via libp2p
- **Resilient network topology** - no single point of failure
- **Relay server support** for NAT traversal

### Advanced Connectivity

- **WebRTC data channels** for high-performance direct peer communication
- **Multiple STUN/TURN servers** for reliable NAT traversal
- **Automatic bootstrapping** to public IPFS nodes
- **Connection health monitoring** and automatic recovery

### File Management

- **Content-addressable storage** using IPFS CIDs (Content Identifiers)
- **Chunked file transfer** with configurable piece sizes (default: 1MB)
- **Resume capability** through piece-based downloads
- **SHA-256 integrity verification** for all file transfers
- **Progress tracking** with visual progress indicators

### Local Database

- **SQLite-based persistence** for metadata and download history
- **Peer reputation system** with scoring mechanism
- **File indexing** for fast local searches
- **Download resume state** tracking

## 🏗️ Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Torrentium Desktop App                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Vue.js Frontend                               │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌──────────┐ ┌────────┐  │   │
│  │  │ Dashboard │ │   Files   │ │ Downloads │ │  Stats   │ │Settings│  │   │
│  │  └───────────┘ └───────────┘ └───────────┘ └──────────┘ └────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              ▲                                              │
│                              │ Wails Bindings                               │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Go Backend                                   │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐  │   │
│  │  │ App (Wails) │  │   Config    │  │   Logger    │  │  Database  │  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘  │   │
│  │         │                                                  │         │   │
│  │         ▼                                                  ▼         │   │
│  │  ┌─────────────────────────────────────────────────────────────┐    │   │
│  │  │              Torrentium Client (Core Engine)                │    │   │
│  │  │  • File Management    • DHT Operations    • Peer Discovery  │    │   │
│  │  │  • Download/Upload    • CID Generation    • Health Monitor  │    │   │
│  │  └─────────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           P2P Network Layer                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐  │
│  │   libp2p Host   │    │   Relay Server  │    │   Bootstrap Nodes       │  │
│  │  • DHT Routing  │◄──►│  • NAT Traversal│◄──►│  • IPFS Public Nodes   │  │
│  │  • Peer Connect │    │  • Hole Punch   │    │  • DHT Bootstrap       │  │
│  └─────────────────┘    └─────────────────┘    └─────────────────────────┘  │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    WebRTC Data Channels                             │   │
│  │              Direct P2P File Transfer (Encrypted)                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Network Stack

1. **Transport Layer**: TCP/WebSocket for libp2p communication
2. **P2P Layer**: libp2p for peer discovery and connection management
3. **DHT Layer**: Kademlia DHT for content and peer routing
4. **Data Layer**: WebRTC data channels for file transfer
5. **Application Layer**: Wails GUI and file management

## 🚀 Quick Start

### Option 1: Download Pre-built Executable (Recommended)

1. **Download** `Torrentium.exe` from the [Releases](https://github.com/ArunCS1005/Torrentium/releases) page
2. **Run** the executable - no installation required!
3. **Start sharing** files immediately

> **Note**: On first run, a `private_key` file will be generated in the same folder as the exe. This is your unique peer identity.

### Option 2: Build from Source

#### Prerequisites

- **Go 1.23+** ([Download](https://go.dev/dl/))
- **Node.js 18+** ([Download](https://nodejs.org/))
- **Wails CLI** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **64-bit MinGW** (Windows only, for CGO support)

#### Build Steps

1. **Clone the repository**:

```bash
git clone https://github.com/ArunCS1005/Torrentium.git
cd Torrentium
```

2. **Install dependencies**:

```bash
cd gui
go mod tidy
cd frontend && npm install && cd ..
```

3. **Build the application**:

```bash
# Windows (64-bit)
wails build -platform windows/amd64

# macOS
wails build -platform darwin/universal

# Linux
wails build -platform linux/amd64
```

4. **Run the built executable**:

```bash
# Windows
./build/bin/Torrentium.exe

# macOS/Linux
./build/bin/Torrentium
```

#### Development Mode

Run with hot-reload for development:

```bash
cd gui
wails dev
```

### CLI Client (Legacy)

A command-line client is also available:

```bash
go build -o torrentium ./cmd/CLIENT/
./torrentium
```

## 📖 Usage Guide

### GUI Application

The desktop application provides an intuitive interface with five main sections:

#### 📊 Dashboard

- View your **Peer ID** and connection status
- Monitor **connected peers** count
- Check **relay connection** status
- See **network health** indicators

#### 📁 Files (Share)

- **Share File**: Select any file to share on the network
- **Share Directory**: Share entire folders
- **Copy CID**: Copy the Content Identifier to share with others
- **Re-announce**: Refresh your file availability on the DHT

#### ⬇️ Downloads

- **Download by CID**: Paste a CID to download from the network
- **Search**: Find files by name across the network
- **Download History**: View all completed downloads
- **Progress Tracking**: Real-time download progress

#### 📈 Stats

- **Upload Statistics**: Total data uploaded
- **Download Statistics**: Total data downloaded
- **Share Ratio**: Upload/download ratio
- **Transfer History**: Recent transfers

#### ⚙️ Settings

- **Download Directory**: Configure where files are saved
- **Upload Rate Limit**: Control bandwidth usage
- **DHT Refresh Interval**: Adjust network refresh rate

### CLI Commands (Legacy Client)

```
> add /path/to/your/file.txt
✓ File 'file.txt' is now being shared
 CID: bafybeig...(generated hash)
 Hash: a1b2c3...
 Size: 1.2 MB
```

#### Listing Shared Files

```
> list
=== Your Shared Files ===
Name: file.txt
 CID: bafybeig...
 Size: 1.2 MB
 Path: /path/to/your/file.txt
```

#### Searching for Files

```
# Search by filename
> search "document"
Local index matches for 'document':
- document.pdf  CID:bafybeig...

# Search by exact CID
> search bafybeig...
Found 3 provider(s):
 1. 12D3KooW... - Connected
 2. 12D3KooW... - Not connected
```

#### Downloading Files

```
> download bafybeig...
Looking for providers of CID: bafybeig...
Found 2 provider(s). Attempting connections...
Downloading to bafybeig....download...
Download complete!
```

#### Network Management

```
# View connected peers
> peers
=== Connected Peers (5) ===
Peer: 12D3KooW...
 Address: /ip4/192.168.1.100/tcp/4001

# Connect to specific peer
> connect /ip4/127.0.0.1/tcp/54437/p2p/12D3KooW...

# Check network health
> health
=== Connection Health ===
Connected peers: 8
 - Good peer connectivity
DHT routing table size: 45
 - Good DHT connectivity
```

### Advanced Features

#### Manual File Announcement

```
> announce bafybeig...
Re-announcing CID bafybeig... to DHT...
 - Successfully announced to DHT
```

#### Debug Information

```
> debug
=== Network Debug Info ===
Our Peer ID: 12D3KooW...
Our Addresses:
 /ip4/192.168.1.100/tcp/4001/p2p/12D3KooW...
 /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...

Connected Peers (8):
DHT Routing Table Size: 45
Shared Files (3):
```

## 🔧 Technical Deep Dive

### File Processing Pipeline

1. **File Addition**:

   - File is read and SHA-256 hash calculated
   - Content is chunked into 1MB pieces
   - Each piece hash is stored in SQLite
   - IPFS CID is generated using multihash
   - File metadata announced to DHT

2. **Peer Discovery**:

   - DHT lookup for content providers
   - Connection establishment via libp2p
   - WebRTC negotiation through signaling protocol
   - Peer reputation scoring

3. **Data Transfer**:
   - WebRTC data channel establishment
   - Chunked transfer with progress tracking
   - Real-time integrity verification
   - Resume capability for interrupted downloads

### Database Schema

The SQLite database maintains several key tables:

```sql
-- File metadata for shared content
CREATE TABLE local_files (
    id TEXT PRIMARY KEY,
    cid TEXT UNIQUE NOT NULL,
    filename TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Download history and state
CREATE TABLE downloads (
    id TEXT PRIMARY KEY,
    cid TEXT UNIQUE NOT NULL,
    filename TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    download_path TEXT NOT NULL,
    downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'completed'
);

-- Piece-level tracking for resume capability
CREATE TABLE pieces (
    id TEXT PRIMARY KEY,
    cid TEXT NOT NULL,
    idx INTEGER NOT NULL,
    offset INTEGER NOT NULL,
    size INTEGER NOT NULL,
    hash TEXT NOT NULL,
    have INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (cid, idx)
);

-- Peer reputation system
CREATE TABLE peer_scores (
    peer_id TEXT PRIMARY KEY,
    score REAL NOT NULL,
    seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### WebRTC Integration

Torrentium uses WebRTC data channels for efficient peer-to-peer communication:

- **ICE servers**: Multiple STUN/TURN servers for NAT traversal
- **Signaling**: Custom libp2p protocol for WebRTC offer/answer exchange
- **Data transfer**: Binary data channels for file content
- **Control messages**: JSON messages for file requests and metadata

## 🔗 Dependencies

### Core Libraries

- **[go-libp2p](https://github.com/libp2p/go-libp2p)**: P2P networking framework
- **[go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht)**: Kademlia DHT implementation
- **[pion/webrtc](https://github.com/pion/webrtc)**: WebRTC implementation in Go
- **[go-sqlite3](https://github.com/mattn/go-sqlite3)**: SQLite database driver

### Utility Libraries

- **[go-cid](https://github.com/ipfs/go-cid)**: Content Identifier implementation
- **[go-multihash](https://github.com/multiformats/go-multihash)**: Multihash support
- **[progressbar](https://github.com/schollz/progressbar)**: CLI progress visualization
- **[humanize](https://github.com/dustin/go-humanize)**: Human-readable file sizes

### 🛠️ Development

### Project Structure

```
torrentium/
├── cmd/CLIENT/              # Legacy CLI client
│   └── main.go             # CLI interface
├── pkg/                     # Core library packages
│   ├── client/             # WebRTC client implementation
│   │   └── webrtc.go      # WebRTC peer management
│   ├── config/            # Configuration management
│   │   └── config.go      # App settings and defaults
│   ├── db/                # Database layer
│   │   └── db.go          # SQLite operations and schema
│   ├── logger/            # Logging utilities
│   │   └── logger.go      # Structured logging
│   ├── p2p/               # P2P networking
│   │   ├── host.go        # libp2p host creation
│   │   └── signaling.go   # WebRTC signaling protocol
│   └── torrentium_client/ # Core client engine
│       └── client.go      # Main client logic
├── gui/                    # Wails desktop application
│   ├── app.go             # Go backend bindings for GUI
│   ├── main.go            # Wails app entry point
│   ├── wails.json         # Wails configuration
│   ├── build/             # Build outputs
│   │   └── bin/           # Compiled executables
│   └── frontend/          # Vue.js frontend
│       ├── index.html     # HTML entry point
│       ├── package.json   # Node dependencies
│       ├── vite.config.js # Vite bundler config
│       └── src/
│           ├── App.vue    # Main Vue component
│           ├── main.js    # Vue entry point
│           ├── style.css  # Global styles
│           └── components/
│               ├── Dashboard.vue   # Dashboard view
│               ├── Files.vue       # File sharing view
│               ├── Downloads.vue   # Downloads view
│               ├── Stats.vue       # Statistics view
│               ├── Settings.vue    # Settings view
│               └── Sidebar.vue     # Navigation sidebar
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── private_key            # libp2p identity (generated)
├── peer.db                # SQLite database (generated)
└── README.md              # This file
```

### Building and Testing

```bash
# Build the GUI application
cd gui
wails build -platform windows/amd64

# Run in development mode (hot-reload)
wails dev

# Build CLI client (legacy)
go build -o torrentium ./cmd/CLIENT/
```

### Technology Stack

| Component              | Technology                                                   |
| ---------------------- | ------------------------------------------------------------ |
| **Desktop Framework**  | [Wails v2](https://wails.io/)                                |
| **Frontend**           | [Vue.js 3](https://vuejs.org/) + [Vite](https://vitejs.dev/) |
| **Backend**            | [Go 1.23+](https://go.dev/)                                  |
| **P2P Networking**     | [libp2p](https://github.com/libp2p/go-libp2p)                |
| **DHT**                | [Kademlia DHT](https://github.com/libp2p/go-libp2p-kad-dht)  |
| **Data Transfer**      | [Pion WebRTC](https://github.com/pion/webrtc)                |
| **Database**           | [SQLite3](https://github.com/mattn/go-sqlite3)               |
| **Content Addressing** | [IPFS CID](https://github.com/ipfs/go-cid)                   |

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 🔒 Security Considerations

- **Content Integrity**: All files verified using SHA-256 hashing
- **Peer Authentication**: libp2p cryptographic identities
- **NAT Traversal**: Secure STUN/TURN server usage
- **Local Storage**: SQLite database with appropriate file permissions
- **Network Security**: Encrypted WebRTC data channels

## 🗺️ Roadmap

- [x] **Desktop GUI** - Modern Wails-based desktop application
- [x] **Cross-platform builds** - Windows, macOS, Linux support
- [x] **Relay server integration** - NAT traversal support
- [ ] **Auto-updates** - Automatic application updates
- [ ] **Mobile Support** - Android/iOS client applications
- [ ] **Improved Search** - Full-text search across file contents
- [ ] **Bandwidth Control** - Rate limiting and QoS features
- [ ] **Plugin System** - Extensible architecture for custom protocols
- [ ] **Analytics Dashboard** - Network statistics and performance metrics

## 🐛 Troubleshooting

### Common Issues

1. **"Relay connection failed"**: The relay server may have restarted with a new peer ID. Check for updates or report the issue.
2. **No peers found**: Check internet connection and firewall settings. Allow the app through Windows Firewall.
3. **Download failures**: Verify CID format and ensure the provider is online.
4. **Window closes immediately**: Ensure you're running the 64-bit version and have WebView2 installed.
5. **Database errors**: Ensure write permissions in the application directory.
6. **Build errors on Windows**: Make sure you have 64-bit MinGW installed (not 32-bit).

### System Requirements

| Requirement | Minimum                                     |
| ----------- | ------------------------------------------- |
| **OS**      | Windows 10/11 (64-bit), macOS 10.15+, Linux |
| **Runtime** | WebView2 (pre-installed on Windows 10/11)   |
| **RAM**     | 512 MB                                      |
| **Storage** | 100 MB + space for downloads                |
| **Network** | Internet connection required                |

### Debug Mode

The application logs are available in the console when running from terminal. Use the GUI Dashboard to check connection status and peer information.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **[Wails](https://wails.io/)** for the excellent Go + Web desktop framework
- **[Vue.js](https://vuejs.org/)** for the reactive frontend framework
- **[IPFS Project](https://ipfs.io/)** for content-addressable storage concepts
- **[libp2p Community](https://libp2p.io/)** for the robust P2P networking stack
- **[Pion WebRTC](https://github.com/pion/webrtc)** for excellent Go WebRTC implementation
- **Go Community** for the excellent ecosystem and libraries

## 📞 Support

For issues, questions, or contributions:

- **GitHub Issues**: [Create an issue](https://github.com/ArunCS1005/Torrentium/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ArunCS1005/Torrentium/discussions)

---

**Torrentium** - Decentralized file sharing made simple 🌐

![Torrentium Screenshot](docs/screenshot.png)
