<script setup>
import { ref, computed, onMounted, onUnmounted, inject } from "vue";
import {
  GetLocalFiles,
  AddFile,
  SelectFile,
  SelectDirectory,
  CopyToClipboard,
  AnnounceFile,
  OpenFileLocation,
  PublishToWeb,
  GenerateMagnetLink,
  CopyMagnetLink,
  GetWebShareConfig,
} from "../../wailsjs/go/main/App";
import { EventsOn, EventsOff, OnFileDrop } from "../../wailsjs/runtime/runtime";
import FilePreview from "./FilePreview.vue";

const addToast = inject('addToast');

const files = ref([]);
const loading = ref(true);
const isAdding = ref(false);
const searchQuery = ref('');
const isDragging = ref(false);
const showPreview = ref(false);
const previewCid = ref(null);
const dropUnsubscribe = ref(null);

// Publish dialog state
const showPublishDialog = ref(false);
const publishingCid = ref(null);
const publishingFile = ref(null);
const isPublishing = ref(false);
const publishForm = ref({
  description: '',
  category: 'other',
  tags: '',
  visibility: 'public',
  expiresIn: 0
});

const categories = [
  { value: 'documents', label: 'Documents' },
  { value: 'images', label: 'Images' },
  { value: 'videos', label: 'Videos' },
  { value: 'audio', label: 'Audio' },
  { value: 'archives', label: 'Archives' },
  { value: 'software', label: 'Software' },
  { value: 'code', label: 'Code' },
  { value: 'other', label: 'Other' }
];

const expirationOptions = [
  { value: 0, label: 'Never' },
  { value: 24, label: '24 hours' },
  { value: 72, label: '3 days' },
  { value: 168, label: '1 week' },
  { value: 720, label: '30 days' }
];

const filteredFiles = computed(() => {
  if (!searchQuery.value) return files.value;
  const query = searchQuery.value.toLowerCase();
  return files.value.filter(f => 
    f.filename.toLowerCase().includes(query) ||
    f.cid.toLowerCase().includes(query)
  );
});

const loadFiles = async () => {
  try {
    files.value = await GetLocalFiles();
  } catch (err) {
    addToast("Failed to load files: " + err, "error");
  } finally {
    loading.value = false;
  }
};

const addFile = async () => {
  try {
    const path = await SelectFile();
    if (path) {
      isAdding.value = true;
      const cid = await AddFile(path);
      addToast(`File added successfully!`, "success");
      await loadFiles();
    }
  } catch (err) {
    addToast("Failed to add file: " + err, "error");
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
      addToast(`Directory added successfully!`, "success");
      await loadFiles();
    }
  } catch (err) {
    addToast("Failed to add directory: " + err, "error");
  } finally {
    isAdding.value = false;
  }
};

const addFileByPath = async (path) => {
  try {
    isAdding.value = true;
    const cid = await AddFile(path);
    addToast(`File added successfully!`, "success");
    await loadFiles();
    return cid;
  } catch (err) {
    addToast("Failed to add file: " + err, "error");
    throw err;
  } finally {
    isAdding.value = false;
  }
};

const copyCID = async (cid) => {
  try {
    await CopyToClipboard(cid);
    addToast("CID copied to clipboard!", "success");
  } catch (err) {
    addToast("Failed to copy: " + err, "error");
  }
};

const announceFile = async (cid) => {
  try {
    await AnnounceFile(cid);
    addToast("File re-announced to DHT!", "success");
  } catch (err) {
    addToast("Failed to announce: " + err, "error");
  }
};

const openLocation = async (filePath) => {
  try {
    await OpenFileLocation(filePath);
  } catch (err) {
    addToast("Failed to open location: " + err, "error");
  }
};

const openPreview = (cid) => {
  previewCid.value = cid;
  showPreview.value = true;
};

const closePreview = () => {
  showPreview.value = false;
  previewCid.value = null;
};

// Publish to Web functions
const openPublishDialog = (file) => {
  publishingCid.value = file.cid;
  publishingFile.value = file;
  publishForm.value = {
    description: '',
    category: 'other',
    tags: '',
    visibility: 'public',
    expiresIn: 0
  };
  showPublishDialog.value = true;
};

const closePublishDialog = () => {
  showPublishDialog.value = false;
  publishingCid.value = null;
  publishingFile.value = null;
  isPublishing.value = false;
};

