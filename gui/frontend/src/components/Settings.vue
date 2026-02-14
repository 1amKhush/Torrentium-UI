<script setup>
import { ref, onMounted, inject } from "vue";
import {
  GetConfig,
  SetMaxUploadRate,
  SetDownloadDirectory,
  SelectDownloadDirectory,
  GetPeerID,
  RefreshDHT,
  CopyToClipboard,
  SetMaxParallelDownloads,
  SetAdaptiveParallelDownloads,
  SetEndgameMode,
  SetMaxDownloadRate,
  GetWebShareConfig,
  SetWebShareConfig,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const addToast = inject("addToast");
const theme = inject("theme");

const config = ref(null);
const peerId = ref("");
const loading = ref(true);
const uploadRateInput = ref("");
const downloadRateInput = ref("");
const maxParallelInput = ref(5);
const isUnlimited = ref(true);
const isDownloadUnlimited = ref(true);
const adaptiveEnabled = ref(false);
const endgameEnabled = ref(false);
const isRefreshing = ref(false);

// Web Share settings
const webShareConfig = ref({
  portalUrl: '',
  apiKey: '',
  defaultVisibility: 'public',
  defaultExpiration: 0
});

const loadConfig = async () => {
  try {
    const [configData, peerIdData, webShareData] = await Promise.all([
      GetConfig(),
      GetPeerID(),
      GetWebShareConfig(),
    ]);
    config.value = configData;
    peerId.value = peerIdData;
    webShareConfig.value = webShareData;

    if (configData.maxUploadRate === 0) {
      isUnlimited.value = true;
      uploadRateInput.value = "";
    } else {
      isUnlimited.value = false;
      uploadRateInput.value = Math.floor(
        configData.maxUploadRate / 1024,
      ).toString();
    }

    // Load download settings
    maxParallelInput.value = configData.maxParallelDownloads || 5;
    adaptiveEnabled.value = configData.adaptiveParallelDownloads || false;
    endgameEnabled.value = configData.enableEndgameMode || false;

    if (configData.maxDownloadRate === 0) {
      isDownloadUnlimited.value = true;
      downloadRateInput.value = "";
    } else {
      isDownloadUnlimited.value = false;
      downloadRateInput.value = Math.floor(
        configData.maxDownloadRate / 1024,
      ).toString();
    }
  } catch (err) {
    addToast("Failed to load config: " + err, "error");
  } finally {
    loading.value = false;
  }
};

const changeDownloadDir = async () => {
  try {
    const dir = await SelectDownloadDirectory();
    if (dir) {
      await SetDownloadDirectory(dir);
      addToast("Download directory updated!", "success");
      await loadConfig();
    }
  } catch (err) {
    addToast("Failed to set directory: " + err, "error");
  }
};

const updateUploadRate = async () => {
  try {
    let rate = 0;
    if (!isUnlimited.value && uploadRateInput.value) {
      rate = parseInt(uploadRateInput.value) * 1024;
      if (isNaN(rate) || rate < 0) {
        addToast("Please enter a valid number", "warning");
        return;
      }
    }
    await SetMaxUploadRate(rate);
    addToast(
      rate === 0
        ? "Upload rate set to unlimited"
        : `Upload rate set to ${uploadRateInput.value} KB/s`,
      "success",
    );
    await loadConfig();
  } catch (err) {
    addToast("Failed to update rate: " + err, "error");
  }
};

const updateDownloadRate = async () => {
  try {
    let rate = 0;
    if (!isDownloadUnlimited.value && downloadRateInput.value) {
      rate = parseInt(downloadRateInput.value) * 1024;
      if (isNaN(rate) || rate < 0) {
        addToast("Please enter a valid number", "warning");
        return;
      }
    }
    await SetMaxDownloadRate(rate);
    addToast(
      rate === 0
        ? "Download rate set to unlimited"
        : `Download rate set to ${downloadRateInput.value} KB/s`,
      "success",
    );
    await loadConfig();
  } catch (err) {
    addToast("Failed to update download rate: " + err, "error");
  }
};

const updateMaxParallel = async () => {
  try {
    const value = parseInt(maxParallelInput.value);
    if (isNaN(value) || value < 1 || value > 20) {
      addToast("Please enter a number between 1 and 20", "warning");
      return;
    }
    await SetMaxParallelDownloads(value);
    addToast(`Max parallel downloads set to ${value}`, "success");
    await loadConfig();
  } catch (err) {
    addToast("Failed to update parallel downloads: " + err, "error");
  }
};

const toggleAdaptive = async () => {
  try {
    const newValue = !adaptiveEnabled.value;
    await SetAdaptiveParallelDownloads(newValue);
    adaptiveEnabled.value = newValue;
    addToast(
      newValue ? "Adaptive downloads enabled" : "Adaptive downloads disabled",
      "success",
    );
  } catch (err) {
    addToast("Failed to toggle adaptive downloads: " + err, "error");
  }
};

const toggleEndgame = async () => {
  try {
    const newValue = !endgameEnabled.value;
    await SetEndgameMode(newValue);
    endgameEnabled.value = newValue;
    addToast(
      newValue ? "Endgame mode enabled" : "Endgame mode disabled",
      "success",
    );
  } catch (err) {
    addToast("Failed to toggle endgame mode: " + err, "error");
  }
};

const toggleUnlimited = () => {
  isUnlimited.value = !isUnlimited.value;
  if (isUnlimited.value) {
    updateUploadRate();
  }
};

const toggleDownloadUnlimited = () => {
  isDownloadUnlimited.value = !isDownloadUnlimited.value;
  if (isDownloadUnlimited.value) {
    updateDownloadRate();
  }
};

const copyPeerId = async () => {
  try {
    await CopyToClipboard(peerId.value);
    addToast("Peer ID copied!", "success");
  } catch (err) {
    addToast("Failed to copy", "error");
  }
};

const refreshDHT = async () => {
  try {
    isRefreshing.value = true;
    await RefreshDHT();
    addToast("DHT refresh initiated!", "success");
  } catch (err) {
    addToast("DHT refresh failed: " + err, "error");
  } finally {
    setTimeout(() => (isRefreshing.value = false), 1000);
  }
};

const toggleTheme = () => {
  const newTheme = theme.value === "dark" ? "light" : "dark";
  theme.value = newTheme;
  document.documentElement.setAttribute("data-theme", newTheme);
  localStorage.setItem("theme", newTheme);
};

const updateWebShareConfig = async () => {
  try {
    await SetWebShareConfig(
      webShareConfig.value.portalUrl,
      webShareConfig.value.apiKey,
      webShareConfig.value.defaultVisibility,
      webShareConfig.value.defaultExpiration
    );
    addToast("Web Share settings updated!", "success");
  } catch (err) {
    addToast("Failed to update Web Share settings: " + err, "error");
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
    <header class="page-header">
      <div>
        <h1>Settings</h1>
        <p class="subtitle">Configure your Torrentium client</p>
      </div>
    </header>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading settings...</p>
    </div>

    <template v-else-if="config">
      <!-- Appearance Section -->
      <section class="settings-section">
        <div class="section-header">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="5" />
            <line x1="12" y1="1" x2="12" y2="3" />
            <line x1="12" y1="21" x2="12" y2="23" />
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
            <line x1="1" y1="12" x2="3" y2="12" />
            <line x1="21" y1="12" x2="23" y2="12" />
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
          </svg>
          <h2>Appearance</h2>
        </div>
        <div class="setting-card">
          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Theme</span>
              <span class="setting-description"
                >Choose between dark and light mode</span
              >
            </div>
            <button class="theme-toggle-btn" @click="toggleTheme">
              <svg
                v-if="theme === 'dark'"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <circle cx="12" cy="12" r="5" />
                <line x1="12" y1="1" x2="12" y2="3" />
                <line x1="12" y1="21" x2="12" y2="23" />
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                <line x1="1" y1="12" x2="3" y2="12" />
                <line x1="21" y1="12" x2="23" y2="12" />
              </svg>
              <svg
                v-else
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
              </svg>
              <span>{{ theme === "dark" ? "Light Mode" : "Dark Mode" }}</span>
            </button>
          </div>
        </div>
      </section>

      <!-- Identity Section -->
      <section class="settings-section">
        <div class="section-header">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
            <line x1="16" y1="2" x2="16" y2="6" />
            <line x1="8" y1="2" x2="8" y2="6" />
            <circle cx="12" cy="14" r="4" />
          </svg>
          <h2>Identity</h2>
        </div>
        <div class="setting-card">
          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Your Peer ID</span>
              <span class="setting-description"
                >Your unique identifier on the network</span
              >
            </div>
            <div class="setting-value">
              <code class="peer-id">{{ peerId.slice(0, 16) }}...</code>
              <button
                class="icon-btn"
                @click="copyPeerId"
                title="Copy full Peer ID"
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect x="9" y="9" width="13" height="13" rx="2" />
                  <path
                    d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                  />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- Storage Section -->
      <section class="settings-section">
        <div class="section-header">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <ellipse cx="12" cy="5" rx="9" ry="3" />
            <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
            <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
          </svg>
          <h2>Storage</h2>
        </div>
        <div class="setting-card">
          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Download Directory</span>
              <span class="setting-description"
                >Where downloaded files will be saved</span
              >
            </div>
            <div class="setting-value path-value">
              <code>{{ config.downloadDir }}</code>
              <button class="btn-action" @click="changeDownloadDir">
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                  />
                </svg>
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
        <div class="section-header">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="2" y1="12" x2="22" y2="12" />
            <path
              d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"
            />
          </svg>
          <h2>Network</h2>
        </div>
        <div class="setting-card">
          <div class="setting-row upload-row">
            <div class="setting-info">
              <span class="setting-label">Max Upload Rate</span>
              <span class="setting-description">
                Limit your upload bandwidth
                <span class="current-value"
                  >(Current: {{ config.maxUploadRateHuman }})</span
                >
              </span>
            </div>
            <div class="setting-controls">
              <label class="toggle-switch">
                <input
                  type="checkbox"
                  :checked="isUnlimited"
                  @change="toggleUnlimited"
                />
                <span class="toggle-slider"></span>
                <span class="toggle-label">Unlimited</span>
              </label>
              <Transition name="slide-up">
                <div class="rate-input" v-if="!isUnlimited">
                  <input
                    type="number"
                    v-model="uploadRateInput"
                    placeholder="Rate"
                    min="0"
                    class="number-input"
                  />
                  <span class="unit">KB/s</span>
                  <button class="btn-action" @click="updateUploadRate">
                    Apply
                  </button>
                </div>
              </Transition>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">DHT Refresh</span>
              <span class="setting-description"
                >Manually refresh the distributed hash table</span
              >
            </div>
            <button
              class="btn-secondary"
              @click="refreshDHT"
              :disabled="isRefreshing"
            >
              <svg
                :class="{ spin: isRefreshing }"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M21 12a9 9 0 1 1-6.219-8.56" />
              </svg>
              {{ isRefreshing ? "Refreshing..." : "Refresh DHT" }}
            </button>
          </div>
        </div>
      </section>

      <!-- Performance Section -->
      <section class="settings-section">
        <div class="section-header">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
          </svg>
          <h2>Performance</h2>
        </div>
        <div class="setting-card">
          <div class="setting-row upload-row">
            <div class="setting-info">
              <span class="setting-label">Max Download Rate</span>
              <span class="setting-description">
                Limit your download bandwidth
                <span class="current-value"
                  >(Current:
                  {{ config.maxDownloadRateHuman || "Unlimited" }})</span
                >
              </span>
            </div>
            <div class="setting-controls">
              <label class="toggle-switch">
                <input
                  type="checkbox"
                  :checked="isDownloadUnlimited"
                  @change="toggleDownloadUnlimited"
                />
                <span class="toggle-slider"></span>
                <span class="toggle-label">Unlimited</span>
              </label>
              <Transition name="slide-up">
                <div class="rate-input" v-if="!isDownloadUnlimited">
                  <input
                    type="number"
                    v-model="downloadRateInput"
                    placeholder="Rate"
                    min="0"
                    class="number-input"
                  />
                  <span class="unit">KB/s</span>
                  <button class="btn-action" @click="updateDownloadRate">
                    Apply
                  </button>
                </div>
              </Transition>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Max Parallel Downloads</span>
              <span class="setting-description"
                >Maximum number of simultaneous downloads (1-20)</span
              >
            </div>
            <div class="parallel-input">
              <input
                type="number"
                v-model="maxParallelInput"
                min="1"
                max="20"
                class="number-input small"
              />
              <button class="btn-action" @click="updateMaxParallel">
                Apply
              </button>
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Adaptive Parallel Downloads</span>
              <span class="setting-description"
                >Automatically adjust parallel downloads based on
                bandwidth</span
              >
            </div>
            <label class="toggle-switch">
              <input
                type="checkbox"
                :checked="adaptiveEnabled"
                @change="toggleAdaptive"
              />
              <span class="toggle-slider"></span>
              <span class="toggle-label">{{
                adaptiveEnabled ? "Enabled" : "Disabled"
              }}</span>
            </label>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Endgame Mode</span>
              <span class="setting-description"
                >Request last pieces from multiple peers to speed up
                completion</span
              >
            </div>
            <label class="toggle-switch">
              <input
                type="checkbox"
                :checked="endgameEnabled"
                @change="toggleEndgame"
              />
              <span class="toggle-slider"></span>
              <span class="toggle-label">{{
                endgameEnabled ? "Enabled" : "Disabled"
              }}</span>
            </label>
          </div>
        </div>
      </section>

      <!-- Web Share Portal Section -->
      <section class="settings-section">
        <div class="section-header">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="2" y1="12" x2="22" y2="12" />
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
          </svg>
          <h2>Web Share Portal</h2>
        </div>
        <div class="setting-card">
          <div class="setting-row vertical">
            <div class="setting-info">
              <span class="setting-label">Portal URL</span>
              <span class="setting-description">
                URL of the web share portal for publishing files
              </span>
            </div>
            <div class="input-with-button">
              <input
                type="url"
                v-model="webShareConfig.portalUrl"
                placeholder="https://torrentium-webshare.onrender.com/portal/#/"
                class="text-input full-width"
              />
            </div>
          </div>

          <div class="setting-row vertical">
            <div class="setting-info">
              <span class="setting-label">API Key (Optional)</span>
              <span class="setting-description">
                Authentication key if the portal requires it
              </span>
            </div>
            <div class="input-with-button">
              <input
                type="password"
                v-model="webShareConfig.apiKey"
                placeholder="Leave empty if not required"
                class="text-input full-width"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Default Visibility</span>
              <span class="setting-description">
                Default visibility for published files
              </span>
            </div>
            <select v-model="webShareConfig.defaultVisibility" class="select-input">
              <option value="public">🌐 Public (searchable)</option>
              <option value="unlisted">🔗 Unlisted (link only)</option>
            </select>
          </div>

          <div class="setting-row">
            <div class="setting-info">
              <span class="setting-label">Default Expiration</span>
              <span class="setting-description">
                Auto-expire published files after this time
              </span>
            </div>
            <select v-model="webShareConfig.defaultExpiration" class="select-input">
              <option :value="0">Never</option>
              <option :value="24">24 hours</option>
              <option :value="72">3 days</option>
              <option :value="168">1 week</option>
              <option :value="720">30 days</option>
            </select>
          </div>

          <div class="setting-row action-row">
            <button class="btn-primary" @click="updateWebShareConfig">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
                <polyline points="17 21 17 13 7 13 7 21" />
                <polyline points="7 3 7 8 15 8" />
              </svg>
              Save Web Share Settings
            </button>
          </div>
        </div>
      </section>

      <!-- About Section -->
      <section class="settings-section">
        <div class="section-header">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="16" x2="12" y2="12" />
            <line x1="12" y1="8" x2="12.01" y2="8" />
          </svg>
          <h2>About</h2>
        </div>
        <div class="setting-card about-card">
          <div class="about-header">
            <div class="app-logo">
              <img
                src="../assets/images/logo-universal.png"
                alt="Torrentium"
                onerror="this.style.display = 'none'"
              />
              <span class="logo-fallback">🌊</span>
            </div>
            <div class="app-title">
              <h3>Torrentium</h3>
              <span class="version-badge">v1.0.0</span>
            </div>
          </div>
          <p class="about-description">
            A decentralized peer-to-peer file sharing application built with
            libp2p and WebRTC for secure, fast, and reliable file transfers.
          </p>
          <div class="features-grid">
            <div class="feature-item">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>Decentralized sharing</span>
            </div>
            <div class="feature-item">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>DHT peer discovery</span>
            </div>
            <div class="feature-item">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>WebRTC data channels</span>
            </div>
            <div class="feature-item">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>Rarest-first selection</span>
            </div>
            <div class="feature-item">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>Upload rate limiting</span>
            </div>
            <div class="feature-item">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>Multi-file support</span>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.settings-page {
  width: 100%;
  max-width: 1600px;
  margin: 0 auto;
  animation: fadeIn 0.4s ease;
}

/* Header */
.page-header {
  margin-bottom: 32px;
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

/* Loading */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80px;
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

/* Section */
.settings-section {
  margin-bottom: 28px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  color: var(--accent);
}

.section-header h2 {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

/* Setting Card */
.setting-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  overflow: hidden;
}

.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
  gap: 20px;
}

.setting-row:last-child {
  border-bottom: none;
}

.setting-row.upload-row {
  flex-direction: column;
  align-items: stretch;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.setting-label {
  font-weight: 600;
  font-size: 0.95rem;
}

.setting-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.current-value {
  color: var(--accent);
}

.setting-value {
  display: flex;
  align-items: center;
  gap: 12px;
}

.setting-value.path-value {
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.setting-value code {
  background: var(--bg-tertiary);
  padding: 10px 14px;
  border-radius: var(--radius-md);
  font-size: 0.8rem;
  font-family: "SF Mono", "Consolas", monospace;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}

.peer-id {
  color: var(--accent) !important;
}

/* Buttons */
.icon-btn {
  width: 40px;
  height: 40px;
  border: none;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.icon-btn:hover {
  background: var(--accent);
  color: var(--bg-primary);
}

.btn-action {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-action:hover {
  background: var(--accent);
  color: var(--bg-primary);
  border-color: var(--accent);
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--border);
}

.btn-secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary .spin {
  animation: spin 1s linear infinite;
}

/* Theme Toggle */
.theme-toggle-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.theme-toggle-btn:hover {
  background: var(--accent);
  color: var(--bg-primary);
  border-color: var(--accent);
}

/* Toggle Switch */
.toggle-switch {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
}

.toggle-switch input {
  display: none;
}

.toggle-slider {
  width: 44px;
  height: 24px;
  background: var(--bg-tertiary);
  border-radius: 12px;
  position: relative;
  transition: all var(--transition-fast);
}

.toggle-slider::after {
  content: "";
  position: absolute;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--text-secondary);
  top: 3px;
  left: 3px;
  transition: all var(--transition-fast);
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent);
}

.toggle-switch input:checked + .toggle-slider::after {
  background: var(--bg-primary);
  transform: translateX(20px);
}

.toggle-label {
  font-size: 0.9rem;
  color: var(--text-secondary);
}

/* Setting Controls */
.setting-controls {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 14px;
  margin-top: 16px;
}

.rate-input {
  display: flex;
  align-items: center;
  gap: 10px;
}

.parallel-input {
  display: flex;
  align-items: center;
  gap: 10px;
}

.number-input {
  width: 100px;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: 0.9rem;
}

.number-input.small {
  width: 70px;
}

.number-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-glow);
}

.unit {
  color: var(--text-muted);
  font-size: 0.9rem;
}

/* About Section */
.about-card {
  padding: 24px;
}

.about-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

/* Web Share Settings Styles */
.setting-row.vertical {
  flex-direction: column;
  align-items: flex-start;
}

.input-with-button {
  width: 100%;
  display: flex;
  gap: 10px;
}

.text-input {
  padding: 10px 14px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: 0.9rem;
  transition: all var(--transition-fast);
}

.text-input:focus {
  border-color: var(--accent);
  outline: none;
  box-shadow: 0 0 0 3px var(--accent-glow);
}

.text-input.full-width {
  width: 100%;
}

.select-input {
  padding: 10px 14px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: 0.9rem;
  cursor: pointer;
  min-width: 180px;
  transition: all var(--transition-fast);
}

.select-input:focus {
  border-color: var(--accent);
  outline: none;
}

.setting-row.action-row {
  border-top: 1px solid var(--border);
  padding-top: 20px;
  margin-top: 8px;
  justify-content: flex-end;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: var(--gradient-primary);
  color: var(--bg-primary);
  border: none;
  border-radius: var(--radius-md);
  font-weight: 500;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all var(--transition-normal);
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-glow);
}

.app-logo {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-md);
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.app-logo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-fallback {
  font-size: 2rem;
}

.app-logo img + .logo-fallback {
  display: none;
}

.app-title h3 {
  font-size: 1.3rem;
  margin-bottom: 4px;
}

.version-badge {
  background: var(--accent-glow);
  color: var(--accent);
  padding: 4px 10px;
  border-radius: var(--radius-xl);
  font-size: 0.75rem;
  font-weight: 600;
}

.about-description {
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: 24px;
  font-size: 0.9rem;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.feature-item svg {
  color: var(--success);
  flex-shrink: 0;
}

/* Animations */
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.3s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

/* Responsive Styles */
@media (max-width: 768px) {
  .settings-page {
    padding: 0;
  }

  .page-header h1 {
    font-size: 1.5rem;
  }

  .settings-section {
    margin-bottom: 20px;
  }

  .setting-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 14px;
    padding: 16px;
  }

  .setting-value {
    width: 100%;
    flex-wrap: wrap;
  }

  .setting-value.path-value {
    align-items: flex-start;
  }

  .setting-value code {
    max-width: 100%;
    word-break: break-all;
    white-space: normal;
  }

  .theme-toggle-btn {
    width: 100%;
    justify-content: center;
  }

  .about-header {
    flex-direction: column;
    text-align: center;
  }

  .features-grid {
    grid-template-columns: 1fr;
  }

  .rate-input {
    flex-wrap: wrap;
  }

  .number-input {
    flex: 1;
    min-width: 80px;
  }
}

@media (max-width: 480px) {
  .section-header h2 {
    font-size: 0.9rem;
  }

  .setting-label {
    font-size: 0.9rem;
  }

  .setting-description {
    font-size: 0.8rem;
  }

  .btn-action,
  .btn-secondary {
    width: 100%;
    justify-content: center;
  }

  .icon-btn {
    width: 36px;
    height: 36px;
  }

  .about-card {
    padding: 16px;
  }

  .app-title h3 {
    font-size: 1.1rem;
  }

  .loading-state {
    padding: 40px 20px;
  }
}
</style>
