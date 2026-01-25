<script setup>
import { ref, computed, watch } from "vue";
import { GetFilePreview } from "../../wailsjs/go/main/App";

const props = defineProps({
  cid: {
    type: String,
    default: null
  },
  show: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['close']);

const preview = ref(null);
const loading = ref(false);
const error = ref(null);
const textContent = ref(null);

// Watch for CID changes to load preview
watch(() => props.cid, async (newCid) => {
  if (newCid && props.show) {
    await loadPreview(newCid);
  }
}, { immediate: true });

watch(() => props.show, async (visible) => {
  if (visible && props.cid) {
    await loadPreview(props.cid);
  } else if (!visible) {
    // Reset state when closing
    preview.value = null;
    textContent.value = null;
    error.value = null;
  }
});

const loadPreview = async (cid) => {
  if (!cid) return;
  
  loading.value = true;
  error.value = null;
  textContent.value = null;
  
  try {
    preview.value = await GetFilePreview(cid);
    
    // For text files, we could fetch the content if needed
    // This is a simplified version - actual implementation would need
    // a backend endpoint to read file content
    if (preview.value?.fileType === 'document' && 
        (preview.value?.mimeType === 'text/plain' || preview.value?.mimeType === 'text/markdown')) {
      // Text preview would require reading file content from backend
      textContent.value = 'Text preview is available. Click to open the file.';
    }
  } catch (err) {
    error.value = err.toString();
  } finally {
    loading.value = false;
  }
};

const close = () => {
  emit('close');
};

const handleBackdropClick = (e) => {
  if (e.target.classList.contains('preview-overlay')) {
    close();
  }
};

const getFileIcon = computed(() => {
  if (!preview.value) return '📄';
  const type = preview.value.fileType;
  const icons = {
    image: '🖼️',
    video: '🎬',
    audio: '🎵',
    document: '📄',
    other: '📎'
  };
  return icons[type] || '📄';
});

const formatSize = (bytes) => {
  if (!bytes) return '0 B';
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`;
};
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="show" class="preview-overlay" @click="handleBackdropClick">
        <div class="preview-modal">
          <!-- Header -->
          <div class="preview-header">
            <div class="preview-title">
              <span class="file-icon">{{ getFileIcon }}</span>
              <span class="filename">{{ preview?.filename || 'File Preview' }}</span>
            </div>
            <button class="close-btn" @click="close" title="Close">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>

          <!-- Content -->
          <div class="preview-content">
            <!-- Loading -->
            <div v-if="loading" class="preview-loading">
              <div class="spinner"></div>
              <p>Loading preview...</p>
            </div>

            <!-- Error -->
            <div v-else-if="error" class="preview-error">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              <p>{{ error }}</p>
            </div>

            <!-- Preview -->
            <div v-else-if="preview" class="preview-view">
              <!-- Image Preview -->
              <div v-if="preview.fileType === 'image'" class="image-preview">
                <img 
                  :src="preview.previewURL" 
                  :alt="preview.filename"
                  @error="error = 'Failed to load image'"
                />
              </div>

              <!-- Video Preview -->
              <div v-else-if="preview.fileType === 'video'" class="video-preview">
                <video 
                  :src="preview.previewURL" 
                  controls
                  @error="error = 'Failed to load video'"
                >
                  Your browser does not support the video tag.
                </video>
              </div>

              <!-- Audio Preview -->
              <div v-else-if="preview.fileType === 'audio'" class="audio-preview">
                <div class="audio-icon">🎵</div>
                <audio 
                  :src="preview.previewURL" 
                  controls
                  @error="error = 'Failed to load audio'"
                >
                  Your browser does not support the audio tag.
                </audio>
              </div>

              <!-- Document Preview (PDF) -->
              <div v-else-if="preview.fileType === 'document' && preview.mimeType === 'application/pdf'" class="pdf-preview">
                <iframe 
                  :src="preview.previewURL" 
                  type="application/pdf"
                  @error="error = 'Failed to load PDF'"
                ></iframe>
              </div>

              <!-- Text Preview -->
              <div v-else-if="preview.fileType === 'document' && textContent" class="text-preview">
                <pre>{{ textContent }}</pre>
              </div>

              <!-- No Preview Available -->
              <div v-else class="no-preview">
                <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                  <polyline points="14 2 14 8 20 8"/>
                </svg>
                <h3>Preview not available</h3>
                <p>This file type cannot be previewed in the browser.</p>
              </div>
            </div>
          </div>

          <!-- Footer with file info -->
          <div v-if="preview && !loading" class="preview-footer">
            <div class="file-info">
              <span class="info-item">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8"/>
                </svg>
                {{ preview.sizeHuman || formatSize(preview.fileSize) }}
              </span>
              <span class="info-item">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                  <polyline points="14 2 14 8 20 8"/>
                </svg>
                {{ preview.mimeType || 'Unknown type' }}
              </span>
            </div>
            <code class="cid-display">{{ preview.cid?.slice(0, 32) }}...</code>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.preview-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(8px);
  padding: 24px;
}

.preview-modal {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 900px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-tertiary);
}

.preview-title {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.file-icon {
  font-size: 1.5rem;
}

.filename {
  font-weight: 600;
  font-size: 1rem;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.close-btn {
  width: 40px;
  height: 40px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.close-btn:hover {
  background: var(--bg-primary);
  color: var(--error);
}

.preview-content {
  flex: 1;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  min-height: 300px;
  background: var(--bg-primary);
}

.preview-loading,
.preview-error,
.no-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  color: var(--text-secondary);
  text-align: center;
}

.preview-error svg {
  color: var(--error);
}

.spinner {
  width: 48px;
  height: 48px;
  border: 3px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.preview-view {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Image Preview */
.image-preview {
  max-width: 100%;
  max-height: 100%;
}

.image-preview img {
  max-width: 100%;
  max-height: 60vh;
  object-fit: contain;
  border-radius: var(--radius-md);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

/* Video Preview */
.video-preview {
  width: 100%;
  max-width: 800px;
}

.video-preview video {
  width: 100%;
  max-height: 60vh;
  border-radius: var(--radius-md);
  background: #000;
}

/* Audio Preview */
.audio-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  padding: 40px;
}

.audio-icon {
  font-size: 64px;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 0.8; }
  50% { transform: scale(1.1); opacity: 1; }
}

.audio-preview audio {
  width: 100%;
  max-width: 400px;
}

/* PDF Preview */
.pdf-preview {
  width: 100%;
  height: 60vh;
}

.pdf-preview iframe {
  width: 100%;
  height: 100%;
  border: none;
  border-radius: var(--radius-md);
}

/* Text Preview */
.text-preview {
  width: 100%;
  max-height: 60vh;
  overflow: auto;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  padding: 20px;
}

.text-preview pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 0.9rem;
  color: var(--text-primary);
}

/* No Preview */
.no-preview svg {
  color: var(--text-muted);
  margin-bottom: 8px;
}

.no-preview h3 {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text-primary);
}

.no-preview p {
  margin: 0;
  font-size: 0.9rem;
}

/* Footer */
.preview-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
  background: var(--bg-tertiary);
}

.file-info {
  display: flex;
  gap: 20px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.cid-display {
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 0.75rem;
  color: var(--text-muted);
  background: var(--bg-primary);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
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

.fade-enter-active .preview-modal,
.fade-leave-active .preview-modal {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.fade-enter-from .preview-modal,
.fade-leave-to .preview-modal {
  opacity: 0;
  transform: scale(0.95);
}

/* Responsive */
@media (max-width: 768px) {
  .preview-overlay {
    padding: 12px;
  }

  .preview-modal {
    max-height: 95vh;
  }

  .preview-header {
    padding: 12px 16px;
  }

  .preview-content {
    padding: 16px;
    min-height: 200px;
  }

  .preview-footer {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }
}
</style>
