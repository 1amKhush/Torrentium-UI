<script setup>
import { computed } from "vue";

const props = defineProps({
  currentView: String,
  networkStatus: Object,
});

const emit = defineEmits(["navigate"]);

const menuItems = [
  { id: "dashboard", label: "Dashboard", icon: "📊" },
  { id: "files", label: "My Files", icon: "📁" },
  { id: "downloads", label: "Downloads", icon: "⬇️" },
  { id: "stats", label: "Statistics", icon: "📈" },
  { id: "settings", label: "Settings", icon: "⚙️" },
];

const connectionStatus = computed(() => {
  if (!props.networkStatus)
    return { text: "Connecting...", class: "connecting" };
  if (props.networkStatus.connectedPeers > 0)
    return { text: "Connected", class: "connected" };
  return { text: "Disconnected", class: "disconnected" };
});

const peerCount = computed(() => props.networkStatus?.connectedPeers || 0);
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <h1 class="logo">🌊 Torrentium</h1>
    </div>

    <nav class="nav-menu">
      <button
        v-for="item in menuItems"
        :key="item.id"
        :class="['nav-item', { active: currentView === item.id }]"
        @click="emit('navigate', item.id)"
      >
        <span class="nav-icon">{{ item.icon }}</span>
        <span class="nav-label">{{ item.label }}</span>
      </button>
    </nav>

    <div class="sidebar-footer">
      <div class="connection-status">
        <div :class="['status-indicator', connectionStatus.class]"></div>
        <span class="status-text">{{ connectionStatus.text }}</span>
      </div>
      <div class="peer-count" v-if="networkStatus">
        <span class="peer-icon">👥</span>
        <span>{{ peerCount }} peers</span>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background-color: var(--bg-secondary);
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid var(--border);
}

.logo {
  font-size: 1.3rem;
  font-weight: 700;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
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
  font-size: 0.95rem;
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.2s ease;
  text-align: left;
}

.nav-item:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.active {
  background-color: var(--accent);
  color: white;
}

.nav-icon {
  font-size: 1.1rem;
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid var(--border);
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-indicator.connected {
  background-color: var(--success);
  box-shadow: 0 0 8px var(--success);
}

.status-indicator.disconnected {
  background-color: var(--error);
}

.status-indicator.connecting {
  background-color: var(--warning);
  animation: pulse 1s infinite;
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

.status-text {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.peer-count {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  color: var(--text-secondary);
}
</style>
