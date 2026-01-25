<script setup>
import { ref, onMounted, onUnmounted, computed } from "vue";
import { GetStats, GetUploadProgress } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const stats = ref(null);
const uploadProgress = ref([]);
const loading = ref(true);
const bandwidthHistory = ref([]);
let intervalId = null;

const loadData = async () => {
  try {
    const [statsData, progressData] = await Promise.all([
      GetStats(),
      GetUploadProgress(),
    ]);
    stats.value = statsData;
    uploadProgress.value = progressData;

    // Track bandwidth history for graph
    if (statsData) {
      bandwidthHistory.value.push({
        time: Date.now(),
        upload: statsData.bytesUploaded || 0,
        download: statsData.bytesDownloaded || 0,
      });
      // Keep last 20 data points
      if (bandwidthHistory.value.length > 20) {
        bandwidthHistory.value.shift();
      }
    }
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

const ratioClass = computed(() => {
  if (!stats.value) return "";
  if (stats.value.ratio >= 1) return "good";
  if (stats.value.ratio >= 0.5) return "okay";
  return "low";
});

onMounted(() => {
  loadData();
  intervalId = setInterval(loadData, 5000);
  EventsOn("downloadComplete", loadData);
  EventsOn("fileAdded", loadData);
});

onUnmounted(() => {
  if (intervalId) clearInterval(intervalId);
});
</script>

<template>
  <div class="stats-page">
    <header class="page-header">
      <div>
        <h1>Statistics</h1>
        <p class="subtitle">Track your upload and download activity</p>
      </div>
      <div class="refresh-indicator">
        <svg
          class="spin-slow"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 12a9 9 0 1 1-6.219-8.56" />
        </svg>
        Auto-refresh
      </div>
    </header>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading statistics...</p>
    </div>

    <template v-else-if="stats">
      <!-- Main Stats Cards -->
      <div class="stats-overview">
        <div class="stat-card upload-card">
          <div class="stat-icon-wrapper upload">
            <svg
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="17 8 12 3 7 8" />
              <line x1="12" y1="3" x2="12" y2="15" />
            </svg>
          </div>
          <div class="stat-content">
            <span class="stat-label">Total Uploaded</span>
            <div class="stat-value">{{ stats.uploadedHuman }}</div>
            <div class="stat-details">
              <span class="detail-item">
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                </svg>
                {{ stats.chunksServed }} chunks
              </span>
              <span class="detail-item">
                <svg
                  width="12"
                  height="12"
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
                {{ stats.peersServed }} peers
              </span>
            </div>
          </div>
        </div>

        <div class="stat-card download-card">
          <div class="stat-icon-wrapper download">
            <svg
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
          </div>
          <div class="stat-content">
            <span class="stat-label">Total Downloaded</span>
            <div class="stat-value">{{ stats.downloadedHuman }}</div>
            <div class="stat-details">
              <span class="detail-item">
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"
                  />
                </svg>
                {{ stats.filesDownloaded }} files
              </span>
            </div>
          </div>
        </div>

        <div class="stat-card ratio-card">
          <div :class="['stat-icon-wrapper', 'ratio', ratioClass]">
            <svg
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="18" y1="20" x2="18" y2="10" />
              <line x1="12" y1="20" x2="12" y2="4" />
              <line x1="6" y1="20" x2="6" y2="14" />
            </svg>
          </div>
          <div class="stat-content">
            <span class="stat-label">Share Ratio</span>
            <div class="stat-value ratio-value" :style="{ color: ratioColor }">
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
        </div>

        <div class="stat-card files-card">
          <div class="stat-icon-wrapper files">
            <svg
              width="24"
              height="24"
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
          <div class="stat-content">
            <span class="stat-label">Files Shared</span>
            <div class="stat-value">{{ stats.filesShared }}</div>
            <div class="stat-details">
              <span class="detail-item active">Available on network</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Bandwidth Section -->
      <div class="bandwidth-section">
        <h2 class="section-title">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
          </svg>
          Bandwidth
        </h2>
        <div class="bandwidth-card">
          <div class="bandwidth-row">
            <div class="bandwidth-label">
              <span class="bw-indicator upload"></span>
              <span>Max Upload Rate</span>
            </div>
            <span class="bandwidth-value">{{ stats.maxUploadRateHuman }}</span>
          </div>
        </div>
      </div>

      <!-- Active Seeding -->
      <div class="seeding-section" v-if="uploadProgress.length > 0">
        <h2 class="section-title">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="17 8 12 3 7 8" />
            <line x1="12" y1="3" x2="12" y2="15" />
          </svg>
          Active Seeding
          <span class="count-badge">{{ uploadProgress.length }}</span>
        </h2>
        <div class="seeding-list">
          <div
            class="seeding-item"
            v-for="item in uploadProgress"
            :key="item.cid"
          >
            <div class="seeding-header">
              <code class="seeding-cid">{{ item.cid.slice(0, 32) }}...</code>
              <span class="speed-badge">{{ item.avgSpeed }}</span>
            </div>
            <div class="seeding-stats">
              <div class="seed-stat">
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="17 8 12 3 7 8" />
                  <line x1="12" y1="3" x2="12" y2="15" />
                </svg>
                {{ item.uploadedHuman }}
              </div>
              <div class="seed-stat">
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect x="3" y="3" width="18" height="18" rx="2" />
                </svg>
                {{ item.chunksServed }} chunks
              </div>
              <div class="seed-stat">
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                  <circle cx="9" cy="7" r="4" />
                </svg>
                {{ item.peersServed }} peers
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Ratio Explanation -->
      <div class="explanation-section">
        <h2 class="section-title">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="10" />
            <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
          About Share Ratio
        </h2>
        <div class="explanation-card">
          <p>
            Your share ratio is the amount of data you've uploaded divided by
            what you've downloaded.
          </p>
          <div class="ratio-legend">
            <div class="legend-item">
              <span class="ratio-indicator good"></span>
              <div class="legend-content">
                <strong>≥ 1.0 — Excellent</strong>
                <span>You're giving back as much or more than you take</span>
              </div>
            </div>
            <div class="legend-item">
              <span class="ratio-indicator okay"></span>
              <div class="legend-content">
                <strong>0.5 - 1.0 — Good</strong>
                <span>Good progress, keep seeding!</span>
              </div>
            </div>
            <div class="legend-item">
              <span class="ratio-indicator low"></span>
              <div class="legend-content">
                <strong>&lt; 0.5 — Needs Work</strong>
                <span>Consider seeding more to help the network</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.stats-page {
  width: 100%;
  animation: fadeIn 0.4s ease;
}

/* Header */
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
}

.refresh-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8rem;
  color: var(--text-muted);
  padding: 8px 14px;
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border);
}

