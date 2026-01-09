<script setup>
import { ref, onMounted, computed } from "vue";
import { GetStats, GetUploadProgress } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const stats = ref(null);
const uploadProgress = ref([]);
const loading = ref(true);

const loadData = async () => {
  try {
    const [statsData, progressData] = await Promise.all([
      GetStats(),
      GetUploadProgress(),
    ]);
    stats.value = statsData;
    uploadProgress.value = progressData;
  } catch (err) {
    console.error("Failed to load stats:", err);
  } finally {
    loading.value = false;
  }
};

const ratioColor = computed(() => {
  if (!stats.value) return "var(--text-secondary)";
  if (stats.value.ratio >= 1) return "var(--success)";
  if (stats.value.ratio >= 0.5) return "var(--warning)";
  return "var(--error)";
});

const ratioText = computed(() => {
  if (!stats.value) return "0.00";
  return stats.value.ratio.toFixed(2);
});

onMounted(() => {
  loadData();

  // Refresh every 5 seconds
  const interval = setInterval(loadData, 5000);

  EventsOn("downloadComplete", loadData);
  EventsOn("fileAdded", loadData);
});
</script>

<template>
  <div class="stats-page">
    <header class="page-header">
      <h1>Statistics</h1>
      <p class="subtitle">Track your upload and download activity</p>
    </header>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading statistics...</p>
    </div>

    <template v-else-if="stats">
      <!-- Main Stats Cards -->
      <div class="stats-overview">
        <div class="stat-card primary">
          <div class="stat-header">
            <span class="stat-icon">⬆️</span>
            <span class="stat-label">Total Uploaded</span>
          </div>
          <div class="stat-value">{{ stats.uploadedHuman }}</div>
          <div class="stat-details">
            <span>{{ stats.chunksServed }} chunks served</span>
            <span>{{ stats.peersServed }} peers served</span>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-icon">⬇️</span>
            <span class="stat-label">Total Downloaded</span>
          </div>
          <div class="stat-value">{{ stats.downloadedHuman }}</div>
          <div class="stat-details">
            <span>{{ stats.filesDownloaded }} files</span>
          </div>
        </div>

        <div class="stat-card ratio-card">
          <div class="stat-header">
            <span class="stat-icon">📊</span>
            <span class="stat-label">Share Ratio</span>
          </div>
          <div class="stat-value" :style="{ color: ratioColor }">
            {{ ratioText }}
          </div>
          <div class="ratio-bar">
            <div
              class="ratio-fill"
              :style="{
                width: Math.min(stats.ratio * 50, 100) + '%',
                background: ratioColor,
              }"
            ></div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-icon">📁</span>
            <span class="stat-label">Files Shared</span>
          </div>
          <div class="stat-value">{{ stats.filesShared }}</div>
          <div class="stat-details">
            <span>Available on network</span>
          </div>
        </div>
      </div>

      <!-- Upload Rate -->
      <div class="info-section">
        <h2>Upload Settings</h2>
        <div class="info-card">
          <div class="info-row">
            <span class="info-label">Maximum Upload Rate</span>
            <span class="info-value">{{ stats.maxUploadRateHuman }}</span>
          </div>
        </div>
      </div>

      <!-- Active Seeding -->
      <div class="seeding-section" v-if="uploadProgress.length > 0">
        <h2>Active Seeding</h2>
        <div class="seeding-list">
          <div
            class="seeding-item"
            v-for="item in uploadProgress"
            :key="item.cid"
          >
            <div class="seeding-info">
              <span class="seeding-cid">{{ item.cid.slice(0, 24) }}...</span>
              <div class="seeding-stats">
                <span>📤 {{ item.uploadedHuman }}</span>
                <span>📦 {{ item.chunksServed }} chunks</span>
                <span>👥 {{ item.peersServed }} peers</span>
                <span>⚡ {{ item.avgSpeed }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Ratio Explanation -->
      <div class="explanation-section">
        <h2>About Share Ratio</h2>
        <div class="explanation-card">
          <p>
            Your share ratio is the amount of data you've uploaded divided by
            what you've downloaded.
          </p>
          <ul>
            <li>
              <span class="ratio-indicator good"></span>
              <strong>≥ 1.0:</strong> You're giving back as much or more than
              you take
            </li>
            <li>
              <span class="ratio-indicator okay"></span>
              <strong>0.5 - 1.0:</strong> Good progress, keep seeding!
            </li>
            <li>
              <span class="ratio-indicator low"></span>
              <strong>&lt; 0.5:</strong> Consider seeding more to help the
              network
            </li>
          </ul>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.stats-page {
  max-width: 1000px;
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
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px;
  color: var(--text-secondary);
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.stats-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

.stat-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 24px;
  border: 1px solid var(--border);
}

.stat-card.primary {
  border-color: var(--accent);
  background: linear-gradient(
    135deg,
    rgba(59, 130, 246, 0.1) 0%,
    var(--bg-secondary) 100%
  );
}

.stat-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.stat-icon {
  font-size: 1.3rem;
}

.stat-label {
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 8px;
}

.stat-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.ratio-card .stat-value {
  font-size: 2.5rem;
}

.ratio-bar {
  height: 6px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
  margin-top: 8px;
}

.ratio-fill {
  height: 100%;
  transition: width 0.5s ease;
}

.info-section,
.seeding-section,
.explanation-section {
  margin-bottom: 32px;
}

.info-section h2,
.seeding-section h2,
.explanation-section h2 {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 16px;
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
}

.info-label {
  color: var(--text-secondary);
}

.info-value {
  font-weight: 600;
}

.seeding-list {
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 1px solid var(--border);
  overflow: hidden;
}

.seeding-item {
  padding: 16px;
  border-bottom: 1px solid var(--border);
}

.seeding-item:last-child {
  border-bottom: none;
}

.seeding-cid {
  font-family: monospace;
  font-size: 0.9rem;
  margin-bottom: 8px;
  display: block;
}

.seeding-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.explanation-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 20px;
  border: 1px solid var(--border);
}

.explanation-card p {
  margin-bottom: 16px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.explanation-card ul {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.explanation-card li {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ratio-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.ratio-indicator.good {
  background: var(--success);
}
.ratio-indicator.okay {
  background: var(--warning);
}
.ratio-indicator.low {
  background: var(--error);
}
</style>
