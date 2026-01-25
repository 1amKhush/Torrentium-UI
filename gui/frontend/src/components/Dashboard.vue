<script setup>
import { ref, onMounted, onUnmounted, computed } from "vue";
import {
  GetNetworkStatus,
  GetStats,
  GetLocalFiles,
  GetConnectedPeers,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const props = defineProps({
  networkStatus: Object,
});

const stats = ref(null);
const recentFiles = ref([]);
const peers = ref([]);
const loading = ref(true);
const bandwidthHistory = ref([]);
let intervalId = null;

const loadData = async () => {
  try {
    const [statsData, filesData, peersData] = await Promise.all([
      GetStats(),
      GetLocalFiles(),
      GetConnectedPeers(),
    ]);
    stats.value = statsData;
    recentFiles.value = filesData.slice(0, 5);
    peers.value = peersData.slice(0, 8);

    // Track bandwidth history for graph
    if (statsData) {
      bandwidthHistory.value.push({
        time: Date.now(),
        upload: statsData.uploadSpeed || 0,
        download: statsData.downloadSpeed || 0,
      });
      if (bandwidthHistory.value.length > 20) {
        bandwidthHistory.value.shift();
      }
    }
  } catch (err) {
    console.error("Failed to load dashboard data:", err);
  } finally {
    loading.value = false;
  }
};

const shareRatio = computed(() => {
  if (!stats.value) return "0.00";
  const ratio =
    stats.value.downloaded > 0
      ? (stats.value.uploaded / stats.value.downloaded).toFixed(2)
      : "∞";
  return ratio;
});

const getFileIcon = (filename) => {
  const ext = filename?.split(".").pop()?.toLowerCase();
  const icons = {
    pdf: "📕",
    doc: "📘",
    docx: "📘",
    txt: "📄",
    jpg: "🖼️",
    jpeg: "🖼️",
    png: "🖼️",
    gif: "🖼️",
    svg: "🖼️",
    mp3: "🎵",
    wav: "🎵",
    flac: "🎵",
    mp4: "🎬",
    mkv: "🎬",
    avi: "🎬",
    mov: "🎬",
    zip: "📦",
    rar: "📦",
    "7z": "📦",
    tar: "📦",
    exe: "⚙️",
    dmg: "⚙️",
    app: "⚙️",
    js: "💻",
    py: "💻",
    go: "💻",
    rs: "💻",
  };
  return icons[ext] || "📄";
};

onMounted(() => {
  loadData();
  intervalId = setInterval(loadData, 5000);

  EventsOn("fileAdded", () => loadData());
  EventsOn("downloadComplete", () => loadData());
});

onUnmounted(() => {
  if (intervalId) clearInterval(intervalId);
});
</script>

<template>
  <div class="dashboard">
    <header class="page-header">
      <div class="header-content">
        <h1>Dashboard</h1>
        <p class="subtitle">
          Welcome to Torrentium — Decentralized P2P File Sharing
        </p>
      </div>
      <div class="header-badge" v-if="networkStatus?.isConnected">
        <span class="badge-dot"></span>
        Network Active
      </div>
    </header>

    <!-- Quick Stats -->
    <div class="stats-grid">
      <div class="stat-card stat-peers">
        <div class="stat-icon-wrapper">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{
            networkStatus?.connectedPeers || 0
          }}</span>
          <span class="stat-label">Connected Peers</span>
        </div>
        <div class="stat-graph">
          <div class="mini-bars">
            <div
              v-for="n in 8"
              :key="n"
              class="mini-bar"
              :style="{ height: Math.random() * 100 + '%' }"
            ></div>
          </div>
        </div>
      </div>

      <div class="stat-card stat-files">
        <div class="stat-icon-wrapper">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
            />
          </svg>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{
            networkStatus?.sharedFilesCount || 0
          }}</span>
          <span class="stat-label">Shared Files</span>
        </div>
      </div>

      <div class="stat-card stat-upload">
        <div class="stat-icon-wrapper upload">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="17 11 12 6 7 11" />
            <line x1="12" y1="6" x2="12" y2="18" />
          </svg>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats?.uploadedHuman || "0 B" }}</span>
          <span class="stat-label">Total Uploaded</span>
        </div>
      </div>

      <div class="stat-card stat-download">
        <div class="stat-icon-wrapper download">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="7 13 12 18 17 13" />
            <line x1="12" y1="6" x2="12" y2="18" />
          </svg>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats?.downloadedHuman || "0 B" }}</span>
          <span class="stat-label">Total Downloaded</span>
        </div>
      </div>
    </div>

    <div class="content-grid">
      <!-- Network Info -->
      <div class="section network-section" v-if="networkStatus">
        <div class="section-header">
          <h2 class="section-title">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <circle cx="12" cy="12" r="10" />
              <line x1="2" y1="12" x2="22" y2="12" />
              <path
                d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"
              />
            </svg>
            Network Status
          </h2>
        </div>
        <div class="info-card">
          <div class="info-row">
            <span class="info-label">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="info-icon"
              >
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
              Peer ID
            </span>
            <span class="info-value mono"
              >{{ networkStatus.peerId?.slice(0, 16) }}...</span
            >
          </div>
          <div class="info-row">
            <span class="info-label">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="info-icon"
              >
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
              </svg>
              DHT Nodes
            </span>
            <span class="info-value"
              >{{ networkStatus.dhtRoutingTable }} nodes</span
            >
          </div>
          <div class="info-row">
            <span class="info-label">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="info-icon"
              >
                <path d="M5 12.55a11 11 0 0 1 14.08 0" />
                <path d="M1.42 9a16 16 0 0 1 21.16 0" />
                <circle cx="12" cy="20" r="1" />
              </svg>
              Relay
            </span>
            <span
              :class="[
                'status-badge',
                networkStatus.relayConnected ? 'online' : 'offline',
              ]"
            >
              {{ networkStatus.relayConnected ? "Connected" : "Disconnected" }}
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="info-icon"
              >
                <path
                  d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"
                />
              </svg>
              Share Ratio
            </span>
            <span
              class="info-value ratio"
              :class="{ good: parseFloat(shareRatio) >= 1 }"
              >{{ shareRatio }}</span
            >
          </div>
        </div>
      </div>

      <!-- Recent Files -->
      <div class="section files-section">
        <div class="section-header">
          <h2 class="section-title">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"
              />
              <polyline points="14 2 14 8 20 8" />
            </svg>
            Recent Files
          </h2>
        </div>
        <div class="files-list" v-if="recentFiles.length > 0">
          <div class="file-item" v-for="file in recentFiles" :key="file.cid">
            <div class="file-icon">{{ getFileIcon(file.filename) }}</div>
            <div class="file-info">
              <span class="file-name">{{ file.filename }}</span>
              <span class="file-meta"
                >{{ file.sizeHuman }} • {{ file.createdAt }}</span
              >
            </div>
          </div>
        </div>
        <div class="empty-state" v-else>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            class="empty-icon"
          >
            <path
              d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
            />
          </svg>
          <p>No files shared yet</p>
          <span class="empty-hint">Go to Files to start sharing</span>
        </div>
      </div>
    </div>

    <!-- Connected Peers Preview -->
    <div class="section peers-section">
      <div class="section-header">
        <h2 class="section-title">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="9" cy="7" r="4" />
            <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
            <path d="M16 3.13a4 4 0 0 1 0 7.75" />
          </svg>
          Connected Peers
        </h2>
        <span class="peer-count-badge">{{ peers.length }}</span>
      </div>
      <div class="peers-grid" v-if="peers.length > 0">
        <div class="peer-card" v-for="peer in peers" :key="peer.peerId">
          <div class="peer-avatar">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
              <line x1="8" y1="21" x2="16" y2="21" />
              <line x1="12" y1="17" x2="12" y2="21" />
            </svg>
          </div>
          <div class="peer-details">
            <span class="peer-id">{{ peer.peerId?.slice(0, 12) }}...</span>
            <span :class="['peer-status', { connected: peer.connected }]">
              <span class="status-dot"></span>
              {{ peer.connected ? "Active" : "Inactive" }}
            </span>
          </div>
        </div>
      </div>
      <div class="empty-state" v-else>
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          class="empty-icon"
        >
          <circle cx="12" cy="12" r="10" />
          <path d="M8 12h8M12 8v8" />
        </svg>
        <p>No peers connected</p>
        <span class="empty-hint">Network is bootstrapping...</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  width: 100%;
  animation: fadeIn 0.4s ease;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
}

