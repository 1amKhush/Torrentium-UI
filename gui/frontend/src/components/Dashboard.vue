<script setup>
import { ref, onMounted } from "vue";
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

const loadData = async () => {
  try {
    const [statsData, filesData, peersData] = await Promise.all([
      GetStats(),
      GetLocalFiles(),
      GetConnectedPeers(),
    ]);
    stats.value = statsData;
    recentFiles.value = filesData.slice(0, 5);
    peers.value = peersData.slice(0, 5);
  } catch (err) {
    console.error("Failed to load dashboard data:", err);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  loadData();

  // Refresh data periodically
  const interval = setInterval(loadData, 10000);

  EventsOn("fileAdded", () => loadData());
  EventsOn("downloadComplete", () => loadData());
});
</script>

<template>
  <div class="dashboard">
    <header class="page-header">
      <h1>Dashboard</h1>
      <p class="subtitle">
        Welcome to Torrentium - Decentralized P2P File Sharing
      </p>
    </header>

    <!-- Quick Stats -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon">👥</div>
        <div class="stat-info">
          <span class="stat-value">{{
            networkStatus?.connectedPeers || 0
          }}</span>
          <span class="stat-label">Connected Peers</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon">📁</div>
        <div class="stat-info">
          <span class="stat-value">{{
            networkStatus?.sharedFilesCount || 0
          }}</span>
          <span class="stat-label">Shared Files</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon">⬆️</div>
        <div class="stat-info">
          <span class="stat-value">{{ stats?.uploadedHuman || "0 B" }}</span>
          <span class="stat-label">Uploaded</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon">⬇️</div>
        <div class="stat-info">
          <span class="stat-value">{{ stats?.downloadedHuman || "0 B" }}</span>
          <span class="stat-label">Downloaded</span>
        </div>
      </div>
    </div>

    <!-- Network Info -->
    <div class="section" v-if="networkStatus">
      <h2 class="section-title">Network Information</h2>
      <div class="info-card">
        <div class="info-row">
          <span class="info-label">Peer ID</span>
          <span class="info-value mono"
            >{{ networkStatus.peerId?.slice(0, 20) }}...</span
          >
        </div>
        <div class="info-row">
          <span class="info-label">DHT Routing Table</span>
          <span class="info-value"
            >{{ networkStatus.dhtRoutingTable }} nodes</span
          >
        </div>
        <div class="info-row">
          <span class="info-label">Status</span>
          <span
            :class="[
              'status-badge',
              networkStatus.isConnected ? 'online' : 'offline',
            ]"
          >
            {{ networkStatus.isConnected ? "Online" : "Offline" }}
          </span>
        </div>
      </div>
    </div>

    <!-- Recent Files -->
    <div class="section">
      <h2 class="section-title">Recent Shared Files</h2>
      <div class="files-list" v-if="recentFiles.length > 0">
        <div class="file-item" v-for="file in recentFiles" :key="file.cid">
          <div class="file-icon">📄</div>
          <div class="file-info">
            <span class="file-name">{{ file.filename }}</span>
            <span class="file-meta"
              >{{ file.sizeHuman }} • {{ file.createdAt }}</span
            >
          </div>
        </div>
      </div>
      <div class="empty-state" v-else>
        <p>No files shared yet. Go to Files to add some!</p>
      </div>
    </div>

    <!-- Connected Peers Preview -->
    <div class="section">
      <h2 class="section-title">Connected Peers</h2>
      <div class="peers-list" v-if="peers.length > 0">
        <div class="peer-item" v-for="peer in peers" :key="peer.peerId">
          <div class="peer-icon">🖥️</div>
          <div class="peer-info">
            <span class="peer-id">{{ peer.peerId.slice(0, 30) }}...</span>
            <span class="peer-status" :class="{ connected: peer.connected }">
              {{ peer.connected ? "Connected" : "Disconnected" }}
            </span>
          </div>
        </div>
      </div>
      <div class="empty-state" v-else>
        <p>No peers connected yet. The network is bootstrapping...</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1200px;
}

.page-header {
  margin-bottom: 32px;
}

.page-header h1 {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 8px;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 1rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.stat-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 1px solid var(--border);
}

.stat-icon {
  font-size: 2rem;
}

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
}

.stat-label {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.section {
  margin-bottom: 32px;
}

.section-title {
  font-size: 1.2rem;
  font-weight: 600;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 20px;
  border: 1px solid var(--border);
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  color: var(--text-secondary);
}

.info-value {
  font-weight: 500;
}

.info-value.mono {
  font-family: monospace;
  font-size: 0.9rem;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 0.8rem;
  font-weight: 500;
}

.status-badge.online {
  background: rgba(34, 197, 94, 0.2);
  color: var(--success);
}

.status-badge.offline {
  background: rgba(239, 68, 68, 0.2);
  color: var(--error);
}

.files-list,
.peers-list {
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 1px solid var(--border);
  overflow: hidden;
}

.file-item,
.peer-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--border);
}

.file-item:last-child,
.peer-item:last-child {
  border-bottom: none;
}

.file-icon,
.peer-icon {
  font-size: 1.5rem;
}

.file-info,
.peer-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-name {
  font-weight: 500;
}

.file-meta {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.peer-id {
  font-family: monospace;
  font-size: 0.85rem;
}

.peer-status {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.peer-status.connected {
  color: var(--success);
}

.empty-state {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 32px;
  text-align: center;
  color: var(--text-secondary);
  border: 1px dashed var(--border);
}
</style>
