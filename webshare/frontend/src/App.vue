<script setup>
import { ref, onMounted } from 'vue'
import { RouterView } from 'vue-router'

const stats = ref({
  totalFiles: 0,
  totalDownloads: 0,
  totalSizeHuman: '0 B'
})

const fetchStats = async () => {
  try {
    const res = await fetch('/api/v1/stats')
    if (res.ok) {
      stats.value = await res.json()
    }
  } catch (err) {
    console.error('Failed to fetch stats:', err)
  }
}

onMounted(() => {
  fetchStats()
  // Refresh stats every 30 seconds
  setInterval(fetchStats, 30000)
})
</script>

<template>
  <header class="header">
    <div class="header-content">
      <router-link to="/" class="logo">
        <span class="logo-icon">⚡</span>
        <span>Torrentium Share</span>
      </router-link>
      <div class="header-stats">
        <div class="stat-item">
          <div class="stat-value">{{ stats.totalFiles }}</div>
          <div class="stat-label">Files</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ stats.totalDownloads }}</div>
          <div class="stat-label">Downloads</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ stats.totalSizeHuman }}</div>
          <div class="stat-label">Total Size</div>
        </div>
      </div>
    </div>
  </header>

  <main class="main-content">
    <RouterView />
  </main>

  <footer class="footer">
    <p>
      Torrentium Web Share Portal • Decentralized File Sharing •
      <a href="https://github.com/1amkhush/torrentium" target="_blank">GitHub</a>
    </p>
  </footer>
</template>