const publishFile = async () => {
  if (!publishingCid.value) return;
  
  isPublishing.value = true;
  try {
    const tags = publishForm.value.tags
      .split(',')
      .map(t => t.trim())
      .filter(t => t.length > 0);
    
    const result = await PublishToWeb(
      publishingCid.value,
      publishForm.value.description,
      publishForm.value.category,
      tags,
      publishForm.value.visibility,
      publishForm.value.expiresIn
    );
    
    if (result.success) {
      addToast("File published successfully!", "success");
      if (result.magnetLink) {
        await CopyToClipboard(result.magnetLink);
        addToast("Magnet link copied to clipboard!", "success");
      }
    }
    closePublishDialog();
  } catch (err) {
    addToast("Failed to publish: " + err, "error");
  } finally {
    isPublishing.value = false;
  }
};

const copyMagnetLink = async (cid) => {
  try {
    await CopyMagnetLink(cid);
    addToast("Magnet link copied to clipboard!", "success");
  } catch (err) {
    addToast("Failed to generate magnet link: " + err, "error");
  }
};

const getFileIcon = (filename) => {
  const ext = filename.split('.').pop()?.toLowerCase();
  const icons = {
    pdf: '📕', doc: '📘', docx: '📘', txt: '📄',
    jpg: '🖼️', jpeg: '🖼️', png: '🖼️', gif: '🖼️', svg: '🖼️', webp: '🖼️',
    mp3: '🎵', wav: '🎵', flac: '🎵', ogg: '🎵',
    mp4: '🎬', mkv: '🎬', avi: '🎬', mov: '🎬', webm: '🎬',
    zip: '📦', rar: '📦', '7z': '📦', tar: '📦', gz: '📦',
    js: '⚡', ts: '⚡', py: '🐍', go: '🔵', rs: '🦀',
    html: '🌐', css: '🎨', json: '📋', xml: '📋',
    exe: '⚙️', dll: '⚙️', iso: '💿',
  };
  return icons[ext] || '📄';
};

const handleDragOver = (e) => {
  e.preventDefault();
  isDragging.value = true;
};

const handleDragLeave = (e) => {
  // Only set to false if we're leaving the drop zone entirely
  const rect = e.currentTarget.getBoundingClientRect();
  if (
    e.clientX < rect.left ||
    e.clientX > rect.right ||
    e.clientY < rect.top ||
    e.clientY > rect.bottom
  ) {
    isDragging.value = false;
  }
};

const handleDrop = async (e) => {
  e.preventDefault();
  isDragging.value = false;
  // HTML5 file drop won't give us full paths for security reasons
  // Wails uses OnFileDrop for native file drop with full paths
  addToast("Processing dropped files...", "info");
};

// Handle native file drops from Wails
const handleFileDrop = async (x, y, filePaths) => {
  if (!filePaths || filePaths.length === 0) {
    return;
  }

  isDragging.value = false;
  const count = filePaths.length;
  addToast(`Processing ${count} file${count > 1 ? 's' : ''}...`, "info");

  let successCount = 0;
  let failCount = 0;

  for (const path of filePaths) {
    try {
      await addFileByPath(path);
      successCount++;
    } catch (err) {
      failCount++;
      console.error(`Failed to add file ${path}:`, err);
    }
  }

  if (successCount > 0 && failCount === 0) {
    addToast(`Successfully added ${successCount} file${successCount > 1 ? 's' : ''}!`, "success");
  } else if (successCount > 0 && failCount > 0) {
    addToast(`Added ${successCount} file${successCount > 1 ? 's' : ''}, ${failCount} failed`, "warning");
  } else if (failCount > 0) {
    addToast(`Failed to add ${failCount} file${failCount > 1 ? 's' : ''}`, "error");
  }
};

onMounted(() => {
  loadFiles();
  EventsOn("fileAdded", () => loadFiles());
  
  // Register for native file drops from Wails
  // OnFileDrop returns an unsubscribe function
  if (typeof OnFileDrop === 'function') {
    dropUnsubscribe.value = OnFileDrop(handleFileDrop, true);
  }
});

onUnmounted(() => {
  EventsOff("fileAdded");
  // Unsubscribe from file drop events
  if (dropUnsubscribe.value && typeof dropUnsubscribe.value === 'function') {
    dropUnsubscribe.value();
  }
});
</script>