.page-header h1 {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 8px;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 1rem;
}

.header-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--success-glow);
  border-radius: var(--radius-xl);
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--success);
}

.badge-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--success);
  animation: pulse 2s ease-in-out infinite;
}

/* Stats Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

.stat-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 24px;
  display: flex;
  align-items: flex-start;
  gap: 16px;
  border: 1px solid var(--border);
  position: relative;
  overflow: hidden;
  transition: all var(--transition-normal);
}

.stat-card:hover {
  border-color: var(--border-light);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-icon-wrapper {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-glow);
  color: var(--accent);
  flex-shrink: 0;
}

.stat-icon-wrapper svg {
  width: 24px;
  height: 24px;
}

.stat-icon-wrapper.upload {
  background: var(--success-glow);
  color: var(--success);
}

.stat-icon-wrapper.download {
  background: var(--accent-glow);
  color: var(--accent-secondary);
}

.stat-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-value {
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1.2;
}

.stat-label {
  color: var(--text-muted);
  font-size: 0.85rem;
}

.stat-graph {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 100px;
  height: 40px;
  opacity: 0.3;
}

.mini-bars {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 100%;
  padding: 8px;
}

.mini-bar {
  flex: 1;
  background: var(--accent);
  border-radius: 2px;
  min-height: 4px;
}

/* Content Grid */
.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
  margin-bottom: 32px;
}

