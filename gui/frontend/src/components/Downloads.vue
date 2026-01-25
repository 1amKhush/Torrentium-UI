<script setup>
import { ref, onMounted, inject, computed, onUnmounted } from "vue";
import {
  GetDownloads,
  DownloadFile,
  SearchByText,
  CopyToClipboard,
  GetDownloadQueue,
  PauseDownload,
  ResumeDownload,
  CancelDownload,
  SetDownloadPriority,
  SetDownloadBandwidth,
} from "../../wailsjs/go/main/App";
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";

const addToast = inject("addToast");

const downloads = ref([]);
const searchResults = ref([]);
const downloadQueue = ref([]);
const loading = ref(true);
const cidInput = ref("");
const searchQuery = ref("");
const isDownloading = ref(false);
const isSearching = ref(false);
const activeDownloads = ref({});
const activeTab = ref("download"); // 'download', 'search', 'queue', 'history'
const showPriorityModal = ref(false);
const showBandwidthModal = ref(false);
const selectedDownload = ref(null);
const newPriority = ref(0);
const newBandwidth = ref(0);

const activeDownloadsList = computed(() =>
  Object.entries(activeDownloads.value),
);
const hasActiveDownloads = computed(() => activeDownloadsList.value.length > 0);

// Queue polling interval
let queuePollInterval = null;

const loadDownloads = async () => {
  try {
    downloads.value = await GetDownloads();
  } catch (err) {
    addToast("Failed to load downloads: " + err, "error");
  } finally {
    loading.value = false;
  }
};

const loadQueue = async () => {
  try {
    const queue = await GetDownloadQueue();
    downloadQueue.value = queue || [];
  } catch (err) {
    console.error("Failed to load queue:", err);
  }
};

const startDownload = async () => {
  if (!cidInput.value.trim()) {
    addToast("Please enter a CID", "warning");
    return;
  }

  try {
    isDownloading.value = true;
    activeDownloads.value[cidInput.value] = { status: "starting", progress: 0 };
    await DownloadFile(cidInput.value.trim());
    cidInput.value = "";
    loadQueue();
  } catch (err) {
    addToast("Download failed: " + err, "error");
    delete activeDownloads.value[cidInput.value];
  } finally {
    isDownloading.value = false;
  }
};

const pauseDownloadItem = async (cid) => {
  try {
    await PauseDownload(cid);
    addToast("Download paused", "info");
    loadQueue();
  } catch (err) {
    addToast("Failed to pause: " + err, "error");
  }
};

const resumeDownloadItem = async (cid) => {
  try {
    await ResumeDownload(cid);
    addToast("Download resumed", "success");
    loadQueue();
  } catch (err) {
    addToast("Failed to resume: " + err, "error");
  }
};

const cancelDownloadItem = async (cid) => {
  try {
    await CancelDownload(cid);
    addToast("Download cancelled", "info");
    loadQueue();
    delete activeDownloads.value[cid];
  } catch (err) {
    addToast("Failed to cancel: " + err, "error");
  }
};

const openPriorityModal = (download) => {
  selectedDownload.value = download;
  newPriority.value = download.priority || 0;
  showPriorityModal.value = true;
};

const savePriority = async () => {
  if (selectedDownload.value) {
    try {
      await SetDownloadPriority(selectedDownload.value.cid, newPriority.value);
      addToast("Priority updated", "success");
      showPriorityModal.value = false;
      loadQueue();
    } catch (err) {
      addToast("Failed to set priority: " + err, "error");
    }
  }
};

const openBandwidthModal = (download) => {
  selectedDownload.value = download;
  newBandwidth.value = Math.floor((download.bandwidthLimit || 0) / 1024); // KB/s
  showBandwidthModal.value = true;
};

