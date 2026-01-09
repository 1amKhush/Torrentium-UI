<script setup>
import { ref, onMounted, computed } from "vue";
import Sidebar from "./components/Sidebar.vue";
import Dashboard from "./components/Dashboard.vue";
import Files from "./components/Files.vue";
import Downloads from "./components/Downloads.vue";
import Stats from "./components/Stats.vue";
import Settings from "./components/Settings.vue";
import { EventsOn } from "../wailsjs/runtime/runtime";

const currentView = ref("dashboard");
const isReady = ref(false);
const statusMessage = ref("Initializing...");
const networkStatus = ref(null);
const errorMessage = ref(null);

const views = {
  dashboard: Dashboard,
  files: Files,
  downloads: Downloads,
  stats: Stats,
  settings: Settings,
};

const currentComponent = computed(() => views[currentView.value]);

onMounted(() => {
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
  });
});
</script>

<template>
  <div class="app-container">
    <!-- Loading Screen -->
    <div v-if="!isReady" class="loading-screen">
      <div class="loading-content">
        <div class="logo-animation">
          <svg viewBox="0 0 100 100" class="spinner">
            <circle
              cx="50"
              cy="50"
              r="40"
              stroke="#3b82f6"
              stroke-width="4"
              fill="none"
              stroke-dasharray="251.2"
              stroke-dashoffset="62.8"
            />
          </svg>
        </div>
        <h1 class="app-title">Torrentium</h1>
        <p class="status-text">{{ statusMessage }}</p>
      </div>
    </div>

    <!-- Error Screen -->
    <div v-else-if="errorMessage" class="error-screen">
      <div class="error-content">
        <div class="error-icon">⚠️</div>
        <h1>Initialization Error</h1>
        <p class="error-message">{{ errorMessage }}</p>
        <p class="error-hint">
          Please check your network connection and try again.
        </p>
      </div>
    </div>

    <!-- Main App -->
    <template v-else>
      <Sidebar
        :currentView="currentView"
        :networkStatus="networkStatus"
        @navigate="currentView = $event"
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

:root {
  --bg-primary: #121212;
  --bg-secondary: #1e1e1e;
  --bg-tertiary: #2d2d2d;
  --text-primary: #ffffff;
  --text-secondary: #b3b3b3;
  --accent: #3b82f6;
  --accent-hover: #2563eb;
  --success: #22c55e;
  --warning: #f59e0b;
  --error: #ef4444;
  --border: #404040;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen,
    Ubuntu, sans-serif;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  overflow: hidden;
}

.app-container {
  display: flex;
  height: 100vh;
  width: 100vw;
}

.loading-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, var(--bg-primary) 0%, #1a1a2e 100%);
}

.loading-content {
  text-align: center;
}

.logo-animation {
  width: 80px;
  height: 80px;
  margin: 0 auto 20px;
}

.spinner {
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.app-title {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 10px;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.status-text {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.error-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, var(--bg-primary) 0%, #1a1a2e 100%);
}

.error-content {
  text-align: center;
  max-width: 500px;
  padding: 40px;
}

.error-icon {
  font-size: 4rem;
  margin-bottom: 20px;
}

.error-content h1 {
  font-size: 1.5rem;
  margin-bottom: 16px;
  color: var(--error);
}

.error-message {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: 8px;
  font-family: monospace;
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 16px;
  word-break: break-word;
}

.error-hint {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.main-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  background-color: var(--bg-primary);
}
</style>
