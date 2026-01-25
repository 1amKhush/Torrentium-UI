<script setup>
import { ref, onMounted, computed, provide, onUnmounted } from "vue";
import Sidebar from "./components/Sidebar.vue";
import Dashboard from "./components/Dashboard.vue";
import Files from "./components/Files.vue";
import Downloads from "./components/Downloads.vue";
import Stats from "./components/Stats.vue";
import Settings from "./components/Settings.vue";
import ToastNotification from "./components/ToastNotification.vue";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";

const currentView = ref("dashboard");
const isReady = ref(false);
const statusMessage = ref("Initializing...");
const networkStatus = ref(null);
const errorMessage = ref(null);
const theme = ref(localStorage.getItem('theme') || 'dark');
const toasts = ref([]);
const sidebarOpen = ref(false);

// Provide theme globally
provide('theme', theme);
provide('addToast', addToast);

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value;
}

function closeSidebar() {
  sidebarOpen.value = false;
}

const views = {
  dashboard: Dashboard,
  files: Files,
  downloads: Downloads,
  stats: Stats,
  settings: Settings,
};

const currentComponent = computed(() => views[currentView.value]);

function addToast(message, type = 'success', duration = 3000) {
  const id = Date.now();
  toasts.value.push({ id, message, type });
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id);
  }, duration);
}

function removeToast(id) {
  toasts.value = toasts.value.filter(t => t.id !== id);
}

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark';
  localStorage.setItem('theme', theme.value);
  document.documentElement.setAttribute('data-theme', theme.value);
}

provide('toggleTheme', toggleTheme);

// Navigate to a specific view (used by system tray menu)
function navigateTo(view) {
  if (views[view]) {
    currentView.value = view;
  }
}

// Expose navigation function for child components
provide('navigateTo', navigateTo);

onMounted(() => {
  // Apply saved theme
  document.documentElement.setAttribute('data-theme', theme.value);
  
  EventsOn("ready", (ready) => {
    isReady.value = ready;
  });

  EventsOn("status", (msg) => {
    statusMessage.value = msg;
  });

  EventsOn("networkUpdate", (status) => {
    networkStatus.value = status;
  });

  EventsOn("error", (err) => {
    console.error("Torrentium error:", err);
    errorMessage.value = err;
  });

  EventsOn("warning", (warn) => {
    console.warn("Torrentium warning:", warn);
    addToast(warn, 'warning');
  });

  EventsOn("downloadComplete", (info) => {
    addToast(`Download complete: ${info.filename || 'File'}`, 'success');
  });

  EventsOn("fileAdded", (info) => {
    addToast(`File shared: ${info.filename || 'File'}`, 'success');
  });

  // Handle navigation from system tray
  EventsOn("navigate", (view) => {
    navigateTo(view);
  });
});

onUnmounted(() => {
  EventsOff("ready");
  EventsOff("status");
  EventsOff("networkUpdate");
  EventsOff("error");
  EventsOff("warning");
  EventsOff("downloadComplete");
  EventsOff("fileAdded");
  EventsOff("navigate");
});
</script>

