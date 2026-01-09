<script setup>
import { ref, onMounted, computed } from "vue";
import {
  GetConfig,
  SetMaxUploadRate,
  SetDownloadDirectory,
  SelectDownloadDirectory,
  GetPeerID,
  RefreshDHT,
  CopyToClipboard,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const config = ref(null);
const peerId = ref("");
const loading = ref(true);
const notification = ref(null);
const uploadRateInput = ref("");
const isUnlimited = ref(true);

const loadConfig = async () => {
  try {
    const [configData, peerIdData] = await Promise.all([
      GetConfig(),
      GetPeerID(),
    ]);
    config.value = configData;
    peerId.value = peerIdData;

    if (configData.maxUploadRate === 0) {
      isUnlimited.value = true;
      uploadRateInput.value = "";
    } else {
      isUnlimited.value = false;
      uploadRateInput.value = Math.floor(
        configData.maxUploadRate / 1024
      ).toString(); // Convert to KB/s
    }
  } catch (err) {
    showNotification("Failed to load config: " + err, "error");
  } finally {
    loading.value = false;
  }
};

const showNotification = (message, type = "success") => {
  notification.value = { message, type };
  setTimeout(() => (notification.value = null), 3000);
};

const changeDownloadDir = async () => {
  try {
    const dir = await SelectDownloadDirectory();
    if (dir) {
      await SetDownloadDirectory(dir);
      showNotification("Download directory updated!");
      await loadConfig();
    }
  } catch (err) {
    showNotification("Failed to set directory: " + err, "error");
  }
};

const updateUploadRate = async () => {
  try {
    let rate = 0;
    if (!isUnlimited.value && uploadRateInput.value) {
      rate = parseInt(uploadRateInput.value) * 1024; // Convert KB/s to B/s
      if (isNaN(rate) || rate < 0) {
        showNotification("Please enter a valid number", "error");
        return;
      }
    }
    await SetMaxUploadRate(rate);
    showNotification(
      rate === 0
        ? "Upload rate set to unlimited"
        : `Upload rate set to ${uploadRateInput.value} KB/s`
    );
    await loadConfig();
  } catch (err) {
    showNotification("Failed to update rate: " + err, "error");
  }
};

const toggleUnlimited = () => {
  isUnlimited.value = !isUnlimited.value;
  if (isUnlimited.value) {
    updateUploadRate();
  }
};

const copyPeerId = async () => {
  try {
    await CopyToClipboard(peerId.value);
    showNotification("Peer ID copied!");
  } catch (err) {
    showNotification("Failed to copy", "error");
  }
};

const refreshDHT = async () => {
  try {
    await RefreshDHT();
    showNotification("DHT refresh initiated!");
  } catch (err) {
    showNotification("DHT refresh failed: " + err, "error");
  }
};

onMounted(() => {
  loadConfig();
  EventsOn("configUpdated", (newConfig) => {
    config.value = newConfig;
  });
});
</script>

<template>
  <div class="settings-page">
    <!-- Notification -->
    <Transition name="slide">
      <div v-if="notification" :class="['notification', notification.type]">
        {{ notification.message }}
      </div>
    </Transition>

    <header class="page-header">
      <h1>Settings</h1>
      <p class="subtitle">Configure your Torrentium client</p>
    </header>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading settings...</p>
    </div>

    <template v-else-if="config">
      <!-- Identity Section -->
      <section class="settings-section">
        <h2>🪪 Identity</h2>
        <div class="setting-card">
          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Your Peer ID</span>
              <span class="setting-description"
                >Your unique identifier on the network</span
              >
            </div>
            <div class="setting-value">
              <code class="peer-id">{{ peerId.slice(0, 20) }}...</code>
              <button
                class="btn-icon"
                @click="copyPeerId"
                title="Copy full Peer ID"
              >
                📋
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- Storage Section -->
      <section class="settings-section">
        <h2>💾 Storage</h2>
        <div class="setting-card">
          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Download Directory</span>
              <span class="setting-description"
                >Where downloaded files will be saved</span
              >
            </div>
            <div class="setting-value">
              <code>{{ config.downloadDir }}</code>
              <button class="btn btn-small" @click="changeDownloadDir">
                Change
              </button>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Database Path</span>
              <span class="setting-description"
                >Local database for file tracking</span
              >
            </div>
            <div class="setting-value">
              <code>{{ config.databasePath }}</code>
            </div>
          </div>
        </div>
      </section>

      <!-- Network Section -->
      <section class="settings-section">
        <h2>🌐 Network</h2>
        <div class="setting-card">
          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Max Upload Rate</span>
              <span class="setting-description"
                >Limit your upload bandwidth (current:
                {{ config.maxUploadRateHuman }})</span
              >
            </div>
            <div class="setting-controls">
              <label class="checkbox-label">
                <input
                  type="checkbox"
                  :checked="isUnlimited"
                  @change="toggleUnlimited"
                />
                <span>Unlimited</span>
              </label>
              <div class="rate-input" v-if="!isUnlimited">
                <input
                  type="number"
                  v-model="uploadRateInput"
                  placeholder="Rate"
                  min="0"
                  class="number-input"
                />
                <span class="unit">KB/s</span>
                <button class="btn btn-small" @click="updateUploadRate">
                  Apply
                </button>
              </div>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">DHT Refresh</span>
              <span class="setting-description"
                >Manually refresh the distributed hash table</span
              >
            </div>
            <button class="btn btn-secondary" @click="refreshDHT">
              🔄 Refresh DHT
            </button>
          </div>
        </div>
      </section>

      <!-- About Section -->
      <section class="settings-section">
        <h2>ℹ️ About</h2>
        <div class="setting-card">
          <div class="about-content">
            <div class="app-info">
              <h3>🌊 Torrentium</h3>
              <p>Version 1.0.0</p>
              <p class="description">
                A decentralized peer-to-peer file sharing application built with
                libp2p and WebRTC.
              </p>
            </div>
            <div class="features">
              <h4>Features:</h4>
              <ul>
                <li>✅ Decentralized file sharing</li>
                <li>✅ DHT-based peer discovery</li>
                <li>✅ WebRTC data channels</li>
                <li>✅ Rarest-first piece selection</li>
                <li>✅ Upload rate limiting</li>
                <li>✅ Multi-file support</li>
              </ul>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 800px;
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

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
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

.settings-section {
  margin-bottom: 32px;
}

.settings-section h2 {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 16px;
}

.setting-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  border: 1px solid var(--border);
  overflow: hidden;
}