const saveBandwidth = async () => {
  if (selectedDownload.value) {
    try {
      await SetDownloadBandwidth(
        selectedDownload.value.cid,
        newBandwidth.value * 1024,
      );
      addToast("Bandwidth limit updated", "success");
      showBandwidthModal.value = false;
      loadQueue();
    } catch (err) {
      addToast("Failed to set bandwidth: " + err, "error");
    }
  }
};

const searchFiles = async () => {
  if (!searchQuery.value.trim()) {
    searchResults.value = [];
    return;
  }

  try {
    isSearching.value = true;
    searchResults.value = await SearchByText(searchQuery.value.trim());
    if (searchResults.value.length === 0) {
      addToast("No files found matching your search", "info");
    }
  } catch (err) {
    addToast("Search failed: " + err, "error");
  } finally {
    isSearching.value = false;
  }
};

const downloadFromSearch = async (cid) => {
  try {
    activeDownloads.value[cid] = { status: "starting", progress: 0 };
    await DownloadFile(cid);
    addToast("Download started!", "success");
    loadQueue();
  } catch (err) {
    addToast("Download failed: " + err, "error");
    delete activeDownloads.value[cid];
  }
};

const copyCID = async (cid) => {
  try {
    await CopyToClipboard(cid);
    addToast("CID copied!", "success");
  } catch (err) {
    addToast("Failed to copy", "error");
  }
};

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
  };
  return icons[ext] || "📄";
};

const getStatusIcon = (status) => {
  const icons = {
    pending: "⏳",
    downloading: "⬇️",
    paused: "⏸️",
    completed: "✅",
    failed: "❌",
  };
  return icons[status] || "❓";
};

const getStatusColor = (status) => {
  const colors = {
    pending: "var(--text-muted)",
    downloading: "var(--accent)",
    paused: "var(--warning)",
    completed: "var(--success)",
    failed: "var(--error)",
  };
  return colors[status] || "var(--text-secondary)";
};

const formatSpeed = (bytesPerSec) => {
  if (!bytesPerSec || bytesPerSec === 0) return "0 B/s";
  if (bytesPerSec < 1024) return `${bytesPerSec} B/s`;
  if (bytesPerSec < 1024 * 1024)
    return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
  return `${(bytesPerSec / (1024 * 1024)).toFixed(2)} MB/s`;
};

const formatETA = (seconds) => {
  if (!seconds || seconds <= 0) return "--";
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600)
    return `${Math.floor(seconds / 60)}m ${Math.round(seconds % 60)}s`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
};

onMounted(() => {
  loadDownloads();
  loadQueue();

  // Poll queue every 2 seconds
  queuePollInterval = setInterval(loadQueue, 2000);

  EventsOn("downloadStarted", (cid) => {
    activeDownloads.value[cid] = { status: "downloading", progress: 0 };
    loadQueue();
  });

  EventsOn("downloadComplete", (cid) => {
    delete activeDownloads.value[cid];
    addToast("Download complete!", "success");
    loadDownloads();
    loadQueue();
  });

  EventsOn("downloadError", (data) => {
    delete activeDownloads.value[data.cid];
    addToast("Download failed: " + data.error, "error");
    loadQueue();
  });

  EventsOn("downloadProgress", (data) => {
    if (activeDownloads.value[data.cid]) {
      activeDownloads.value[data.cid].progress = data.progress;
      activeDownloads.value[data.cid].speed = data.speed;
    }
  });
});

onUnmounted(() => {
  if (queuePollInterval) {
    clearInterval(queuePollInterval);
  }
  EventsOff("downloadStarted");
  EventsOff("downloadComplete");
  EventsOff("downloadError");
  EventsOff("downloadProgress");
});
</script>

