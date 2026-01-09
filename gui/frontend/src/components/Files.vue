<script setup>
import { ref, onMounted } from "vue";
import {
  GetLocalFiles,
  AddFile,
  SelectFile,
  SelectDirectory,
  CopyToClipboard,
  AnnounceFile,
  OpenFileLocation,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const files = ref([]);
const loading = ref(true);
const isAdding = ref(false);
const notification = ref(null);

const loadFiles = async () => {
  try {
    files.value = await GetLocalFiles();
  } catch (err) {
    showNotification("Failed to load files: " + err, "error");
  } finally {
    loading.value = false;
  }
};

const showNotification = (message, type = "success") => {
  notification.value = { message, type };
  setTimeout(() => (notification.value = null), 3000);
};

const addFile = async () => {
  try {
    const path = await SelectFile();
    if (path) {
      isAdding.value = true;
      const cid = await AddFile(path);
      showNotification(`File added! CID: ${cid.slice(0, 20)}...`);
      await loadFiles();
    }
  } catch (err) {
    showNotification("Failed to add file: " + err, "error");
  } finally {
    isAdding.value = false;
  }
};

const addDirectory = async () => {
  try {
    const path = await SelectDirectory();
    if (path) {
      isAdding.value = true;
      const cid = await AddFile(path);
      showNotification(`Directory added! CID: ${cid.slice(0, 20)}...`);
      await loadFiles();
    }
  } catch (err) {
    showNotification("Failed to add directory: " + err, "error");
  } finally {
    isAdding.value = false;
  }
};

const copyCID = async (cid) => {
  try {
    await CopyToClipboard(cid);
    showNotification("CID copied to clipboard!");
  } catch (err) {
    showNotification("Failed to copy: " + err, "error");
  }
};

const announceFile = async (cid) => {
  try {
    await AnnounceFile(cid);
    showNotification("File re-announced to DHT!");
  } catch (err) {
    showNotification("Failed to announce: " + err, "error");
  }
};

const openLocation = async (filePath) => {
  try {
    await OpenFileLocation(filePath);
  } catch (err) {
    showNotification("Failed to open location: " + err, "error");
  }
};

onMounted(() => {
  loadFiles();
  EventsOn("fileAdded", () => loadFiles());
});
</script>

<template>
  <div class="files-page">
    <!-- Notification -->
    <Transition name="slide">
      <div v-if="notification" :class="['notification', notification.type]">
        {{ notification.message }}
      </div>
    </Transition>

    <header class="page-header">
      <div>
        <h1>My Files</h1>
        <p class="subtitle">Share files and directories on the network</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-primary" @click="addFile" :disabled="isAdding">
          <span v-if="isAdding">Adding...</span>
          <span v-else>📄 Add File</span>
        </button>
        <button
          class="btn btn-secondary"
          @click="addDirectory"
          :disabled="isAdding"
        >
          📁 Add Directory
        </button>
      </div>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading files...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="files.length === 0" class="empty-state">
      <div class="empty-icon">📁</div>
      <h3>No files shared yet</h3>
      <p>Click "Add File" or "Add Directory" to start sharing</p>
    </div>

    <!-- Files List -->
    <div v-else class="files-container">
      <div class="files-table">
        <div class="table-header">
          <span class="col-name">Name</span>
          <span class="col-size">Size</span>
          <span class="col-cid">CID</span>
          <span class="col-date">Added</span>
          <span class="col-actions">Actions</span>
        </div>

        <div class="table-body">
          <div class="file-row" v-for="file in files" :key="file.cid">
            <div class="col-name">
              <span class="file-icon">📄</span>
              <div class="file-details">
                <span class="file-name" :title="file.filename">{{
                  file.filename
                }}</span>
                <span class="file-path">{{ file.filePath }}</span>
              </div>
            </div>
            <span class="col-size">{{ file.sizeHuman }}</span>
            <div class="col-cid">
              <span class="cid-text" :title="file.cid"
                >{{ file.cid.slice(0, 16) }}...</span
              >
              <button
                class="btn-icon"
                @click="copyCID(file.cid)"
                title="Copy CID"
              >
                📋
              </button>
            </div>
            <span class="col-date">{{ file.createdAt }}</span>
            <div class="col-actions">
              <button
                class="btn-icon"
                @click="announceFile(file.cid)"
                title="Re-announce to DHT"
              >
                📢
              </button>
              <button
                class="btn-icon"
                @click="openLocation(file.filePath)"
                title="Open file location"
              >
                📂
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.files-page {
  position: relative;
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
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
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

.header-actions {
  display: flex;
  gap: 12px;
}

.btn {
  padding: 12px 20px;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 8px;
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

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
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

.empty-state {
  text-align: center;
  padding: 80px 40px;
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 2px dashed var(--border);
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 16px;
}

.empty-state h3 {
  font-size: 1.3rem;
  margin-bottom: 8px;
}

.empty-state p {
  color: var(--text-secondary);
}

.files-table {
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 1px solid var(--border);
  overflow: hidden;
}

.table-header {
  display: grid;
  grid-template-columns: 2fr 100px 200px 120px 100px;
  padding: 16px 20px;
  background: var(--bg-tertiary);
  font-weight: 600;
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.file-row {
  display: grid;
  grid-template-columns: 2fr 100px 200px 120px 100px;
  padding: 16px 20px;
  align-items: center;
  border-bottom: 1px solid var(--border);
  transition: background 0.2s ease;
}

.file-row:hover {
  background: var(--bg-tertiary);
}

.file-row:last-child {
  border-bottom: none;
}

.col-name {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.file-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.file-details {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.file-name {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-path {
  font-size: 0.75rem;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.col-size {
  font-size: 0.9rem;
}

.col-cid {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cid-text {
  font-family: monospace;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.col-date {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.col-actions {
  display: flex;
  gap: 8px;
}

.btn-icon {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--bg-tertiary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.btn-icon:hover {
  background: var(--accent);
}
</style>
