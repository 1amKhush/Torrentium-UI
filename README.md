# Torrentium - Decentralized P2P File Sharing System

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/doc/install)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20|%20macOS%20|%20Linux-lightgrey.svg)]()
[![Last Updated](https://img.shields.io/badge/updated-February%202026-brightgreen.svg)]()

## Overview

Torrentium is a **fully decentralized peer-to-peer file sharing system** with a modern desktop GUI. Built with Go backend and Vue.js frontend using the Wails framework, it eliminates the need for centralized tracking servers. Unlike traditional BitTorrent systems that rely on trackers, Torrentium uses **libp2p's Distributed Hash Table (DHT)** for peer discovery and **WebRTC** for direct peer-to-peer data transfer.

**🎯 Just download the `.exe` and start sharing files - no installation required!**

## 📋 Table of Contents

- [Overview](#overview)
- [Key Features](#-key-features)
- [Architecture](#-architecture)
- [User Flow](#-user-flow)
- [Quick Start](#-quick-start)
- [Usage Guide](#-usage-guide)
- [Technical Deep Dive](#-technical-deep-dive)
- [Current Limitations & Challenges](#-current-limitations--challenges)
- [Future Scope](#-future-scope)
- [Dependencies](#-dependencies)
- [Development](#️-development)
- [Contributing](#contributing)
- [License](#-license)

## 🌟 Key Features

### Desktop Application

- **Standalone executable** - single `.exe` file, no installation needed
- **Modern GUI** - beautiful Vue.js interface with dark/light theme toggle
- **Cross-platform** - Windows, macOS, and Linux support
- **Real-time updates** - live connection status and transfer progress
- **File preview** - in-app preview for images, videos, audio, PDFs, and text files
- **Toast notifications** - real-time feedback for operations

### Decentralized Architecture

- **No central servers required** - fully peer-to-peer operation
- **DHT-based peer discovery** using Kademlia algorithm via libp2p
- **Resilient network topology** - no single point of failure
- **Multi-relay failover** - automatic failover to healthy relays with latency tracking

### Advanced Connectivity

- **WebRTC data channels** for high-performance direct peer communication
- **Dual data channels** - unreliable (fast) + reliable (ordered) channels
- **Connection pooling** - reuse connections with configurable pool size (default: 20)
- **Multiple STUN/TURN servers** for reliable NAT traversal
- **Automatic bootstrapping** to public IPFS nodes
- **Connection health monitoring** and automatic recovery
- **Hole punching** support for direct peer connections

### Smart Download Management

- **Download queue system** - priority-based queue (Low/Normal/High)
- **Pause/Resume/Cancel** - full control over active downloads
- **Per-download bandwidth limits** - control individual download speeds
- **Adaptive parallel downloads** - auto-adjusts based on bandwidth variance
- **Endgame mode** - requests final pieces from multiple peers for faster completion
- **Rarest-first piece selection** - BitTorrent-style smart piece prioritization
- **Download checkpointing** - periodic state saves for reliable resume

### File Management

- **Content-addressable storage** using IPFS CIDs (Content Identifiers)
- **Multi-file (directory) support** - share entire folders as single torrents
- **Chunked file transfer** with configurable piece sizes (default: 1MB)
- **Resume capability** through piece-based downloads
- **SHA-256 integrity verification** for all file transfers
- **Magnet link support** - generate and share `torrentium://` magnet links
- **Progress tracking** with visual progress indicators

### Upload & Seeding

- **Global upload rate limiting** - control total upload bandwidth
- **Per-CID upload statistics** - track bytes, chunks, and peers served
- **Real-time upload progress** tracking
- **Automatic re-announcement** to DHT

### Web Share Portal Integration

- **Publish files to web portal** - share files via web interface
- **Configurable visibility** - public or unlisted sharing
- **Expiration settings** - auto-expire shared files
- **API key authentication** - secure portal access

### Local Database

- **SQLite-based persistence** for metadata and download history
- **Peer reputation system** with scoring mechanism
- **File indexing** for fast local searches
- **Multi-file entry tracking** for directory torrents
- **Upload statistics** persistence

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
│  │  ┌─────────────────────┐ ┌─────────────────────┐ ┌───────────────┐  │   │
│  │  │  │    File Preview     │ │  Toast Notifications│ │  Theme Toggle │  │   │
│  │  └─────────────────────┘ └─────────────────────┘ └───────────────┘  │   │
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
│  │  ┌─────────────────────────────────────────────────────────────────┐    │   │
│  │  │              Torrentium Client (Core Engine)                │    │   │
│  │  │  • File Management    • DHT Operations    • Peer Discovery  │    │   │
│  │  │  • Download Queue     • Adaptive Downloads• Endgame Mode    │    │   │
│  │  │  • Upload Tracking    • WebShare Portal   • Magnet Links    │    │   │
│  │  └─────────────────────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           P2P Network Layer                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐  │
│  │   libp2p Host   │    │  Relay Failover │    │   Bootstrap Nodes       │  │
│  │  • DHT Routing  │◄──►│  • Multi-relay  │◄──►│  • IPFS Public Nodes   │  │
│  │  • Peer Connect │    │  • Health Check │    │  • DHT Bootstrap       │  │
│  │  • Hole Punch   │    │  • Latency Track│    │  • Auto Re-bootstrap   │  │
│  └─────────────────┘    └─────────────────┘    └─────────────────────────┘  │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    WebRTC Data Channels                             │   │
│  │         Connection Pool • Dual Channels • Keep-Alive               │   │
│  │              Direct P2P File Transfer (Encrypted)                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              TORRENTIUM STACK                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌────────────────────────────────────────────────────────────────────┐    │
│   │                    PRESENTATION LAYER                              │    │
│   │  ┌──────────────────────────────────────────────────────────────┐  │    │
│   │  │  Vue.js 3 + Vite                                             │  │    │
│   │  │  • Dashboard Component   • File Management Component         │  │    │
│   │  │  • Downloads Component   • Statistics Component              │  │    │
│   │  │  • Settings Component    • Toast Notifications               │  │    │
│   │  │  • File Preview Modal    • Dark/Light Theme Support          │  │    │
│   │  └──────────────────────────────────────────────────────────────┘  │    │
│   └────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│                         Wails IPC Bridge                                     │
│                                    │                                         │
│   ┌────────────────────────────────────────────────────────────────────┐    │
│   │                    APPLICATION LAYER (Go)                          │    │
│   │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────┐   │    │
│   │  │   App.go       │  │  Config.go     │  │   Logger.go        │   │    │
│   │  │  (GUI Binds)   │  │  (Settings)    │  │   (Structured)     │   │    │
│   │  └────────────────┘  └────────────────┘  └────────────────────┘   │    │
│   └────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│   ┌────────────────────────────────────────────────────────────────────┐    │
│   │                    BUSINESS LOGIC LAYER                            │    │
│   │  ┌──────────────────────────────────────────────────────────────┐  │    │
│   │  │                Torrentium Client (Core)                      │  │    │
│   │  │  ┌───────────────┐  ┌───────────────┐  ┌─────────────────┐   │  │    │
│   │  │  │ Download Queue│  │ Adaptive Mgr  │  │  Endgame Mode   │   │  │    │
│   │  │  │ (Priority Q)  │  │ (Parallel Adj)│  │  (Fast Finish)  │   │  │    │
│   │  │  └───────────────┘  └───────────────┘  └─────────────────┘   │  │    │
│   │  │  ┌───────────────┐  ┌───────────────┐  ┌─────────────────┐   │  │    │
│   │  │  │ Checkpointer  │  │ Rarest-First  │  │  DHT Retry      │   │  │    │
│   │  │  │ (State Save)  │  │ (Piece Select)│  │  (With Backoff) │   │  │    │
│   │  │  └───────────────┘  └───────────────┘  └─────────────────┘   │  │    │
│   │  └──────────────────────────────────────────────────────────────┘  │    │
│   └────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│   ┌────────────────────────────────────────────────────────────────────┐    │
│   │                    DATA LAYER                                      │    │
│   │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────┐   │    │
│   │  │   SQLite DB    │  │  Repository    │  │   File System      │   │    │
│   │  │  (Metadata)    │  │  (CRUD Ops)    │  │   (Content)        │   │    │
│   │  └────────────────┘  └────────────────┘  └────────────────────┘   │    │
│   └────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│   ┌────────────────────────────────────────────────────────────────────┐    │
│   │                    NETWORK LAYER                                   │    │
│   │  ┌──────────────────────────────────────────────────────────────┐  │    │
│   │  │                    P2P Module                                │  │    │
│   │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │  │    │
│   │  │  │  libp2p Host │  │  DHT (Kad)   │  │  Relay Failover  │   │  │    │
│   │  │  │  (Identity)  │  │  (Discovery) │  │  (Multi-relay)   │   │  │    │
│   │  │  └──────────────┘  └──────────────┘  └──────────────────┘   │  │    │
│   │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │  │    │
│   │  │  │  Signaling   │  │  Hole Punch  │  │  Bootstrap       │   │  │    │
│   │  │  │  Protocol    │  │  Support     │  │  Manager         │   │  │    │
│   │  │  └──────────────┘  └──────────────┘  └──────────────────┘   │  │    │
│   │  └──────────────────────────────────────────────────────────────┘  │    │
│   │  ┌──────────────────────────────────────────────────────────────┐  │    │
│   │  │                    WebRTC Module                             │  │    │
│   │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │  │    │
│   │  │  │  Data Channel│  │  Connection  │  │  STUN/TURN       │   │  │    │
│   │  │  │  (Dual Mode) │  │  Pooling     │  │  ICE Servers     │   │  │    │
│   │  │  └──────────────┘  └──────────────┘  └──────────────────┘   │  │    │
│   │  └──────────────────────────────────────────────────────────────┘  │    │
│   └────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Network Stack

1. **Transport Layer**: TCP/WebSocket for libp2p communication
2. **P2P Layer**: libp2p for peer discovery and connection management
3. **DHT Layer**: Kademlia DHT for content and peer routing with retry/backoff
4. **Data Layer**: WebRTC data channels with connection pooling
5. **Application Layer**: Wails GUI, download queue, and file management

### Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DATA FLOW DIAGRAM                               │
└─────────────────────────────────────────────────────────────────────────┘

UPLOAD FLOW:
┌──────────┐    ┌────────────┐    ┌──────────┐    ┌────────────┐    ┌─────────┐
│  User    │───►│  GUI       │───►│  Client  │───►│  DHT       │───►│  Peers  │
│  Selects │    │  Frontend  │    │  Backend │    │  Announce  │    │  Can    │
│  File    │    │            │    │  Process │    │            │    │  Find   │
└──────────┘    └────────────┘    └──────────┘    └────────────┘    └─────────┘
                     │                 │
                     │                 ▼
                     │           ┌──────────┐
                     │           │  SQLite  │
                     │           │  Store   │
                     │           │  Meta    │
                     │           └──────────┘
                     │                 │
                     │                 ▼
                     │           ┌──────────┐
                     │           │  Generate│
                     │           │  CID     │
                     └──────────►│  Hash    │
                                 └──────────┘

DOWNLOAD FLOW:
┌──────────┐    ┌────────────┐    ┌──────────┐    ┌────────────┐    ┌─────────┐
│  User    │───►│  Download  │───►│  DHT     │───►│  Provider  │───►│ WebRTC  │
│  Enters  │    │  Queue     │    │  Lookup  │    │  Discovery │    │ Connect │
│  CID     │    │  (Priority)│    │          │    │            │    │         │
└──────────┘    └────────────┘    └──────────┘    └────────────┘    └─────────┘
                                                                         │
┌──────────┐    ┌────────────┐    ┌──────────┐    ┌────────────┐         │
│  File    │◄───│  Piece     │◄───│  Verify  │◄───│  Receive   │◄────────┘
│  Complete│    │  Assembly  │    │  SHA-256 │    │  Chunks    │
└──────────┘    └────────────┘    └──────────┘    └────────────┘
```

## 👤 User Flow

### First-Time User Experience

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    FIRST-TIME USER JOURNEY                                  │
└─────────────────────────────────────────────────────────────────────────────┘

1. DOWNLOAD & LAUNCH
   ┌─────────────────┐
   │  Download       │     No installation required!
   │  Torrentium.exe │────► Just double-click to run
   └─────────────────┘
            │
            ▼
2. AUTOMATIC SETUP (Behind the scenes)
   ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
   │  Generate       │────►│  Connect to     │────►│  Bootstrap      │
   │  Peer Identity  │     │  Relay Server   │     │  DHT Network    │
   │  (private_key)  │     │  (NAT Traverse) │     │  (IPFS Nodes)   │
   └─────────────────┘     └─────────────────┘     └─────────────────┘
            │
            ▼
3. READY TO USE
   ┌─────────────────────────────────────────────────────────────────────┐
   │                      DASHBOARD                                       │
   │   • View Peer ID          • Connection Status: Connected ✓          │
   │   • Connected Peers: 8    • Shared Files: 0                         │
   └─────────────────────────────────────────────────────────────────────┘
```

### Sharing a File (Upload Flow)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    FILE SHARING WORKFLOW                                    │
└─────────────────────────────────────────────────────────────────────────────┘

Step 1: Navigate to "Files" tab
        ┌────────────────────────────────────────┐
        │  📁 Files                              │
        │  ┌──────────────────────────────────┐  │
        │  │  [+ Share File] [+ Share Folder] │  │
        │  │  Or drag & drop files here       │  │
        │  └──────────────────────────────────┘  │
        └────────────────────────────────────────┘

Step 2: Select file → Processing begins
        ┌────────────────────────────────────────┐
        │  Processing "video.mp4"...             │
        │  ├─ Calculating SHA-256 hash           │
        │  ├─ Chunking into 1MB pieces           │
        │  ├─ Generating IPFS CID                │
        │  └─ Announcing to DHT network          │
        └────────────────────────────────────────┘

Step 3: File is now shared
        ┌────────────────────────────────────────┐
        │  ✓ File shared successfully!           │
        │  ┌──────────────────────────────────┐  │
        │  │  📹 video.mp4                    │  │
        │  │  CID: bafybeig...                │  │
        │  │  Size: 125.4 MB                  │  │
        │  │  [Copy CID] [Copy Magnet Link]   │  │
        │  └──────────────────────────────────┘  │
        └────────────────────────────────────────┘

Step 4: Share the CID or Magnet Link with others
        • CID: bafybeig2abc123...
        • Magnet: torrentium://bafybeig2abc123/video.mp4?size=131534336
```

### Downloading a File

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    FILE DOWNLOAD WORKFLOW                                   │
└─────────────────────────────────────────────────────────────────────────────┘

Step 1: Navigate to "Downloads" tab → Enter CID
        ┌────────────────────────────────────────┐
        │  ⬇️ Downloads                          │
        │  ┌──────────────────────────────────┐  │
        │  │  Enter CID: [bafybeig...       ] │  │
        │  │  [Download]                      │  │
        │  └──────────────────────────────────┘  │
        └────────────────────────────────────────┘

Step 2: System finds providers and starts download
        ┌────────────────────────────────────────┐
        │  Finding providers for bafybeig...     │
        │  ├─ DHT lookup...                      │
        │  ├─ Found 3 providers                  │
        │  ├─ Establishing WebRTC connections    │
        │  └─ Starting download...               │
        └────────────────────────────────────────┘

Step 3: Monitor progress in Queue tab
        ┌────────────────────────────────────────┐
        │  Active Downloads:                     │
        │  ┌──────────────────────────────────┐  │
        │  │  📹 video.mp4                    │  │
        │  │  ████████████░░░░░░░░ 62%        │  │
        │  │  Speed: 2.4 MB/s | ETA: 45s      │  │
        │  │  [⏸ Pause] [❌ Cancel] [⚙ Limit]  │  │
        │  └──────────────────────────────────┘  │
        └────────────────────────────────────────┘

Step 4: Download complete!
        ┌────────────────────────────────────────┐
        │  ✓ Download complete!                  │
        │  📹 video.mp4 saved to Downloads/      │
        │  [📂 Open Location] [👁 Preview]       │
        └────────────────────────────────────────┘
```

### Queue Management Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    QUEUE MANAGEMENT                                         │
└─────────────────────────────────────────────────────────────────────────────┘

User can manage multiple downloads with priority:

┌─────────────────────────────────────────────────────────────────────────────┐
│  Queue (3 items)                                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│  [🔴 High Priority]                                                         │
│  ├─ important_doc.pdf      ████████████████████ 100% ✓                     │
│                                                                             │
│  [🟡 Normal Priority]                                                       │
│  ├─ movie.mkv              ████████████░░░░░░░░  65%   [Downloading]       │
│  │   Speed: 3.2 MB/s | ETA: 2m 15s                                         │
│  │   Bandwidth Limit: Unlimited                                            │
│  │   [⏸] [❌] [Set Limit: 1 MB/s ▼]                                        │
│                                                                             │
│  [🟢 Low Priority]                                                          │
│  ├─ large_archive.zip      ░░░░░░░░░░░░░░░░░░░░   0%   [Queued]            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

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
- Monitor **connected peers** count and list (up to 8 displayed)
- Check **shared files** count
- View **recent files** (last 5)
- Real-time **bandwidth history** tracking
- Network health indicators

#### 📁 Files (Share)

- **Share File**: Select any file to share on the network
- **Share Directory**: Share entire folders as multi-file torrents
- **Drag & Drop**: Native drag and drop file addition
- **Copy CID**: Copy the Content Identifier to share with others
- **Copy Magnet Link**: Generate `torrentium://` magnet links
- **Re-announce**: Refresh your file availability on the DHT
- **Open Location**: Open file location in system explorer
- **File Preview**: Preview images, videos, audio, PDFs, and text files in-app
- **Publish to Web Share**: Share files via web portal with visibility/expiration settings
- **Search & Filter**: Search through your shared files

#### ⬇️ Downloads

The Downloads section has multiple tabs:

**Download Tab**:

- **Download by CID**: Paste a CID to download from the network
- **Real-time progress**: Speed, ETA, and percentage tracking

**Search Tab**:

- **Search by text**: Find files in your local metadata index

**Queue Tab**:

- **Active downloads** with progress bars
- **Pause/Resume/Cancel** controls for each download
- **Set Priority**: Change download priority (Low/Normal/High)
- **Bandwidth Limit**: Set per-download speed limits
- **Queue management**: View all pending downloads

**History Tab**:

- **Download history**: View all completed downloads
- **File details**: Size, date, and status

#### 📈 Stats

- **Total Uploaded**: Bytes uploaded, chunks served, peers served
- **Total Downloaded**: Bytes downloaded, files completed
- **Share Ratio**: Upload/download ratio with color-coded indicator
- **Upload Rate Limit**: Current limit display
- **Per-file upload progress**: Real-time upload stats per CID
- **Auto-refresh**: Statistics update every 5 seconds

#### ⚙️ Settings

**Transfer Settings**:

- **Download Directory**: Configure where files are saved
- **Upload Rate Limit**: Set global upload bandwidth (KB/s or unlimited)
- **Download Rate Limit**: Set global download bandwidth (KB/s or unlimited)
- **Max Parallel Downloads**: Configure concurrent downloads (1-20)
- **Adaptive Parallel Downloads**: Toggle auto-adjustment based on bandwidth
- **Endgame Mode**: Toggle fast completion for final pieces

**Network Settings**:

- **Peer ID**: View and copy your unique peer identity
- **DHT Refresh**: Manually trigger DHT refresh

**Web Share Settings**:

- **Portal URL**: Configure web share portal endpoint
- **API Key**: Set authentication key
- **Default Visibility**: Choose public or unlisted
- **Default Expiration**: Set auto-expiration time

**Appearance**:

- **Theme Toggle**: Switch between dark and light mode

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
   - Content is chunked into 1MB pieces (configurable)
   - Each piece hash is stored in SQLite
   - IPFS CID is generated using multihash
   - File metadata announced to DHT with retry/backoff

2. **Peer Discovery**:
   - DHT lookup for content providers (max 10 providers)
   - Connection establishment via libp2p
   - WebRTC negotiation through signaling protocol
   - Peer reputation scoring and selection

3. **Data Transfer**:
   - WebRTC data channel establishment (dual channels)
   - Rarest-first piece selection algorithm
   - Chunked transfer (16KB chunks) with progress tracking
   - Bitfield exchange for piece availability
   - Real-time integrity verification
   - Endgame mode for final pieces
   - Checkpointing every 5 pieces

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
    is_directory INTEGER DEFAULT 0,
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

-- Search index for local metadata
CREATE TABLE metadata_index (
    cid TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    file_hash TEXT NOT NULL
);

-- Multi-file torrent entries
CREATE TABLE file_entries (
    id TEXT PRIMARY KEY,
    cid TEXT NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL,
    offset INTEGER NOT NULL,
    file_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Upload statistics tracking
CREATE TABLE uploads (
    id TEXT PRIMARY KEY,
    cid TEXT NOT NULL,
    peer_id TEXT NOT NULL,
    bytes_uploaded INTEGER NOT NULL,
    chunks_served INTEGER NOT NULL,
    uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### WebRTC Integration

Torrentium uses WebRTC data channels for efficient peer-to-peer communication:

- **Dual data channels**: "data" (unreliable, fast) + "reliable" (ordered, retransmits)
- **ICE servers**: Multiple STUN/TURN servers for NAT traversal (Google, Cloudflare)
- **Signaling**: Custom libp2p protocol `/torrentium/webrtc-signaling/1.0`
- **Connection pooling**: Reuse connections (max 20, 5min idle timeout, 30min max age)
- **Keep-alive**: 15-second interval heartbeat messages
- **Binary data channels**: Efficient file content transfer

### Configuration Options

Torrentium supports configuration via YAML file (`config.yaml` or `torrentium.yaml`) or environment variables:

#### P2P Configuration

| Option               | Default              | Environment Variable          |
| -------------------- | -------------------- | ----------------------------- |
| Listen Address       | `/ip4/0.0.0.0/tcp/0` | `TORRENTIUM_LISTEN_ADDRESS`   |
| Private Key Path     | `private_key`        | `TORRENTIUM_PRIVATE_KEY_PATH` |
| DHT Refresh Interval | `10m`                | -                             |
| Min Peer Count       | `5`                  | -                             |
| Bootstrap Timeout    | `15s`                | -                             |

#### Transfer Configuration

| Option                 | Default  | Description             |
| ---------------------- | -------- | ----------------------- |
| Piece Size             | 1 MiB    | File chunking size      |
| Max Chunk Size         | 16 KiB   | WebRTC transfer chunk   |
| Max Parallel Downloads | 3        | Concurrent downloads    |
| Adaptive Downloads     | `true`   | Auto-adjust parallelism |
| Endgame Mode           | `true`   | Fast completion mode    |
| Endgame Threshold      | 5%       | When to trigger endgame |
| Checkpoint Interval    | 5 pieces | State save frequency    |

#### Rate Limiting

| Option            | Default       | Description              |
| ----------------- | ------------- | ------------------------ |
| Max Upload Rate   | 0 (unlimited) | Bytes/sec upload limit   |
| Max Download Rate | 0 (unlimited) | Bytes/sec download limit |

#### Environment Variables

```bash
TORRENTIUM_DB_PATH          # Database file path
TORRENTIUM_LISTEN_ADDRESS   # P2P listen address
TORRENTIUM_PRIVATE_KEY_PATH # Identity key path
TORRENTIUM_RELAY_ADDRESS    # Circuit relay address
TORRENTIUM_LOG_LEVEL        # Logging level
TORRENTIUM_LOG_FORMAT       # Log format
TORRENTIUM_DOWNLOAD_DIR     # Default download directory
```

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
│   │   ├── webrtc.go       # WebRTC peer management
│   │   └── connection_pool.go # Connection pooling
│   ├── config/             # Configuration management
│   │   └── config.go       # App settings and defaults
│   ├── db/                 # Database layer
│   │   └── db.go           # SQLite operations and schema
│   ├── logger/             # Logging utilities
│   │   └── logger.go       # Structured logging
│   ├── p2p/                # P2P networking
│   │   ├── host.go         # libp2p host creation
│   │   ├── signaling.go    # WebRTC signaling protocol
│   │   ├── relay.go        # Dynamic relay discovery
│   │   └── relay_failover.go # Multi-relay failover
│   └── torrentium_client/  # Core client engine
│       ├── client.go       # Main client logic
│       ├── adaptive_download.go # Adaptive parallelism
│       └── download_queue.go # Priority download queue
├── gui/                     # Wails desktop application
│   ├── app.go              # Go backend bindings for GUI
│   ├── main.go             # Wails app entry point
│   ├── wails.json          # Wails configuration
│   ├── build/              # Build outputs
│   │   └── bin/            # Compiled executables
│   └── frontend/           # Vue.js frontend
│       ├── index.html      # HTML entry point
│       ├── package.json    # Node dependencies
│       ├── vite.config.js  # Vite bundler config
│       └── src/
│           ├── App.vue     # Main Vue component
│           ├── main.js     # Vue entry point
│           ├── style.css   # Global styles
│           └── components/
│               ├── Dashboard.vue      # Dashboard view
│               ├── Files.vue          # File sharing view
│               ├── Downloads.vue      # Downloads view
│               ├── Stats.vue          # Statistics view
│               ├── Settings.vue       # Settings view
│               ├── Sidebar.vue        # Navigation sidebar
│               ├── FilePreview.vue    # File preview modal
│               └── ToastNotification.vue # Notifications
├── webshare/               # Web share portal (separate service)
│   ├── main.go            # Server entry point
│   ├── server/            # HTTP server implementation
│   └── frontend/          # Web portal frontend
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
- **Peer Authentication**: libp2p cryptographic identities (Ed25519)
- **NAT Traversal**: Secure STUN/TURN server usage
- **Local Storage**: SQLite database with appropriate file permissions
- **Network Security**: Encrypted WebRTC data channels
- **Connection Security**: TLS-encrypted libp2p connections

## 🗺️ Roadmap

### Completed ✅

- [x] **Desktop GUI** - Modern Wails-based desktop application
- [x] **Cross-platform builds** - Windows, macOS, Linux support
- [x] **Relay server integration** - NAT traversal support
- [x] **Multi-relay failover** - Automatic relay switching
- [x] **Download queue** - Priority-based download management
- [x] **Bandwidth control** - Upload/download rate limiting
- [x] **Adaptive downloads** - Smart parallel download adjustment
- [x] **Endgame mode** - Fast completion for final pieces
- [x] **Directory support** - Multi-file torrent sharing
- [x] **File preview** - In-app media and document preview
- [x] **Web Share portal** - Web-based file sharing
- [x] **Magnet links** - Shareable torrentium:// links
- [x] **Theme toggle** - Dark/light mode support

### Planned 📋

- [ ] **Auto-updates** - Automatic application updates
- [ ] **Mobile Support** - Android/iOS client applications
- [ ] **Improved Search** - Full-text search across file contents
- [ ] **Plugin System** - Extensible architecture for custom protocols
- [ ] **Analytics Dashboard** - Advanced network statistics and metrics

## ⚠️ Current Limitations & Challenges

### Technical Limitations

| Limitation                    | Description                                                             | Impact                                                                | Workaround                                                                                   |
| ----------------------------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| **Single Relay Dependency**   | Initial connection requires at least one relay server to be operational | Users behind strict NATs may fail to connect if all relays are down   | Multi-relay failover system mitigates this, but complete relay outage prevents NAT traversal |
| **DHT Bootstrap Requirement** | Requires connection to IPFS bootstrap nodes for initial DHT population  | First-time users need internet connectivity to public bootstrap nodes | Configurable bootstrap nodes allow custom/private network setups                             |
| **WebRTC TURN Limitations**   | TURN servers have bandwidth costs and may have rate limits              | High-throughput transfers through TURN may be throttled               | Direct connections via STUN/hole-punching preferred when possible                            |
| **Large File Handling**       | Files are loaded into memory for chunking during share operation        | Very large files (>10GB) may cause memory pressure                    | Streaming-based chunking planned for future release                                          |
| **No Encryption at Rest**     | Downloaded files are stored in plain format                             | Local file security depends on OS-level encryption                    | Users should use OS encryption (BitLocker, FileVault)                                        |

### Network Challenges

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    NETWORK CHALLENGE MATRIX                                 │
└─────────────────────────────────────────────────────────────────────────────┘

Challenge                    Severity    Current Solution         Status
─────────────────────────────────────────────────────────────────────────────
Symmetric NAT Traversal      High        Relay fallback           ✓ Mitigated
DHT Partition/Churn          Medium      Auto re-bootstrap        ✓ Handled
Peer Churn During Download   Medium      Multi-peer redundancy    ✓ Handled
Relay Server Unavailability  Medium      Multi-relay failover     ✓ Handled
Slow Peer Detection          Low         Reputation scoring       ✓ Handled
ICE Gathering Timeout        Low         Configurable timeout     ✓ Handled
```

### Operational Challenges

1. **Peer Discovery Latency**
   - DHT lookups can take 2-10 seconds depending on network conditions
   - Content must be re-announced periodically as DHT entries expire
   - Solution: Background DHT refresh and aggressive re-announcement

2. **Relay Server Costs**
   - Running public relay servers requires infrastructure investment
   - Current reliance on community/public relays may not scale
   - Future: Self-hosted relay option with easy deployment

3. **Cross-Platform Build Complexity**
   - CGO dependency (SQLite) requires platform-specific toolchains
   - Windows builds require 64-bit MinGW
   - macOS universal binaries require careful configuration

4. **Content Availability**
   - Files are only available when at least one seeder is online
   - No built-in incentive mechanism for long-term seeding
   - Download checkpointing helps with interrupted transfers

### Known Issues

| Issue                      | Description                                                                  | Status                                           |
| -------------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------ |
| Relay Peer ID Changes      | Relay servers may restart with new peer IDs, invalidating cached connections | Auto-detected and recovered                      |
| Memory Usage on Many Peers | Connection pooling may consume memory with many simultaneous peers           | Pool size configurable (default: 20)             |
| First Piece Delay          | Initial piece download may be slower due to WebRTC negotiation overhead      | Subsequent pieces faster due to connection reuse |
| Windows Firewall Prompts   | Windows may prompt for firewall access on first run                          | Expected behavior, user must allow               |

## 🔮 Future Scope

### Short-Term Goals (Next 6 Months)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SHORT-TERM ROADMAP                                       │
└─────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────┐     ┌──────────────────────┐     ┌──────────────────┐
│  Q1 2026             │     │  Q2 2026             │     │  Q3 2026         │
├──────────────────────┤     ├──────────────────────┤     ├──────────────────┤
│ • Auto-updates       │     │ • Streaming chunking │     │ • Mobile Beta    │
│ • Improved search    │     │ • Private networks   │     │ • Plugin system  │
│ • Bandwidth graphs   │     │ • File encryption    │     │ • REST API       │
│ • Selective download │     │ • Seeding incentives │     │ • CLI overhaul   │
└──────────────────────┘     └──────────────────────┘     └──────────────────┘
```

### Planned Features

| Feature                | Description                                      | Priority | Complexity |
| ---------------------- | ------------------------------------------------ | -------- | ---------- |
| **Auto-Updates**       | In-app update mechanism with delta updates       | High     | Medium     |
| **Streaming Chunking** | Memory-efficient large file handling             | High     | High       |
| **Mobile Apps**        | React Native Android/iOS clients                 | High     | High       |
| **Private Networks**   | Invite-only DHT networks with access control     | Medium   | High       |
| **File Encryption**    | End-to-end encrypted file sharing                | Medium   | Medium     |
| **Selective Download** | Download specific files from multi-file torrents | Medium   | Low        |
| **Seeding Incentives** | Token-based or ratio-based seeding rewards       | Low      | High       |
| **Plugin System**      | Extensible protocol handlers and UI plugins      | Low      | High       |
| **REST API**           | HTTP API for external integrations               | Medium   | Medium     |
| **Bandwidth Graphs**   | Real-time network statistics visualization       | Low      | Low        |

### Long-Term Vision

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TORRENTIUM 2.0 VISION                                    │
└─────────────────────────────────────────────────────────────────────────────┘

                    ┌─────────────────────────────────┐
                    │      TORRENTIUM ECOSYSTEM       │
                    └─────────────────────────────────┘
                                    │
           ┌────────────────────────┼────────────────────────┐
           │                        │                        │
           ▼                        ▼                        ▼
   ┌───────────────┐      ┌───────────────┐      ┌───────────────┐
   │   Desktop     │      │    Mobile     │      │   Web         │
   │   (Current)   │      │   (Planned)   │      │   (WebShare)  │
   │               │      │               │      │               │
   │ • Windows     │      │ • Android     │      │ • Portal UI   │
   │ • macOS       │      │ • iOS         │      │ • File Browse │
   │ • Linux       │      │ • React Native│      │ • Statistics  │
   └───────────────┘      └───────────────┘      └───────────────┘
           │                        │                        │
           └────────────────────────┼────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │        CORE NETWORK           │
                    │                               │
                    │  • Decentralized DHT          │
                    │  • WebRTC Data Channels       │
                    │  • Multi-Relay Support        │
                    │  • Private Network Option     │
                    │  • Plugin Ecosystem           │
                    └───────────────────────────────┘
```

### Research & Exploration

1. **IPFS Integration**
   - Explore native IPFS pinning service integration
   - Potentially use IPFS for permanent content archival

2. **Blockchain-based Incentives**
   - Research decentralized storage incentive mechanisms
   - Evaluate Filecoin/Arweave integration possibilities

3. **WebTransport Support**
   - Investigate WebTransport as WebRTC alternative
   - Could enable browser-native clients

4. **Zero-Knowledge Proofs**
   - Research ZK proofs for private content verification
   - Enable verifiable downloads without content exposure

5. **Distributed Search Index**
   - Build decentralized full-text search over shared content
   - Allow network-wide content discovery

## 🐛 Troubleshooting

### Common Issues

1. **"Relay connection failed"**: The relay server may have restarted with a new peer ID. The app will automatically failover to other relays.
2. **No peers found**: Check internet connection and firewall settings. Allow the app through Windows Firewall.
3. **Download failures**: Verify CID format and ensure the provider is online.
4. **Window closes immediately**: Ensure you're running the 64-bit version and have WebView2 installed.
5. **Database errors**: Ensure write permissions in the application directory.
6. **Build errors on Windows**: Make sure you have 64-bit MinGW installed (not 32-bit).
7. **Slow downloads**: Try enabling adaptive downloads and endgame mode in Settings.
8. **Upload not working**: Check if upload rate limit is set too low in Settings.

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