.spin-slow {
  animation: spin 3s linear infinite;
}

/* Loading */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px;
  color: var(--text-secondary);
}

.spinner {
  width: 48px;
  height: 48px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

/* Stats Overview */
.stats-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

@media (max-width: 600px) {
  .stats-overview {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 24px;
  border: 1px solid var(--border);
  display: flex;
  gap: 20px;
  transition: all var(--transition-normal);
}

.stat-card:hover {
  border-color: var(--border-light);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-icon-wrapper {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon-wrapper.upload {
  background: var(--success-glow);
  color: var(--success);
}

.stat-icon-wrapper.download {
  background: var(--accent-glow);
  color: var(--accent-secondary);
}

.stat-icon-wrapper.ratio {
  background: var(--warning-glow);
  color: var(--warning);
}

.stat-icon-wrapper.ratio.good {
  background: var(--success-glow);
  color: var(--success);
}

.stat-icon-wrapper.ratio.low {
  background: var(--error-glow);
  color: var(--error);
}

.stat-icon-wrapper.files {
  background: rgba(168, 85, 247, 0.15);
  color: #a855f7;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  display: block;
  margin-bottom: 6px;
}

.stat-value {
  font-size: 1.8rem;
  font-weight: 700;
  line-height: 1.2;
  margin-bottom: 8px;
}

.stat-value.ratio-value {
  font-size: 2.2rem;
}

.stat-details {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.detail-item.active {
  color: var(--success);
}

.detail-item svg {
  opacity: 0.7;
}

.ratio-bar {
  height: 6px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
  margin-top: 4px;
}

.ratio-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.5s ease;
}

/* Section Titles */
.section-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-primary);
}

.section-title svg {
  color: var(--accent);
}

.count-badge {
  background: var(--bg-tertiary);
  padding: 4px 10px;
  border-radius: var(--radius-xl);
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Bandwidth Section */
.bandwidth-section {
  margin-bottom: 32px;
}

.bandwidth-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 20px;
  border: 1px solid var(--border);
}

.bandwidth-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.bandwidth-label {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-secondary);
}

.bw-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.bw-indicator.upload {
  background: var(--success);
}

.bandwidth-value {
  font-weight: 600;
  font-size: 1.1rem;
  color: var(--text-primary);
}

/* Seeding Section */
.seeding-section {
  margin-bottom: 32px;
}

.seeding-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.seeding-item {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 18px;
  transition: all var(--transition-fast);
}

.seeding-item:hover {
  border-color: var(--accent);
}

.seeding-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.seeding-cid {
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 0.8rem;
  color: var(--accent);
  background: none;
  padding: 0;
}

.speed-badge {
  background: var(--success-glow);
  color: var(--success);
  padding: 4px 12px;
  border-radius: var(--radius-xl);
  font-size: 0.75rem;
  font-weight: 600;
}

.seeding-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
}

.seed-stat {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.seed-stat svg {
  color: var(--text-muted);
}

/* Explanation Section */
.explanation-section {
  margin-bottom: 32px;
}

.explanation-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 24px;
  border: 1px solid var(--border);
}

.explanation-card p {
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 20px;
}

.ratio-legend {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.legend-item {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.ratio-indicator {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 3px;
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

.legend-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.legend-content strong {
  font-size: 0.9rem;
}

.legend-content span {
  font-size: 0.8rem;
  color: var(--text-muted);
}

/* Animations */
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
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
@media (max-width: 1024px) {
  .stats-overview {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-page {
    padding: 0;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .page-header h1 {
    font-size: 1.5rem;
  }

  .stats-overview {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .stat-card {
    padding: 18px;
  }

  .stat-icon-wrapper {
    width: 44px;
    height: 44px;
  }

  .stat-value {
    font-size: 1.5rem;
  }

  .stat-details {
    flex-wrap: wrap;
  }

  .bandwidth-card {
    padding: 16px;
  }

  .seeding-item {
    padding: 14px;
  }

  .seeding-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .seeding-stats {
    flex-wrap: wrap;
  }

  .explanation-card {
    padding: 18px;
  }

  .ratio-legend {
    gap: 12px;
  }
}

@media (max-width: 480px) {
  .stat-card {
    flex-direction: column;
    text-align: center;
    gap: 12px;
  }

  .stat-content {
    align-items: center;
  }

  .stat-details {
    justify-content: center;
  }

  .seeding-cid {
    font-size: 0.7rem;
    word-break: break-all;
  }

  .seed-stat {
    font-size: 0.75rem;
  }

  .legend-item {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }

  .loading-state {
    padding: 40px 20px;
  }
}
</style>