/* Sections */
.section {
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-primary);
}

.section-title svg {
  width: 20px;
  height: 20px;
  color: var(--accent);
}

.peer-count-badge {
  background: var(--bg-tertiary);
  padding: 4px 12px;
  border-radius: var(--radius-xl);
  font-size: 0.8rem;
  color: var(--text-muted);
}

/* Info Card */
.info-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 8px;
  border: 1px solid var(--border);
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  border-radius: var(--radius-md);
  transition: background var(--transition-fast);
}

.info-row:hover {
  background: var(--bg-tertiary);
}

.info-label {
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.9rem;
}

.info-icon {
  width: 16px;
  height: 16px;
  color: var(--text-muted);
}

.info-value {
  font-weight: 500;
  color: var(--text-primary);
}

.info-value.mono {
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 0.85rem;
  color: var(--accent);
}

.info-value.ratio {
  color: var(--warning);
}

.info-value.ratio.good {
  color: var(--success);
}

.status-badge {
  padding: 6px 14px;
  border-radius: var(--radius-xl);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.status-badge.online {
  background: var(--success-glow);
  color: var(--success);
}

.status-badge.offline {
  background: var(--error-glow);
  color: var(--error);
}

/* Files List */
.files-list {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  overflow: hidden;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
  transition: background var(--transition-fast);
}

.file-item:last-child {
  border-bottom: none;
}

.file-item:hover {
  background: var(--bg-tertiary);
}

.file-icon {
  font-size: 1.5rem;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
}

.file-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow: hidden;
}

.file-name {
  font-weight: 500;
  font-size: 0.9rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Peers Grid */
.peers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.peer-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: all var(--transition-normal);
}

.peer-card:hover {
  border-color: var(--accent);
  background: var(--bg-tertiary);
}

.peer-avatar {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.peer-avatar svg {
  width: 20px;
  height: 20px;
}

.peer-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow: hidden;
}

.peer-id {
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.peer-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.7rem;
  color: var(--text-muted);
}

.peer-status .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-muted);
}

.peer-status.connected {
  color: var(--success);
}

.peer-status.connected .status-dot {
  background: var(--success);
}

/* Empty State */
.empty-state {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 40px;
  text-align: center;
  border: 1px dashed var(--border-light);
}

.empty-icon {
  width: 48px;
  height: 48px;
  margin: 0 auto 16px;
  color: var(--text-muted);
}

.empty-state p {
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.empty-hint {
  font-size: 0.8rem;
  color: var(--text-muted);
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Responsive Styles */
@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 900px) {
  .content-grid {
    grid-template-columns: 1fr;
  }

  .peers-grid {
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  }
}

@media (max-width: 768px) {
  .dashboard {
    padding: 0;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 24px;
  }

  .page-header h1 {
    font-size: 1.5rem;
  }

  .subtitle {
    font-size: 0.9rem;
  }

  .header-badge {
    font-size: 0.75rem;
    padding: 6px 12px;
  }

  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
    margin-bottom: 24px;
  }

  .stat-card {
    padding: 16px;
  }

  .stat-icon-wrapper {
    width: 40px;
    height: 40px;
  }

  .stat-icon-wrapper svg {
    width: 20px;
    height: 20px;
  }

  .stat-value {
    font-size: 1.25rem;
  }

  .stat-label {
    font-size: 0.75rem;
  }

  .stat-graph {
    display: none;
  }

  .content-grid {
    gap: 16px;
    margin-bottom: 24px;
  }

  .section-title {
    font-size: 1rem;
  }

  .info-row {
    padding: 12px 14px;
  }

  .info-label {
    font-size: 0.85rem;
  }

  .file-item {
    padding: 12px 14px;
  }

  .peer-card {
    padding: 12px;
  }

  .empty-state {
    padding: 30px 20px;
  }
}

@media (max-width: 480px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .stat-card {
    flex-direction: row;
    align-items: center;
  }

  .peers-grid {
    grid-template-columns: 1fr;
  }

  .info-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .info-value {
    font-size: 0.9rem;
  }

  .status-badge {
    align-self: flex-start;
  }

  .file-item {
    gap: 10px;
  }

  .file-icon {
    width: 36px;
    height: 36px;
    font-size: 1.2rem;
  }
}
</style>