<template>
  <div class="app-container" :data-theme="theme">
    <!-- Toast Notifications -->
    <div class="toast-container">
      <TransitionGroup name="toast">
        <ToastNotification
          v-for="toast in toasts"
          :key="toast.id"
          :message="toast.message"
          :type="toast.type"
          @close="removeToast(toast.id)"
        />
      </TransitionGroup>
    </div>

    <!-- Loading Screen -->
    <div v-if="!isReady" class="loading-screen">
      <div class="loading-content">
        <div class="logo-container">
          <img src="./assets/images/logo-universal.png" alt="Torrentium" class="loading-logo" onerror="this.style.display='none'" />
          <span class="logo-fallback">🌊</span>
          <div class="logo-glow"></div>
        </div>
        <h1 class="app-title">Torrentium</h1>
        <p class="app-tagline">Decentralized P2P File Sharing</p>
        <div class="loading-bar">
          <div class="loading-progress"></div>
        </div>
        <p class="status-text">{{ statusMessage }}</p>
      </div>
      <div class="loading-particles">
        <div class="particle" v-for="n in 20" :key="n"></div>
      </div>
    </div>

    <!-- Error Screen -->
    <div v-else-if="errorMessage" class="error-screen">
      <div class="error-content">
        <div class="error-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <h1>Connection Error</h1>
        <p class="error-message">{{ errorMessage }}</p>
        <p class="error-hint">
          Please check your network connection and try again.
        </p>
        <button class="btn btn-primary" @click="location.reload()">
          🔄 Retry Connection
        </button>
      </div>
    </div>

    <!-- Main App -->
    <template v-else>
      <!-- Mobile Menu Toggle -->
      <button class="mobile-menu-toggle" @click="toggleSidebar" aria-label="Toggle menu">
        <svg v-if="!sidebarOpen" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="3" y1="6" x2="21" y2="6"/>
          <line x1="3" y1="12" x2="21" y2="12"/>
          <line x1="3" y1="18" x2="21" y2="18"/>
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>

      <!-- Sidebar Overlay for Mobile -->
      <div 
        :class="['sidebar-overlay', { visible: sidebarOpen }]" 
        @click="closeSidebar"
      ></div>

      <Sidebar
        :currentView="currentView"
        :networkStatus="networkStatus"
        :theme="theme"
        :class="{ 'sidebar-open': sidebarOpen }"
        @navigate="(view) => { currentView = view; closeSidebar(); }"
        @toggleTheme="toggleTheme"
      />
      <main class="main-content">
        <component :is="currentComponent" :networkStatus="networkStatus" />
      </main>
    </template>
  </div>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.app-container {
  display: flex;
  height: 100vh;
  width: 100vw;
  background: var(--bg-primary);
}

/* Toast Transitions */
.toast-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.toast-enter-active {
  animation: slideInRight 0.3s ease;
}

.toast-leave-active {
  animation: slideOutRight 0.3s ease;
}

@keyframes slideInRight {
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}

@keyframes slideOutRight {
  from { transform: translateX(0); opacity: 1; }
  to { transform: translateX(100%); opacity: 0; }
}

/* Loading Screen */
.loading-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: var(--bg-primary);
  position: relative;
  overflow: hidden;
}

.loading-screen::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 600px;
  height: 600px;
  transform: translate(-50%, -50%);
  background: radial-gradient(circle, var(--accent-glow) 0%, transparent 70%);
  animation: pulse 3s ease-in-out infinite;
}

.loading-content {
  text-align: center;
  z-index: 10;
}

.logo-container {
  position: relative;
  width: 120px;
  height: 120px;
  margin: 0 auto 24px;
}

.loading-logo {
  width: 100%;
  height: 100%;
  object-fit: contain;
  animation: float 3s ease-in-out infinite;
  filter: drop-shadow(0 0 30px var(--accent-glow));
}

.logo-fallback {
  font-size: 5rem;
  animation: float 3s ease-in-out infinite;
  display: block;
}

.logo-container .loading-logo:not([style*="display: none"]) + .logo-fallback {
  display: none;
}

