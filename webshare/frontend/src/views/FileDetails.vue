<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const file = ref(null)
const loading = ref(true)
const error = ref(null)
const toast = ref(null)
const showReportModal = ref(false)
const reportReason = ref('')

const fetchFile = async () => {
  const cid = route.params.cid
  loading.value = true
  error.value = null

  try {
    const res = await fetch(`/api/v1/files/${cid}`)
    if (res.ok) {
      file.value = await res.json()
    } else if (res.status === 404) {
      error.value = 'File not found'
    } else {
      error.value = 'Failed to load file'
    }
  } catch (err) {
    error.value = 'Network error'
  } finally {
    loading.value = false
  }
}

const getFileIcon = (filename) => {
  const ext = filename?.split('.').pop()?.toLowerCase()
  const icons = {
    pdf: '📕', doc: '📘', docx: '📘', txt: '📄', md: '📝',
    jpg: '🖼️', jpeg: '🖼️', png: '🖼️', gif: '🖼️', svg: '🖼️', webp: '🖼️',
    mp3: '🎵', wav: '🎵', flac: '🎵', ogg: '🎵', m4a: '🎵',
    mp4: '🎬', mkv: '🎬', avi: '🎬', mov: '🎬', webm: '🎬',
    zip: '📦', rar: '📦', '7z': '📦', tar: '📦', gz: '📦',
    exe: '💿', dmg: '💿', iso: '💿', msi: '💿',
    js: '💻', ts: '💻', py: '🐍', go: '🔵', rs: '🦀'
  }
  return icons[ext] || '📄'
}

const formatDate = (dateStr) => {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const copyMagnetLink = async () => {
  try {
    await navigator.clipboard.writeText(file.value.magnetLink)
    showToast('Magnet link copied!', 'success')
  } catch (err) {
    showToast('Failed to copy', 'error')
  }
}

const openInTorrentium = () => {
  // Track download
  fetch(`/api/v1/files/${file.value.cid}/download`, { method: 'POST' })
  
  // Open magnet link - will be handled by Torrentium app if registered
  window.location.href = file.value.magnetLink
}

const copyShareLink = async () => {
  const url = window.location.href
  try {
    await navigator.clipboard.writeText(url)
    showToast('Share link copied!', 'success')
  } catch (err) {
    showToast('Failed to copy', 'error')
  }
}

const submitReport = async () => {
  if (!reportReason.value.trim()) {
    showToast('Please provide a reason', 'error')
    return
  }

  try {
    const res = await fetch(`/api/v1/files/${file.value.id}/report`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: reportReason.value })
    })

    if (res.ok) {
      showToast('Report submitted', 'success')
      showReportModal.value = false
      reportReason.value = ''
    } else {
      showToast('Failed to submit report', 'error')
    }
  } catch (err) {
    showToast('Network error', 'error')
  }
}

const showToast = (message, type = 'info') => {
  toast.value = { message, type }
  setTimeout(() => { toast.value = null }, 3000)
}

onMounted(fetchFile)
</script>

<template>
  <div class="file-details">
    <router-link to="/" class="back-link">
      ← Back to files
    </router-link>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading file details...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="empty-state">
      <div class="empty-icon">😕</div>
      <h3>{{ error }}</h3>
      <p>The file may have been removed or expired</p>
      <button class="btn btn-secondary" @click="router.push('/')">
        Browse other files
      </button>
    </div>

    <!-- File Details -->
    <div v-else-if="file" class="file-details-card">
      <div class="file-details-header">
        <div class="file-details-icon">{{ getFileIcon(file.filename) }}</div>
        <div class="file-details-info">
          <h1>{{ file.filename }}</h1>
          <div class="file-details-meta">
            <span class="file-category">{{ file.category }}</span>
            <span>📦 {{ file.sizeHuman }}</span>
            <span>⬇️ {{ file.downloads }} downloads</span>
            <span>👁️ {{ file.views }} views</span>
          </div>
        </div>
      </div>

      <p v-if="file.description" class="file-description-full">
        {{ file.description }}
      </p>

      <div v-if="file.tags && file.tags.length > 0" class="tags-section">
        <span v-for="tag in file.tags" :key="tag" class="tag">
          #{{ tag }}
        </span>
      </div>

      <!-- Magnet Link Section -->
      <div class="magnet-section">
        <div class="magnet-label">Magnet Link</div>
        <div class="magnet-link">
          <input
            type="text"
            class="magnet-input"
            :value="file.magnetLink"
            readonly
            @click="$event.target.select()"
          />
          <button class="btn btn-secondary" @click="copyMagnetLink">
            📋 Copy
          </button>
        </div>
      </div>

      <!-- CID Section -->
      <div class="magnet-section">
        <div class="magnet-label">Content ID (CID)</div>
        <div class="magnet-link">
          <input
            type="text"
            class="magnet-input"
            :value="file.cid"
            readonly
            @click="$event.target.select()"
          />
        </div>
      </div>

      <!-- Actions -->
      <div class="actions-row">
        <button class="btn btn-primary" @click="openInTorrentium">
          ⚡ Open in Torrentium
        </button>
        <button class="btn btn-secondary" @click="copyShareLink">
          🔗 Copy Share Link
        </button>
        <button class="btn btn-secondary" @click="showReportModal = true">
          🚩 Report
        </button>
      </div>

      <!-- File Info -->
      <div class="file-info-table">
        <div class="info-row">
          <span class="info-label">Published</span>
          <span class="info-value">{{ formatDate(file.publishedAt) }}</span>
        </div>
        <div v-if="file.expiresAt" class="info-row">
          <span class="info-label">Expires</span>
          <span class="info-value">{{ formatDate(file.expiresAt) }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">Size</span>
          <span class="info-value">{{ file.sizeHuman }} ({{ file.fileSize.toLocaleString() }} bytes)</span>
        </div>
      </div>
    </div>

    <!-- Report Modal -->
    <Teleport to="body">
      <div v-if="showReportModal" class="modal-overlay" @click.self="showReportModal = false">
        <div class="modal">
          <h3>Report File</h3>
          <p>Why are you reporting this file?</p>
          <textarea
            v-model="reportReason"
            placeholder="Please describe the issue..."
            rows="4"
          ></textarea>
          <div class="modal-actions">
            <button class="btn btn-secondary" @click="showReportModal = false">Cancel</button>
            <button class="btn btn-primary" @click="submitReport">Submit Report</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Toast -->
    <Transition name="slide">
      <div v-if="toast" class="toast" :class="toast.type">
        {{ toast.message }}
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.tags-section {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 24px;
}

.tag {
  padding: 4px 12px;
  background: var(--bg-tertiary);
  border-radius: 20px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.file-info-table {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--border);
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--border);
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  color: var(--text-muted);
}

.info-value {
  color: var(--text-primary);
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 24px;
  max-width: 480px;
  width: 90%;
}

.modal h3 {
  margin-bottom: 8px;
}

.modal p {
  color: var(--text-secondary);
  margin-bottom: 16px;
}

.modal textarea {
  width: 100%;
  padding: 12px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  resize: vertical;
  margin-bottom: 16px;
}

.modal textarea:focus {
  outline: none;
  border-color: var(--accent);
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
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
</style>