<template>
  <div class="files-page" @dragover="handleDragOver" @dragleave="handleDragLeave" @drop="handleDrop">
    <!-- Drag Overlay -->
    <Transition name="fade">
      <div v-if="isDragging" class="drag-overlay">
        <div class="drag-content">
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="17 8 12 3 7 8"/>
            <line x1="12" y1="3" x2="12" y2="15"/>
          </svg>
          <h3>Drop files here to share</h3>
          <p>Files will be added to your shared content</p>
        </div>
      </div>
    </Transition>

    <header class="page-header">
      <div>
        <h1>My Files</h1>
        <p class="subtitle">Share files and directories on the network</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-primary" @click="addFile" :disabled="isAdding">
          <svg v-if="!isAdding" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="12" y1="18" x2="12" y2="12"/>
            <line x1="9" y1="15" x2="15" y2="15"/>
          </svg>
          <span class="spinner-small" v-if="isAdding"></span>
          <span>{{ isAdding ? 'Adding...' : 'Add File' }}</span>
        </button>
        <button class="btn btn-secondary" @click="addDirectory" :disabled="isAdding">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            <line x1="12" y1="11" x2="12" y2="17"/>
            <line x1="9" y1="14" x2="15" y2="14"/>
          </svg>
          <span>Add Folder</span>
        </button>
      </div>
    </header>

    <!-- Search Bar -->
    <div class="search-container" v-if="files.length > 0">
      <div class="search-box">
        <svg class="search-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input 
          type="text" 
          v-model="searchQuery" 
          placeholder="Search files by name or CID..."
          class="search-input"
        />
        <span v-if="searchQuery" class="search-clear" @click="searchQuery = ''">✕</span>
      </div>
      <div class="file-count">
        {{ filteredFiles.length }} of {{ files.length }} files
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading your files...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="files.length === 0" class="empty-state">
      <svg class="empty-icon" width="80" height="80" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
      </svg>
      <h3>No files shared yet</h3>
      <p>Start sharing by adding files or directories</p>
      <div class="empty-actions">
        <button class="btn btn-primary" @click="addFile">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Add Your First File
        </button>
      </div>
    </div>

    <!-- No Results -->
    <div v-else-if="filteredFiles.length === 0" class="empty-state">
      <svg class="empty-icon" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="11" cy="11" r="8"/>
        <line x1="21" y1="21" x2="16.65" y2="16.65"/>
      </svg>
      <h3>No matching files</h3>
      <p>Try a different search term</p>
    </div>

    <!-- Files Grid -->
    <div v-else class="files-grid">
      <div class="file-card" v-for="file in filteredFiles" :key="file.cid">
        <div class="file-card-header">
          <span class="file-type-icon">{{ getFileIcon(file.filename) }}</span>
          <div class="file-card-actions">
            <button class="action-btn" @click="openPreview(file.cid)" title="Preview">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
            </button>
            <button class="action-btn" @click="copyCID(file.cid)" title="Copy CID">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
              </svg>
            </button>
            <button class="action-btn" @click="copyMagnetLink(file.cid)" title="Copy Magnet Link">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
              </svg>
            </button>
            <button class="action-btn action-btn-publish" @click="openPublishDialog(file)" title="Publish to Web">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="2" y1="12" x2="22" y2="12"/>
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
              </svg>
            </button>
            <button class="action-btn" @click="announceFile(file.cid)" title="Re-announce">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                <path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"/>
              </svg>
            </button>
            <button class="action-btn" @click="openLocation(file.filePath)" title="Open location">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
            </button>
          </div>
        </div>
        <div class="file-card-body">
          <h4 class="file-card-name" :title="file.filename">{{ file.filename }}</h4>
          <p class="file-card-path" :title="file.filePath">{{ file.filePath }}</p>
        </div>
        <div class="file-card-footer">
          <div class="file-meta-item">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            </svg>
            <span>{{ file.sizeHuman }}</span>
          </div>
          <div class="file-meta-item">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
            <span>{{ file.createdAt }}</span>
          </div>
        </div>
        <div class="file-cid">
          <span class="cid-label">CID:</span>
          <code class="cid-value">{{ file.cid.slice(0, 24) }}...</code>
        </div>
      </div>
    </div>

    <!-- File Preview Modal -->
    <FilePreview 
      :cid="previewCid" 
      :show="showPreview" 
      @close="closePreview" 
    />

    <!-- Publish to Web Dialog -->
    <Transition name="fade">
      <div v-if="showPublishDialog" class="publish-overlay" @click.self="closePublishDialog">
        <div class="publish-dialog">
          <div class="publish-header">
            <h3>📡 Publish to Web</h3>
            <button class="close-btn" @click="closePublishDialog">✕</button>
          </div>
          
          <div class="publish-body" v-if="publishingFile">
            <div class="file-preview-row">
              <span class="file-icon">{{ getFileIcon(publishingFile.filename) }}</span>
              <div class="file-preview-info">
                <div class="file-preview-name">{{ publishingFile.filename }}</div>
                <div class="file-preview-size">{{ publishingFile.sizeHuman }}</div>
              </div>
            </div>

            <div class="form-group">
              <label for="description">Description (optional)</label>
              <textarea 
                id="description" 
                v-model="publishForm.description" 
                placeholder="Add a description for your file..."
                rows="3"
              ></textarea>
            </div>

            <div class="form-row">
              <div class="form-group">
                <label for="category">Category</label>
                <select id="category" v-model="publishForm.category">
                  <option v-for="cat in categories" :key="cat.value" :value="cat.value">
                    {{ cat.label }}
                  </option>
                </select>
              </div>

              <div class="form-group">
                <label for="visibility">Visibility</label>
                <select id="visibility" v-model="publishForm.visibility">
                  <option value="public">🌐 Public (searchable)</option>
                  <option value="unlisted">🔗 Unlisted (link only)</option>
                </select>
              </div>
            </div>

            <div class="form-group">
              <label for="tags">Tags (comma separated)</label>
              <input 
                id="tags" 
                type="text" 
                v-model="publishForm.tags" 
                placeholder="e.g., music, album, 2024"
              />
            </div>

            <div class="form-group">
              <label for="expires">Expiration</label>
              <select id="expires" v-model="publishForm.expiresIn">
                <option v-for="opt in expirationOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </div>
          </div>

          <div class="publish-footer">
            <button class="btn btn-secondary" @click="closePublishDialog" :disabled="isPublishing">
              Cancel
            </button>
            <button class="btn btn-primary" @click="publishFile" :disabled="isPublishing">
              <span class="spinner-small" v-if="isPublishing"></span>
              <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="2" y1="12" x2="22" y2="12"/>
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
              </svg>
              {{ isPublishing ? 'Publishing...' : 'Publish' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.files-page {
  position: relative;
  animation: fadeIn 0.4s ease;
  --wails-drop-target: drop; /* Enable Wails file drop for this component */
  min-height: 100%;
}

/* Show visual indicator when Wails signals a drop target */
.files-page[style*="--wails-drop-target: drop"] {
  background: var(--accent-glow);
}

/* Drag Overlay */
.drag-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(4px);
  border: 4px dashed var(--accent);
  margin: 8px;
  border-radius: var(--radius-lg);
}

.drag-content {
  text-align: center;
  color: var(--accent);
  animation: pulse 2s ease-in-out infinite;
}

.drag-content svg {
  margin-bottom: 16px;
  opacity: 0.8;
}

.drag-content h3 {
  font-size: 1.5rem;
  margin-bottom: 8px;
  color: var(--text-primary);
}

.drag-content p {
  color: var(--text-secondary);
}

/* Page Header */
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

.header-actions {
  display: flex;
  gap: 12px;
}

.btn {
  padding: 12px 20px;
  border-radius: var(--radius-md);
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all var(--transition-normal);
  display: flex;
  align-items: center;
  gap: 10px;
}

.btn-primary {
  background: var(--gradient-primary);
  color: var(--bg-primary);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: var(--shadow-glow);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-card);
  color: var(--text-primary);
  border: 1px solid var(--border);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-tertiary);
  border-color: var(--border-light);
}