<template>
  <div class="downloads-page">
    <header class="page-header">
      <div>
        <h1>Downloads</h1>
        <p class="subtitle">Download files from the P2P network</p>
      </div>
      <div v-if="hasActiveDownloads" class="active-badge">
        <span class="pulse-dot"></span>
        {{ activeDownloadsList.length }} active
      </div>
    </header>

    <!-- Tab Navigation -->
    <div class="tab-navigation">
      <button
        :class="['tab-btn', { active: activeTab === 'download' }]"
        @click="activeTab = 'download'"
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
        Download
      </button>
      <button
        :class="['tab-btn', { active: activeTab === 'search' }]"
        @click="activeTab = 'search'"
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
        Search
      </button>
      <button
        :class="['tab-btn', { active: activeTab === 'queue' }]"
        @click="activeTab = 'queue'"
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <line x1="8" y1="6" x2="21" y2="6" />
          <line x1="8" y1="12" x2="21" y2="12" />
          <line x1="8" y1="18" x2="21" y2="18" />
          <line x1="3" y1="6" x2="3.01" y2="6" />
          <line x1="3" y1="12" x2="3.01" y2="12" />
          <line x1="3" y1="18" x2="3.01" y2="18" />
        </svg>
        Queue
        <span v-if="downloadQueue.length" class="count-badge">{{
          downloadQueue.length
        }}</span>
      </button>
      <button
        :class="['tab-btn', { active: activeTab === 'history' }]"
        @click="activeTab = 'history'"
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="12" cy="12" r="10" />
          <polyline points="12 6 12 12 16 14" />
        </svg>
        History
        <span v-if="downloads.length" class="count-badge">{{
          downloads.length
        }}</span>
      </button>
    </div>

    <!-- Active Downloads Queue -->
    <Transition name="slide-down">
      <section v-if="hasActiveDownloads" class="active-downloads-section">
        <h3 class="section-label">
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          Download Queue
        </h3>
        <div class="queue-list">
          <div
            class="queue-item"
            v-for="[cid, dl] in activeDownloadsList"
            :key="cid"
          >
            <div class="queue-icon">
              <svg
                class="spin"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M21 12a9 9 0 1 1-6.219-8.56" />
              </svg>
            </div>
            <div class="queue-info">
              <span class="queue-cid">{{ cid.slice(0, 32) }}...</span>
              <div class="queue-progress">
                <div class="progress-track">
                  <div
                    class="progress-fill"
                    :style="{ width: dl.progress + '%' }"
                  ></div>
                </div>
                <span class="progress-text">{{ dl.status }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </Transition>

    <!-- Download Tab -->
    <section v-if="activeTab === 'download'" class="content-section">
      <div class="download-card">
        <div class="card-icon">
          <svg
            width="32"
            height="32"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
        </div>
        <h2>Download by CID</h2>
        <p>
          Enter a Content Identifier (CID) to download a file from the network
        </p>
        <div class="input-wrapper">
          <input
            v-model="cidInput"
            type="text"
            placeholder="Enter CID (e.g., bafybeig...)"
            class="cid-input"
            @keyup.enter="startDownload"
          />
          <button
            class="download-btn"
            @click="startDownload"
            :disabled="isDownloading || !cidInput.trim()"
          >
            <span v-if="isDownloading" class="spinner-small"></span>
            <svg
              v-else
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
            {{ isDownloading ? "Starting..." : "Download" }}
          </button>
        </div>
      </div>
    </section>

    <!-- Search Tab -->
    <section v-if="activeTab === 'search'" class="content-section">
      <div class="search-card">
        <div class="search-header">
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <h2>Search Local Index</h2>
        </div>
        <p>Search for files available from connected peers</p>
        <div class="search-wrapper">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search by filename..."
            class="search-input"
            @keyup.enter="searchFiles"
          />
          <button
            class="search-btn"
            @click="searchFiles"
            :disabled="isSearching"
          >
            <span v-if="isSearching" class="spinner-small"></span>
            <span v-else>Search</span>
          </button>
        </div>

        <!-- Search Results -->
        <Transition name="fade">
          <div v-if="searchResults.length > 0" class="results-container">
            <div class="results-header">
              <span class="results-count"
                >{{ searchResults.length }} results found</span
              >
            </div>
            <div class="results-list">
              <div
                class="result-card"
                v-for="result in searchResults"
                :key="result.cid"
              >
                <span class="result-icon">{{
                  getFileIcon(result.filename)
                }}</span>
                <div class="result-details">
                  <span class="result-name">{{ result.filename }}</span>
                  <code class="result-cid"
                    >{{ result.cid.slice(0, 28) }}...</code
                  >
                </div>
                <div class="result-actions">
                  <button
                    class="icon-btn"
                    @click="copyCID(result.cid)"
                    title="Copy CID"
                  >
                    <svg
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <rect x="9" y="9" width="13" height="13" rx="2" />
                      <path
                        d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                      />
                    </svg>
                  </button>
                  <button
                    class="action-btn"
                    @click="downloadFromSearch(result.cid)"
                  >
                    <svg
                      width="14"
                      height="14"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                      <polyline points="7 10 12 15 17 10" />
                      <line x1="12" y1="15" x2="12" y2="3" />
                    </svg>
                    Download
                  </button>
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </section>

    <!-- Queue Tab -->
    <section v-if="activeTab === 'queue'" class="content-section">
      <div v-if="downloadQueue.length === 0" class="empty-state">
        <svg
          class="empty-icon"
          width="64"
          height="64"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1"
        >
          <line x1="8" y1="6" x2="21" y2="6" />
          <line x1="8" y1="12" x2="21" y2="12" />
          <line x1="8" y1="18" x2="21" y2="18" />
          <line x1="3" y1="6" x2="3.01" y2="6" />
          <line x1="3" y1="12" x2="3.01" y2="12" />
          <line x1="3" y1="18" x2="3.01" y2="18" />
        </svg>
        <h3>Download queue is empty</h3>
        <p>Start a download to see it appear here</p>
        <button class="btn-primary" @click="activeTab = 'download'">
          Add Download
        </button>
      </div>

      <div v-else class="queue-management">
        <div class="queue-header">
          <h3>
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="8" y1="6" x2="21" y2="6" />
              <line x1="8" y1="12" x2="21" y2="12" />
              <line x1="8" y1="18" x2="21" y2="18" />
              <line x1="3" y1="6" x2="3.01" y2="6" />
              <line x1="3" y1="12" x2="3.01" y2="12" />
              <line x1="3" y1="18" x2="3.01" y2="18" />
            </svg>
            Download Queue
          </h3>
          <span class="queue-count">{{ downloadQueue.length }} items</span>
        </div>

        <div class="queue-items">
          <div
            class="queue-item-card"
            v-for="dl in downloadQueue"
            :key="dl.cid"
          >
            <div class="queue-item-header">
              <div class="queue-item-icon">
                <span class="status-icon">{{ getStatusIcon(dl.status) }}</span>
              </div>
              <div class="queue-item-info">
                <span class="queue-item-name">{{
                  dl.filename || "Unknown file"
                }}</span>
                <code class="queue-item-cid">{{ dl.cid.slice(0, 32) }}...</code>
              </div>
              <div
                class="queue-item-status"
                :style="{ color: getStatusColor(dl.status) }"
              >
                {{ dl.status }}
              </div>
            </div>

            <div class="queue-item-progress">
              <div class="progress-bar-container">
                <div
                  class="progress-bar-fill"
                  :style="{
                    width: dl.progress + '%',
                    background:
                      dl.status === 'paused'
                        ? 'var(--warning)'
                        : 'var(--gradient-primary)',
                  }"
                ></div>
              </div>
              <div class="progress-stats">
                <span class="progress-percent"
                  >{{ dl.progress?.toFixed(1) || 0 }}%</span
                >
                <span class="progress-speed" v-if="dl.status === 'downloading'">
                  {{ formatSpeed(dl.speed) }}
                </span>
                <span
                  class="progress-eta"
                  v-if="dl.status === 'downloading' && dl.eta > 0"
                >
                  ETA: {{ formatETA(dl.eta) }}
                </span>
              </div>
            </div>

            <div class="queue-item-details">
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
                    d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8"
                  />
                </svg>
                {{ dl.sizeHuman || "Unknown size" }}
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
                  <polygon
                    points="12 2 15 8.5 22 9.3 17 14 18.2 21 12 17.8 5.8 21 7 14 2 9.3 9 8.5 12 2"
                  />
                </svg>
                Priority: {{ dl.priority || 0 }}
              </span>
              <span class="detail-item" v-if="dl.bandwidthLimit > 0">
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"
                  />
                </svg>
                Limit: {{ formatSpeed(dl.bandwidthLimit) }}
              </span>
            </div>

            <div class="queue-item-actions">
              <button
                v-if="dl.status === 'downloading' || dl.status === 'pending'"
                class="action-btn pause-btn"
                @click="pauseDownloadItem(dl.cid)"
                title="Pause download"
              >
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect x="6" y="4" width="4" height="16" />
                  <rect x="14" y="4" width="4" height="16" />
                </svg>
                Pause
              </button>
              <button
                v-if="dl.status === 'paused'"
                class="action-btn resume-btn"
                @click="resumeDownloadItem(dl.cid)"
                title="Resume download"
              >
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <polygon points="5 3 19 12 5 21 5 3" />
                </svg>
                Resume
              </button>
              <button
                class="action-btn priority-btn"
                @click="openPriorityModal(dl)"
                title="Set priority"
              >
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <polygon
                    points="12 2 15 8.5 22 9.3 17 14 18.2 21 12 17.8 5.8 21 7 14 2 9.3 9 8.5 12 2"
                  />
                </svg>
                Priority
              </button>
              <button
                class="action-btn bandwidth-btn"
                @click="openBandwidthModal(dl)"
                title="Set bandwidth limit"
              >
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"
                  />
                </svg>
                Limit
              </button>
              <button
                class="action-btn cancel-btn"
                @click="cancelDownloadItem(dl.cid)"
                title="Cancel download"
              >
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
                Cancel
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Priority Modal -->
    <Transition name="fade">
      <div
        v-if="showPriorityModal"
        class="modal-overlay"
        @click.self="showPriorityModal = false"
      >
        <div class="modal-content">
          <h3>Set Download Priority</h3>
          <p>Higher priority downloads will be processed first.</p>
          <div class="modal-input-group">
            <label>Priority (0-10)</label>
            <input
              type="number"
              v-model.number="newPriority"
              min="0"
              max="10"
              class="modal-input"
            />
            <input
              type="range"
              v-model.number="newPriority"
              min="0"
              max="10"
              class="priority-slider"
            />
          </div>
          <div class="modal-actions">
            <button class="modal-btn cancel" @click="showPriorityModal = false">
              Cancel
            </button>
            <button class="modal-btn confirm" @click="savePriority">
              Save
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Bandwidth Modal -->
    <Transition name="fade">
      <div
        v-if="showBandwidthModal"
        class="modal-overlay"
        @click.self="showBandwidthModal = false"
      >
        <div class="modal-content">
          <h3>Set Bandwidth Limit</h3>
          <p>Limit download speed for this file. Set to 0 for unlimited.</p>
          <div class="modal-input-group">
            <label>Bandwidth (KB/s)</label>
            <input
              type="number"
              v-model.number="newBandwidth"
              min="0"
              class="modal-input"
              placeholder="0 = unlimited"
            />
          </div>
          <div class="modal-actions">
            <button
              class="modal-btn cancel"
              @click="showBandwidthModal = false"
            >
              Cancel
            </button>
            <button class="modal-btn confirm" @click="saveBandwidth">
              Save
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- History Tab -->
    <section v-if="activeTab === 'history'" class="content-section">
      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
        <p>Loading download history...</p>
      </div>

      <div v-else-if="downloads.length === 0" class="empty-state">
        <svg
          class="empty-icon"
          width="64"
          height="64"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1"
        >
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
          <polyline points="7 10 12 15 17 10" />
          <line x1="12" y1="15" x2="12" y2="3" />
        </svg>
        <h3>No downloads yet</h3>
        <p>Downloaded files will appear here</p>
        <button class="btn-primary" @click="activeTab = 'download'">
          Start Downloading
        </button>
      </div>

      <div v-else class="history-list">
        <div class="history-item" v-for="dl in downloads" :key="dl.cid">
          <div class="history-icon-wrapper">
            <span class="file-icon">{{ getFileIcon(dl.filename) }}</span>
            <svg
              class="check-badge"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="currentColor"
            >
              <circle cx="12" cy="12" r="10" />
              <path
                d="M9 12l2 2 4-4"
                stroke="white"
                stroke-width="2"
                fill="none"
              />
            </svg>
          </div>
          <div class="history-info">
            <span class="history-name">{{ dl.filename }}</span>
            <div class="history-meta">
              <span class="meta-item">
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8"
                  />
                </svg>
                {{ dl.sizeHuman }}
              </span>
              <span class="meta-item">
                <svg
                  width="12"
                  height="12"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <circle cx="12" cy="12" r="10" />
                  <polyline points="12 6 12 12 16 14" />
                </svg>
                {{ dl.downloadedAt }}
              </span>
            </div>
            <span class="history-path">{{ dl.downloadPath }}</span>
          </div>
          <button class="icon-btn" @click="copyCID(dl.cid)" title="Copy CID">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <rect x="9" y="9" width="13" height="13" rx="2" />
              <path
                d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
              />
            </svg>
          </button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.downloads-page {
  width: 100%;
  animation: fadeIn 0.4s ease;
}

/* Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 28px;
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

.active-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--accent-glow);
  border-radius: var(--radius-xl);
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--accent);
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  animation: pulse 2s ease-in-out infinite;
}

/* Tab Navigation */
.tab-navigation {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  padding: 6px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
}

.tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.9rem;
  font-weight: 500;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab-btn:hover {
  color: var(--text-primary);
  background: var(--bg-tertiary);
}

.tab-btn.active {
  background: var(--gradient-primary);
  color: var(--bg-primary);
}

.tab-btn svg {
  width: 18px;
  height: 18px;
}

.count-badge {
  background: var(--bg-tertiary);
  padding: 2px 8px;
  border-radius: var(--radius-xl);
  font-size: 0.75rem;
}

.tab-btn.active .count-badge {
  background: rgba(0, 0, 0, 0.2);
}

/* Active Downloads Section */
.active-downloads-section {
  background: var(--bg-card);
  border: 1px solid var(--accent);
  border-radius: var(--radius-lg);
  padding: 20px;
  margin-bottom: 24px;
}

.section-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--accent);
  margin-bottom: 16px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.queue-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.queue-item {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
}

.queue-icon {
  color: var(--accent);
}

.queue-icon .spin {
  animation: spin 1s linear infinite;
}

.queue-info {
  flex: 1;
}