.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--border);
  gap: 20px;
}

.setting-row:last-child {
  border-bottom: none;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.setting-label {
  font-weight: 500;
}

.setting-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.setting-value {
  display: flex;
  align-items: center;
  gap: 12px;
}

.setting-value code {
  background: var(--bg-tertiary);
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 0.85rem;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.peer-id {
  font-family: monospace;
}

.setting-controls {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.checkbox-label input {
  width: 18px;
  height: 18px;
  accent-color: var(--accent);
}

.rate-input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.number-input {
  width: 80px;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: 0.9rem;
}

.number-input:focus {
  outline: none;
  border-color: var(--accent);
}

.unit {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.btn {
  padding: 10px 16px;
  border-radius: 6px;
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s ease;
}

.btn-small {
  padding: 8px 12px;
  font-size: 0.85rem;
}

.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border);
}

.btn-secondary:hover {
  background: var(--border);
}

.btn-icon {
  width: 36px;
  height: 36px;
  border: none;
  background: var(--bg-tertiary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-icon:hover {
  background: var(--accent);
}

.about-content {
  padding: 24px;
}

.app-info {
  margin-bottom: 24px;
}

.app-info h3 {
  font-size: 1.5rem;
  margin-bottom: 8px;
}

.app-info p {
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.app-info .description {
  margin-top: 12px;
  line-height: 1.5;
}

.features h4 {
  margin-bottom: 12px;
  font-size: 0.95rem;
}

.features ul {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.features li {
  font-size: 0.9rem;
  color: var(--text-secondary);
}
</style>