.logo-glow {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 150%;
  height: 150%;
  transform: translate(-50%, -50%);
  background: radial-gradient(circle, var(--accent-glow) 0%, transparent 70%);
  animation: glow 2s ease-in-out infinite;
  z-index: -1;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

@keyframes glow {
  0%, 100% { opacity: 0.5; transform: translate(-50%, -50%) scale(1); }
  50% { opacity: 0.8; transform: translate(-50%, -50%) scale(1.1); }
}

.app-title {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 8px;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  letter-spacing: -0.5px;
}

.app-tagline {
  color: var(--text-secondary);
  font-size: 1rem;
  margin-bottom: 32px;
}

.loading-bar {
  width: 200px;
  height: 4px;
  background: var(--bg-tertiary);
  border-radius: 2px;
  margin: 0 auto 16px;
  overflow: hidden;
}

.loading-progress {
  height: 100%;
  width: 30%;
  background: var(--gradient-primary);
  border-radius: 2px;
  animation: loadingBar 1.5s ease-in-out infinite;
}

@keyframes loadingBar {
  0% { transform: translateX(-100%); }
  50% { transform: translateX(200%); }
  100% { transform: translateX(-100%); }
}

.status-text {
  color: var(--text-muted);
  font-size: 0.85rem;
}

/* Particles */
.loading-particles {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.particle {
  position: absolute;
  width: 4px;
  height: 4px;
  background: var(--accent);
  border-radius: 50%;
  opacity: 0;
  animation: particleFloat 4s ease-in-out infinite;
}

.particle:nth-child(1) { left: 10%; animation-delay: 0s; }
.particle:nth-child(2) { left: 20%; animation-delay: 0.2s; }
.particle:nth-child(3) { left: 30%; animation-delay: 0.4s; }
.particle:nth-child(4) { left: 40%; animation-delay: 0.6s; }
.particle:nth-child(5) { left: 50%; animation-delay: 0.8s; }
.particle:nth-child(6) { left: 60%; animation-delay: 1s; }
.particle:nth-child(7) { left: 70%; animation-delay: 1.2s; }
.particle:nth-child(8) { left: 80%; animation-delay: 1.4s; }
.particle:nth-child(9) { left: 90%; animation-delay: 1.6s; }
.particle:nth-child(10) { left: 15%; animation-delay: 1.8s; }
.particle:nth-child(11) { left: 25%; animation-delay: 2s; }
.particle:nth-child(12) { left: 35%; animation-delay: 2.2s; }
.particle:nth-child(13) { left: 45%; animation-delay: 2.4s; }
.particle:nth-child(14) { left: 55%; animation-delay: 2.6s; }
.particle:nth-child(15) { left: 65%; animation-delay: 2.8s; }
.particle:nth-child(16) { left: 75%; animation-delay: 3s; }
.particle:nth-child(17) { left: 85%; animation-delay: 3.2s; }
.particle:nth-child(18) { left: 95%; animation-delay: 3.4s; }
.particle:nth-child(19) { left: 5%; animation-delay: 3.6s; }
.particle:nth-child(20) { left: 50%; animation-delay: 3.8s; }

@keyframes particleFloat {
  0% { bottom: -10px; opacity: 0; }
  10% { opacity: 0.6; }
  90% { opacity: 0.6; }
  100% { bottom: 100%; opacity: 0; }
}

/* Error Screen */
.error-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: var(--bg-primary);
}

.error-content {
  text-align: center;
  max-width: 450px;
  padding: 40px;
}

.error-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto 24px;
  color: var(--error);
}

.error-icon svg {
  width: 100%;
  height: 100%;
}

.error-content h1 {
  font-size: 1.5rem;
  margin-bottom: 16px;
  color: var(--text-primary);
}

.error-message {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: var(--radius-md);
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 0.8rem;
  color: var(--error);
  margin-bottom: 16px;
  word-break: break-word;
  border: 1px solid var(--error-glow);
}

.error-hint {
  color: var(--text-secondary);
  font-size: 0.9rem;
  margin-bottom: 24px;
}

/* Main Content */
.main-content {
  flex: 1;
  padding: 24px 32px;
  overflow-y: auto;
  background: var(--bg-primary);
  animation: fadeIn 0.3s ease;
}

/* Mobile Menu Toggle */
.mobile-menu-toggle {
  display: none;
  position: fixed;
  top: 16px;
  left: 16px;
  z-index: 1001;
  width: 44px;
  height: 44px;
  border: none;
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  color: var(--text-primary);
  cursor: pointer;
  box-shadow: var(--shadow-md);
  transition: all var(--transition-fast);
}

.mobile-menu-toggle:hover {
  background: var(--accent);
  color: var(--bg-primary);
}

.mobile-menu-toggle svg {
  width: 24px;
  height: 24px;
}

/* Sidebar Overlay */
.sidebar-overlay {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 999;
  opacity: 0;
  transition: opacity var(--transition-normal);
}

.sidebar-overlay.visible {
  opacity: 1;
}

/* Responsive Breakpoints */
@media (max-width: 1024px) {
  .main-content {
    padding: 20px 24px;
  }
}

@media (max-width: 768px) {
  .mobile-menu-toggle {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .sidebar-overlay {
    display: block;
  }

  .main-content {
    padding: 70px 16px 16px 16px;
  }

  .toast-container {
    left: 16px;
    right: 16px;
    top: 16px;
  }
}

@media (max-width: 480px) {
  .main-content {
    padding: 70px 12px 12px 12px;
  }

  .app-title {
    font-size: 2rem;
  }

  .loading-content {
    padding: 0 20px;
  }

  .logo-container {
    width: 80px;
    height: 80px;
  }

  .logo-fallback {
    font-size: 3rem;
  }
}
</style>