.queue-cid {
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 0.8rem;
  color: var(--text-secondary);
  display: block;
  margin-bottom: 8px;
}

.queue-progress {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-track {
  flex: 1;
  height: 6px;
  background: var(--bg-primary);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--gradient-primary);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: capitalize;
}

/* Content Section */
.content-section {
  animation: fadeIn 0.3s ease;
}

/* Download Card */
.download-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 40px;
  text-align: center;
}

.card-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto 20px;
  background: var(--accent-glow);
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
}

.download-card h2 {
  font-size: 1.3rem;
  margin-bottom: 8px;
}

.download-card p {
  color: var(--text-secondary);
  margin-bottom: 28px;
  font-size: 0.9rem;
}

.input-wrapper {
  display: flex;
  gap: 12px;
  max-width: 600px;
  margin: 0 auto;
}

.cid-input {
  flex: 1;
  padding: 14px 18px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 0.95rem;
  font-family: "SF Mono", "Consolas", monospace;
}

.cid-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-glow);
}

.cid-input::placeholder {
  font-family: inherit;
  color: var(--text-muted);
}

.download-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 24px;
  background: var(--gradient-primary);
  color: var(--bg-primary);
  border: none;
  border-radius: var(--radius-md);
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-normal);
}

.download-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: var(--shadow-glow);
}