.spinner-small {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(0,0,0,0.2);
  border-top-color: var(--bg-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Search Container */
.search-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
  gap: 16px;
}

.search-box {
  flex: 1;
  max-width: 400px;
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 14px;
  color: var(--text-muted);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 12px 40px 12px 44px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 0.9rem;
  transition: all var(--transition-fast);
}

.search-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-glow);
  outline: none;
}

.search-input::placeholder {
  color: var(--text-muted);
}

.search-clear {
  position: absolute;
  right: 14px;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 0.9rem;
  transition: color var(--transition-fast);
}

.search-clear:hover {
  color: var(--text-primary);
}

.file-count {
  font-size: 0.85rem;
  color: var(--text-muted);
  white-space: nowrap;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 40px;
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

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 80px 40px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 2px dashed var(--border);
  animation: fadeIn 0.5s ease;
}

.empty-icon {
  color: var(--text-muted);
  margin-bottom: 20px;
  opacity: 0.5;
}

.empty-state h3 {
  font-size: 1.3rem;
  margin-bottom: 8px;
  color: var(--text-primary);
}

.empty-state p {
  color: var(--text-secondary);
  margin-bottom: 24px;
}

.empty-actions {
  display: flex;
  justify-content: center;
}

/* Files Grid */
.files-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.file-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 20px;
  transition: all var(--transition-normal);
  animation: fadeIn 0.4s ease;
}

