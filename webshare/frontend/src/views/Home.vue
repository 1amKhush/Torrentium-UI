<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const files = ref([])
const loading = ref(true)
const search = ref('')
const category = ref('all')
const sortBy = ref('newest')
const page = ref(1)
const pagination = ref({ total: 0, totalPages: 1 })
const categories = ref([])
const toast = ref(null)

const allCategories = [
  { id: 'all', name: 'All', icon: '📁' },
  { id: 'documents', name: 'Documents', icon: '📄' },
  { id: 'images', name: 'Images', icon: '🖼️' },
  { id: 'videos', name: 'Videos', icon: '🎬' },
  { id: 'audio', name: 'Audio', icon: '🎵' },
  { id: 'archives', name: 'Archives', icon: '📦' },
  { id: 'software', name: 'Software', icon: '💿' },
  { id: 'code', name: 'Code', icon: '💻' },
  { id: 'other', name: 'Other', icon: '📎' }
]

const sortOptions = [
  { value: 'newest', label: 'Newest First' },
  { value: 'oldest', label: 'Oldest First' },
  { value: 'downloads', label: 'Most Downloads' },
  { value: 'views', label: 'Most Viewed' },
  { value: 'size', label: 'Largest First' },
  { value: 'name', label: 'Name (A-Z)' }
]

const fetchFiles = async () => {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: page.value,
      limit: 20,
      sort: sortBy.value
    })
    if (category.value !== 'all') params.set('category', category.value)
    if (search.value) params.set('search', search.value)

    const res = await fetch(`/api/v1/files?${params}`)
    if (res.ok) {
      const data = await res.json()
      files.value = data.files || []
      pagination.value = data.pagination
    }
  } catch (err) {
    showToast('Failed to load files', 'error')
  } finally {
    loading.value = false
  }
}

const getFileIcon = (filename, categoryId) => {
  const ext = filename.split('.').pop()?.toLowerCase()
  const icons = {
    pdf: '📕', doc: '📘', docx: '📘', txt: '📄', md: '📝',
    jpg: '🖼️', jpeg: '🖼️', png: '🖼️', gif: '🖼️', svg: '🖼️', webp: '🖼️',
    mp3: '🎵', wav: '🎵', flac: '🎵', ogg: '🎵', m4a: '🎵',
    mp4: '🎬', mkv: '🎬', avi: '🎬', mov: '🎬', webm: '🎬',
    zip: '📦', rar: '📦', '7z': '📦', tar: '📦', gz: '📦',
    exe: '💿', dmg: '💿', iso: '💿', msi: '💿',
    js: '💻', ts: '💻', py: '🐍', go: '🔵', rs: '🦀', java: '☕',
    html: '🌐', css: '🎨', json: '📋', xml: '📋'
  }
  return icons[ext] || allCategories.find(c => c.id === categoryId)?.icon || '📄'
}

const formatDate = (dateStr) => {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', { 
    year: 'numeric', 
    month: 'short', 
    day: 'numeric' 
  })
}

const goToFile = (cid) => {
  router.push(`/file/${cid}`)
}

const showToast = (message, type = 'info') => {
  toast.value = { message, type }
  setTimeout(() => { toast.value = null }, 3000)
}

const handleSearch = () => {
  page.value = 1
  fetchFiles()
}

watch([category, sortBy], () => {
  page.value = 1
  fetchFiles()
})

watch(page, fetchFiles)

onMounted(fetchFiles)
</script>

<template>
  <div class="home-page">
    <!-- Search Section -->
    <section class="search-section">
      <div class="search-bar">
        <input
          v-model="search"
          type="text"
          class="search-input"
          placeholder="Search files by name, description, or tags..."
          @keyup.enter="handleSearch"
        />
        <button class="btn btn-primary" @click="handleSearch">
          🔍 Search
        </button>
      </div>

      <div class="filters">
        <div class="category-pills">
          <button
            v-for="cat in allCategories"
            :key="cat.id"
            class="category-pill"
            :class="{ active: category === cat.id }"
            @click="category = cat.id"
          >
            {{ cat.icon }} {{ cat.name }}
          </button>
        </div>

        <select v-model="sortBy" class="filter-select">
          <option v-for="opt in sortOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
      </div>
    </section>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading files...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="files.length === 0" class="empty-state">
      <div class="empty-icon">📭</div>
      <h3>No files found</h3>
      <p v-if="search || category !== 'all'">Try adjusting your search or filters</p>
      <p v-else>Be the first to share a file!</p>
    </div>

    <!-- Files Grid -->
    <div v-else class="files-grid">
      <div
        v-for="file in files"
        :key="file.cid"
        class="file-card"
        @click="goToFile(file.cid)"
      >
        <div class="file-card-header">
          <div class="file-icon">{{ getFileIcon(file.filename, file.category) }}</div>
          <div class="file-info">
            <h3 class="file-name" :title="file.filename">{{ file.filename }}</h3>
            <span class="file-category">{{ file.category }}</span>
          </div>
        </div>
        <p v-if="file.description" class="file-description">{{ file.description }}</p>
        <div class="file-meta">
          <span class="meta-item">📦 {{ file.sizeHuman }}</span>
          <span class="meta-item">⬇️ {{ file.downloads }}</span>
          <span class="meta-item">👁️ {{ file.views }}</span>
          <span class="meta-item">📅 {{ formatDate(file.publishedAt) }}</span>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="pagination.totalPages > 1" class="pagination">
      <button
        class="page-btn"
        :disabled="page === 1"
        @click="page--"
      >
        ← Prev
      </button>
      
      <span class="page-info">
        Page {{ page }} of {{ pagination.totalPages }} ({{ pagination.total }} files)
      </span>
      
      <button
        class="page-btn"
        :disabled="page >= pagination.totalPages"
        @click="page++"
      >
        Next →
      </button>
    </div>

    <!-- Toast -->
    <Transition name="slide">
      <div v-if="toast" class="toast" :class="toast.type">
        {{ toast.message }}
      </div>
    </Transition>
  </div>
</template>

<style scoped>
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