.download-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinner-small {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(0, 0, 0, 0.2);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Search Card */
.search-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 28px;
}

.search-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
  color: var(--accent);
}

.search-header h2 {
  color: var(--text-primary);
  font-size: 1.2rem;
}

.search-card > p {
  color: var(--text-secondary);
  margin-bottom: 20px;
  font-size: 0.9rem;
}

.search-wrapper {
  display: flex;
  gap: 12px;
}

.search-input {
  flex: 1;
  padding: 12px 16px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 0.95rem;
}

.search-input:focus {
  outline: none;
  border-color: var(--accent);
}

.search-btn {
  padding: 12px 24px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.search-btn:hover:not(:disabled) {
  background: var(--border);
}

/* Results */
.results-container {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--border);
}

.results-header {
  margin-bottom: 16px;
}

.results-count {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.result-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}

.result-card:hover {
  background: var(--bg-primary);
}

.result-icon {
  font-size: 1.5rem;
}

.result-details {
  flex: 1;
  min-width: 0;
}

.result-name {
  display: block;
  font-weight: 500;
  margin-bottom: 4px;
}

.result-cid {
  font-size: 0.75rem;
  color: var(--accent);
  background: none;
  padding: 0;
}

.result-actions {
  display: flex;
  gap: 8px;
}

