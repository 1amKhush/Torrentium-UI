<script setup>
import { computed } from "vue";

const props = defineProps({
  currentView: String,
  networkStatus: Object,
  theme: String,
});

const emit = defineEmits(["navigate", "toggleTheme"]);

const menuItems = [
  { id: "dashboard", label: "Dashboard", icon: "dashboard" },
  { id: "files", label: "My Files", icon: "folder" },
  { id: "downloads", label: "Downloads", icon: "download" },
  { id: "stats", label: "Statistics", icon: "chart" },
  { id: "settings", label: "Settings", icon: "settings" },
];

const connectionStatus = computed(() => {
  if (!props.networkStatus)
    return { text: "Connecting...", class: "connecting" };
  if (props.networkStatus.connectedPeers > 0)
    return { text: "Connected", class: "connected" };
  return { text: "Offline", class: "disconnected" };
});

const peerCount = computed(() => props.networkStatus?.connectedPeers || 0);
const relayConnected = computed(() => props.networkStatus?.relayConnected || false);
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <div class="logo-wrapper">
        <img src="../assets/images/logo-universal.png" alt="Torrentium" class="logo-img" onerror="this.style.display='none'" />
        <span class="logo-fallback">🌊</span>
      </div>
      <div class="logo-text">
        <h1 class="logo-title">Torrentium</h1>
        <span class="logo-subtitle">P2P File Sharing</span>
      </div>
    </div>

    <nav class="nav-menu">
      <button
        v-for="item in menuItems"
        :key="item.id"
        :class="['nav-item', { active: currentView === item.id }]"
        @click="emit('navigate', item.id)"
      >
        <span class="nav-icon">
          <svg v-if="item.icon === 'dashboard'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7" rx="1"/>
            <rect x="14" y="3" width="7" height="7" rx="1"/>
            <rect x="3" y="14" width="7" height="7" rx="1"/>
            <rect x="14" y="14" width="7" height="7" rx="1"/>
          </svg>
          <svg v-else-if="item.icon === 'folder'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          <svg v-else-if="item.icon === 'download'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/>
            <line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          <svg v-else-if="item.icon === 'chart'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="20" x2="18" y2="10"/>
            <line x1="12" y1="20" x2="12" y2="4"/>
            <line x1="6" y1="20" x2="6" y2="14"/>
          </svg>
          <svg v-else-if="item.icon === 'settings'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
        </span>
        <span class="nav-label">{{ item.label }}</span>
        <span v-if="item.id === 'downloads'" class="nav-badge">{{ networkStatus?.activeDownloads || 0 }}</span>
      </button>
    </nav>

    <div class="sidebar-footer">
      <!-- Connection Status -->
      <div class="connection-card">
        <div class="connection-header">
          <div :class="['status-dot', connectionStatus.class]"></div>
          <span class="connection-label">{{ connectionStatus.text }}</span>
        </div>
        <div class="connection-stats">
          <div class="stat-item">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="stat-icon">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
              <circle cx="9" cy="7" r="4"/>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
            <span>{{ peerCount }} peers</span>
          </div>
          <div class="stat-item">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="stat-icon" :class="{ active: relayConnected }">
              <path d="M5 12.55a11 11 0 0 1 14.08 0"/>
              <path d="M1.42 9a16 16 0 0 1 21.16 0"/>
              <path d="M8.53 16.11a6 6 0 0 1 6.95 0"/>
              <circle cx="12" cy="20" r="1"/>
            </svg>
            <span>{{ relayConnected ? 'Relay OK' : 'No Relay' }}</span>
          </div>
        </div>
      </div>

      <!-- Theme Toggle -->
      <button class="theme-toggle" @click="emit('toggleTheme')">
        <svg v-if="theme === 'dark'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="5"/>
          <line x1="12" y1="1" x2="12" y2="3"/>
          <line x1="12" y1="21" x2="12" y2="23"/>
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
          <line x1="1" y1="12" x2="3" y2="12"/>
          <line x1="21" y1="12" x2="23" y2="12"/>
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
        </svg>
        <span>{{ theme === 'dark' ? 'Light Mode' : 'Dark Mode' }}</span>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 260px;
  background: var(--bg-secondary);
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);
}

.sidebar-header {
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 14px;
  border-bottom: 1px solid var(--border);
}

.logo-wrapper {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.logo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-fallback {
  font-size: 1.8rem;
}

.logo-wrapper .logo-img:not([src=""]) + .logo-fallback {
  display: none;
}

.logo-text {
  overflow: hidden;
}

.logo-title {
  font-size: 1.2rem;
  font-weight: 700;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  line-height: 1.2;
}

.logo-subtitle {
  font-size: 0.7rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.nav-menu {
  flex: 1;
  padding: 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.9rem;
  cursor: pointer;
  border-radius: var(--radius-md);
  transition: all var(--transition-normal);
  text-align: left;
  position: relative;
}

.nav-item:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.active {
  background: var(--gradient-primary);
  color: var(--bg-primary);
  box-shadow: var(--shadow-glow);
}

.nav-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.nav-icon svg {
  width: 100%;
  height: 100%;
}

.nav-label {
  flex: 1;
}

.nav-badge {
  background: var(--accent);
  color: var(--bg-primary);
  font-size: 0.7rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  min-width: 20px;
  text-align: center;
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.connection-card {
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  padding: 14px;
}

.connection-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.connected {
  background: var(--success);
  box-shadow: 0 0 10px var(--success-glow);
  animation: pulse 2s ease-in-out infinite;
}

.status-dot.disconnected {
  background: var(--error);
}

.status-dot.connecting {
  background: var(--warning);
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 10px var(--success-glow); }
  50% { opacity: 0.7; box-shadow: 0 0 20px var(--success-glow); }
}

.connection-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
}

.connection-stats {
  display: flex;
  gap: 16px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.stat-icon {
  width: 14px;
  height: 14px;
}

.stat-icon.active {
  color: var(--success);
}

.theme-toggle {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all var(--transition-normal);
}

.theme-toggle:hover {
  background: var(--bg-hover);
  border-color: var(--accent);
  color: var(--text-primary);
}

.theme-toggle svg {
  width: 18px;
  height: 18px;
}

/* Responsive Styles */
@media (max-width: 768px) {
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 1000;
    transform: translateX(-100%);
    transition: transform var(--transition-normal);
    box-shadow: var(--shadow-lg);
  }

  .sidebar.sidebar-open {
    transform: translateX(0);
  }

  .sidebar-header {
    padding-top: 24px;
  }

  .nav-menu {
    padding: 12px 8px;
  }

  .nav-item {
    padding: 14px 16px;
  }
}

@media (max-width: 480px) {
  .sidebar {
    width: 280px;
  }

  .logo-wrapper {
    width: 40px;
    height: 40px;
  }

  .logo-title {
    font-size: 1.1rem;
  }

  .connection-stats {
    flex-direction: column;
    gap: 8px;
  }
}
</style>