.file-card:hover {
  border-color: var(--accent);
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg);
}

.file-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.file-type-icon {
  font-size: 2.5rem;
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
}

.file-card-actions {
  display: flex;
  gap: 6px;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--bg-tertiary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
}

.action-btn:hover {
  background: var(--accent);
  color: var(--bg-primary);
}

.action-btn-publish:hover {
  background: #00d4aa;
}

.file-card-body {
  margin-bottom: 16px;
}

.file-card-name {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary);
}

.file-card-path {
  font-size: 0.75rem;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-card-footer {
  display: flex;
  gap: 16px;
  margin-bottom: 14px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border);
}

.file-meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.file-meta-item svg {
  color: var(--text-muted);
}

.file-cid {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-tertiary);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
}

.cid-label {
  font-size: 0.7rem;
  color: var(--text-muted);
  text-transform: uppercase;
  font-weight: 600;
}

.cid-value {
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 0.75rem;
  color: var(--accent);
  background: none;
  padding: 0;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Publish Dialog Styles */
.publish-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  backdrop-filter: blur(4px);
}

.publish-dialog {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  animation: slideUp 0.3s ease;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.publish-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.publish-header h3 {
  font-size: 1.2rem;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--bg-tertiary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 1rem;
  transition: all var(--transition-fast);
}

.close-btn:hover {
  background: var(--border);
  color: var(--text-primary);
}

.publish-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.file-preview-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  margin-bottom: 20px;
}

.file-preview-row .file-icon {
  font-size: 2rem;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-card);
  border-radius: var(--radius-sm);
}

.file-preview-info {
  flex: 1;
  min-width: 0;
}

.file-preview-name {
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-preview-size {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.form-group input,
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: 0.9rem;
  font-family: inherit;
  transition: all var(--transition-fast);
}

.form-group input:focus,
.form-group textarea:focus,
.form-group select:focus {
  border-color: var(--accent);
  outline: none;
  box-shadow: 0 0 0 3px var(--accent-glow);
}

.form-group textarea {
  resize: vertical;
  min-height: 80px;
}

.form-group select {
  cursor: pointer;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.publish-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--border);
  background: var(--bg-secondary);
}

.publish-footer .btn {
  padding: 10px 20px;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Responsive Styles */
@media (max-width: 900px) {
  .files-grid {
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .page-header h1 {
    font-size: 1.5rem;
  }

  .header-actions {
    display: flex;
    width: 100%;
    gap: 10px;
  }

  .header-actions .btn {
    flex: 1;
    padding: 12px 14px;
    font-size: 0.85rem;
  }

  .search-container {
    flex-direction: column;
    align-items: stretch;
  }

  .search-box {
    max-width: none;
  }

  .file-count {
    text-align: right;
  }

  .files-grid {
    gap: 12px;
  }

  .file-card {
    padding: 16px;
  }

  .file-type-icon {
    width: 48px;
    height: 48px;
    font-size: 2rem;
  }

  .file-card-footer {
    flex-wrap: wrap;
    gap: 12px;
  }

  .empty-state {
    padding: 50px 20px;
  }

  .empty-icon {
    width: 60px;
    height: 60px;
  }
}

@media (max-width: 480px) {
  .files-grid {
    grid-template-columns: 1fr;
  }

  .header-actions {
    flex-direction: column;
  }

  .header-actions .btn span:not(.spinner-small) {
    display: inline;
  }

  .file-card-header {
    margin-bottom: 12px;
  }

  .file-cid {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .drag-content h3 {
    font-size: 1.1rem;
  }

  .drag-content p {
    font-size: 0.85rem;
  }
}
</style>