.icon-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.icon-btn:hover {
  background: var(--accent);
  color: var(--bg-primary);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: var(--accent);
  color: var(--bg-primary);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.action-btn:hover {
  transform: translateY(-1px);
}

/* History */
.history-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 20px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  transition: all var(--transition-fast);
}

.history-item:hover {
  border-color: var(--border-light);
  transform: translateX(4px);
}

.history-icon-wrapper {
  position: relative;
}

.file-icon {
  font-size: 2rem;
}

.check-badge {
  position: absolute;
  bottom: -4px;
  right: -4px;
  color: var(--success);
}

.history-info {
  flex: 1;
  min-width: 0;
}

.history-name {
  display: block;
  font-weight: 600;
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.history-meta {
  display: flex;
  gap: 16px;
  margin-bottom: 6px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.history-path {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-family: "SF Mono", "Consolas", monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

/* Empty & Loading States */
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

.empty-state {
  text-align: center;
  padding: 60px 40px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 2px dashed var(--border);
}

.empty-icon {
  color: var(--text-muted);
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-state h3 {
  font-size: 1.2rem;
  margin-bottom: 8px;
}

.empty-state p {
  color: var(--text-secondary);
  margin-bottom: 20px;
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  background: var(--gradient-primary);
  color: var(--bg-primary);
  border: none;
  border-radius: var(--radius-md);
  font-weight: 500;
  cursor: pointer;
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

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
}

.slide-down-enter-from,
.slide-down-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

/* Queue Management Styles */
.queue-management {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 24px;
}

.queue-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.queue-header h3 {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.queue-count {
  font-size: 0.85rem;
  color: var(--text-muted);
  background: var(--bg-tertiary);
  padding: 4px 12px;
  border-radius: var(--radius-xl);
}

.queue-items {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.queue-item-card {
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 18px;
  transition: all var(--transition-fast);
}

.queue-item-card:hover {
  border-color: var(--accent);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.queue-item-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
}

.queue-item-icon {
  font-size: 1.5rem;
}

.queue-item-info {
  flex: 1;
  min-width: 0;
}

.queue-item-name {
  display: block;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.queue-item-cid {
  font-family: "SF Mono", "Consolas", monospace;
  font-size: 0.75rem;
  color: var(--text-muted);
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.queue-item-status {
  font-size: 0.85rem;
  font-weight: 500;
  text-transform: capitalize;
  padding: 4px 12px;
  background: var(--bg-primary);
  border-radius: var(--radius-xl);
}

.queue-item-progress {
  margin-bottom: 14px;
}

.progress-bar-container {
  height: 8px;
  background: var(--bg-primary);
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 8px;
}

.progress-bar-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-stats {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.progress-percent {
  font-weight: 600;
  color: var(--accent);
}

.progress-speed {
  color: var(--success);
}

.progress-eta {
  color: var(--text-secondary);
}

.queue-item-details {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.queue-item-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.queue-item-actions .action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: 1px solid var(--border);
  background: var(--bg-primary);
  color: var(--text-secondary);
  font-size: 0.8rem;
  font-weight: 500;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.queue-item-actions .action-btn:hover {
  color: var(--text-primary);
  border-color: var(--accent);
}

.queue-item-actions .pause-btn:hover {
  background: var(--warning);
  color: var(--bg-primary);
  border-color: var(--warning);
}

.queue-item-actions .resume-btn:hover {
  background: var(--success);
  color: var(--bg-primary);
  border-color: var(--success);
}

.queue-item-actions .cancel-btn:hover {
  background: var(--error);
  color: var(--bg-primary);
  border-color: var(--error);
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-content {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 28px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
}

.modal-content h3 {
  font-size: 1.2rem;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--text-primary);
}

.modal-content p {
  color: var(--text-secondary);
  font-size: 0.9rem;
  margin-bottom: 20px;
}

.modal-input-group {
  margin-bottom: 24px;
}

.modal-input-group label {
  display: block;
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.modal-input {
  width: 100%;
  padding: 12px 16px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 1rem;
  transition: all var(--transition-fast);
}

.modal-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-glow);
}

.priority-slider {
  width: 100%;
  margin-top: 12px;
  accent-color: var(--accent);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.modal-btn {
  padding: 10px 20px;
  border-radius: var(--radius-md);
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.modal-btn.cancel {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-secondary);
}

.modal-btn.cancel:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.modal-btn.confirm {
  background: var(--gradient-primary);
  border: none;
  color: var(--bg-primary);
}

.modal-btn.confirm:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Responsive Styles */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .page-header h1 {
    font-size: 1.5rem;
  }

  .active-badge {
    padding: 6px 12px;
    font-size: 0.8rem;
  }

  .tab-navigation {
    flex-wrap: wrap;
    gap: 6px;
    padding: 6px;
  }

  .tab-btn {
    padding: 10px 12px;
    font-size: 0.85rem;
    gap: 6px;
  }

  .tab-btn svg {
    width: 16px;
    height: 16px;
  }

  .download-card {
    padding: 24px 16px;
  }

  .card-icon {
    width: 60px;
    height: 60px;
  }

  .download-card h2 {
    font-size: 1.1rem;
  }

  .input-wrapper {
    flex-direction: column;
  }

  .search-card {
    padding: 24px 16px;
  }

  .search-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .results-grid {
    grid-template-columns: 1fr;
  }

  .history-section {
    padding: 0;
  }

  .history-list {
    gap: 12px;
  }

  .history-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    padding: 14px;
  }

  .history-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .queue-item {
    padding: 12px;
  }

  .queue-cid {
    font-size: 0.75rem;
  }
}

@media (max-width: 480px) {
  .tab-navigation {
    flex-direction: column;
  }

  .tab-btn {
    justify-content: flex-start;
    padding: 12px 16px;
  }

  .result-card {
    padding: 14px;
  }

  .empty-state {
    padding: 40px 20px;
  }
}
</style>
