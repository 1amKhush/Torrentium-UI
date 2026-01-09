<script setup>
import { ref, onMounted } from "vue";
import {
  GetDownloads,
  DownloadFile,
  SearchByText,
  CopyToClipboard,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const downloads = ref([]);
const searchResults = ref([]);
const loading = ref(true);
const cidInput = ref("");
const searchQuery = ref("");
const isDownloading = ref(false);
const isSearching = ref(false);
const activeDownloads = ref({});
const notification = ref(null);

const loadDownloads = async () => {
  try {
    downloads.value = await GetDownloads();
  } catch (err) {
    showNotification("Failed to load downloads: " + err, "error");
  } finally {
    loading.value = false;
  }
};

const showNotification = (message, type = "success") => {
  notification.value = { message, type };
  setTimeout(() => (notification.value = null), 3000);
};

const startDownload = async () => {
  if (!cidInput.value.trim()) {
    showNotification("Please enter a CID", "error");
    return;
  }

  try {
    isDownloading.value = true;
    activeDownloads.value[cidInput.value] = { status: "starting", progress: 0 };
    await DownloadFile(cidInput.value.trim());
    cidInput.value = "";
  } catch (err) {
    showNotification("Download failed: " + err, "error");
    delete activeDownloads.value[cidInput.value];
  } finally {
    isDownloading.value = false;
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
  } catch (err) {
    showNotification("Search failed: " + err, "error");
  } finally {
    isSearching.value = false;
  }
};

const downloadFromSearch = async (cid) => {
  try {
    activeDownloads.value[cid] = { status: "starting", progress: 0 };
    await DownloadFile(cid);
  } catch (err) {
    showNotification("Download failed: " + err, "error");
    delete activeDownloads.value[cid];
  }
};

const copyCID = async (cid) => {
  try {
    await CopyToClipboard(cid);
    showNotification("CID copied!");
  } catch (err) {
    showNotification("Failed to copy", "error");
  }
};

onMounted(() => {
  loadDownloads();

  EventsOn("downloadStarted", (cid) => {
    activeDownloads.value[cid] = { status: "downloading", progress: 0 };
  });

  EventsOn("downloadComplete", (cid) => {
    delete activeDownloads.value[cid];
    showNotification("Download complete!");
    loadDownloads();
  });

  EventsOn("downloadError", (data) => {
    delete activeDownloads.value[data.cid];
    showNotification("Download failed: " + data.error, "error");
  });
});
</script>

<template>
  <div class="downloads-page">
    <!-- Notification -->
    <Transition name="slide">
      <div v-if="notification" :class="['notification', notification.type]">
        {{ notification.message }}
      </div>
    </Transition>

    <header class="page-header">
      <h1>Downloads</h1>
      <p class="subtitle">Download files from the P2P network</p>
    </header>

    <!-- Download by CID Section -->
    <section class="download-section">
      <h2>Download by CID</h2>
      <div class="input-group">
        <input
          v-model="cidInput"
          type="text"
          placeholder="Enter CID (e.g., bafybeig...)"
          class="text-input"
          @keyup.enter="startDownload"
        />
        <button
          class="btn btn-primary"
          @click="startDownload"
          :disabled="isDownloading || !cidInput.trim()"
        >
          <span v-if="isDownloading">⏳ Starting...</span>
          <span v-else>⬇️ Download</span>
        </button>
      </div>
    </section>

    <!-- Search Section -->
    <section class="search-section">
      <h2>Search Local Index</h2>
      <div class="input-group">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search by filename..."
          class="text-input"
          @keyup.enter="searchFiles"
        />
        <button
          class="btn btn-secondary"
          @click="searchFiles"
          :disabled="isSearching"
        >
          <span v-if="isSearching">🔍 Searching...</span>
          <span v-else>🔍 Search</span>
        </button>
      </div>

      <!-- Search Results -->
      <div v-if="searchResults.length > 0" class="search-results">
        <h3>Search Results</h3>
        <div
          class="result-item"
          v-for="result in searchResults"
          :key="result.cid"
        >
          <div class="result-info">
            <span class="result-name">{{ result.filename }}</span>
            <span class="result-cid">{{ result.cid.slice(0, 24) }}...</span>
          </div>
          <div class="result-actions">
            <button
              class="btn-icon"
              @click="copyCID(result.cid)"
              title="Copy CID"
            >
              📋
            </button>
            <button class="btn-small" @click="downloadFromSearch(result.cid)">
              Download
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- Active Downloads -->
    <section
      v-if="Object.keys(activeDownloads).length > 0"
      class="active-downloads"
    >
      <h2>Active Downloads</h2>
      <div class="active-list">
        <div
          class="active-item"
          v-for="(dl, cid) in activeDownloads"
          :key="cid"
        >
          <div class="active-info">
            <span class="active-cid">{{ cid.slice(0, 24) }}...</span>
            <span class="active-status">{{ dl.status }}</span>
          </div>
          <div class="progress-bar">
            <div
              class="progress-fill"
              :style="{ width: dl.progress + '%' }"
            ></div>
          </div>
        </div>
      </div>
    </section>

    <!-- Download History -->
    <section class="history-section">
      <h2>Download History</h2>

      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
        <p>Loading...</p>
      </div>

      <div v-else-if="downloads.length === 0" class="empty-state">
        <div class="empty-icon">⬇️</div>
        <h3>No downloads yet</h3>
        <p>Enter a CID above to start downloading</p>
      </div>

      <div v-else class="downloads-list">
        <div class="download-item" v-for="dl in downloads" :key="dl.cid">
          <div class="download-icon">✅</div>
          <div class="download-info">
            <span class="download-name">{{ dl.filename }}</span>
            <span class="download-meta"
              >{{ dl.sizeHuman }} • {{ dl.downloadedAt }}</span
            >
            <span class="download-path">{{ dl.downloadPath }}</span>
          </div>
          <div class="download-actions">
            <button class="btn-icon" @click="copyCID(dl.cid)" title="Copy CID">
              📋
            </button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.downloads-page {
  position: relative;
  max-width: 1000px;
}

.notification {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 12px 20px;
  border-radius: 8px;
  font-size: 0.9rem;
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.notification.success {
  background: var(--success);
  color: white;
}
.notification.error {
  background: var(--error);
  color: white;
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}
.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
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

section {
  margin-bottom: 32px;
}

section h2 {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 16px;
}

.input-group {
  display: flex;
  gap: 12px;
}

.text-input {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 0.95rem;
}

.text-input:focus {
  outline: none;
  border-color: var(--accent);
}

.text-input::placeholder {
  color: var(--text-secondary);
}

.btn {
  padding: 12px 20px;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.btn-primary {
  background: var(--accent);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
}
.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border);
}

.btn-secondary:hover {
  background: var(--border);
}

.search-results {
  margin-top: 16px;
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 1px solid var(--border);
  padding: 16px;
}

.search-results h3 {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.result-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: var(--bg-tertiary);
  border-radius: 8px;
  margin-bottom: 8px;
}

.result-item:last-child {
  margin-bottom: 0;
}

.result-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.result-name {
  font-weight: 500;
}
.result-cid {
  font-family: monospace;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.result-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.btn-small {
  padding: 6px 12px;
  font-size: 0.8rem;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.btn-small:hover {
  background: var(--accent-hover);
}

.btn-icon {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--bg-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-icon:hover {
  background: var(--accent);
}

.active-downloads {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 20px;
  border: 1px solid var(--border);
}

.active-item {
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}

.active-item:last-child {
  border-bottom: none;
}

.active-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.active-cid {
  font-family: monospace;
  font-size: 0.85rem;
}
.active-status {
  color: var(--accent);
  font-size: 0.85rem;
}

.progress-bar {
  height: 4px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.3s ease;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px;
  color: var(--text-secondary);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 12px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.empty-state {
  text-align: center;
  padding: 60px 40px;
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 2px dashed var(--border);
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 12px;
}
.empty-state h3 {
  font-size: 1.2rem;
  margin-bottom: 8px;
}
.empty-state p {
  color: var(--text-secondary);
}

.downloads-list {
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 1px solid var(--border);
  overflow: hidden;
}

.download-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--border);
}

.download-item:last-child {
  border-bottom: none;
}
.download-item:hover {
  background: var(--bg-tertiary);
}

.download-icon {
  font-size: 1.5rem;
}

.download-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.download-name {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.download-meta {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.download-path {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
