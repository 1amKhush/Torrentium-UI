<script setup>
defineProps({
  message: String,
  type: {
    type: String,
    default: 'success'
  }
});

const emit = defineEmits(['close']);

const icons = {
  success: '✓',
  error: '✕',
  warning: '⚠',
  info: 'ℹ'
};
</script>

<template>
  <div :class="['toast', type]">
    <div class="toast-icon">{{ icons[type] }}</div>
    <span class="toast-message">{{ message }}</span>
    <button class="toast-close" @click="emit('close')">×</button>
  </div>
</template>

<style scoped>
.toast {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-lg);
  min-width: 320px;
  max-width: 450px;
}

.toast.success {
  border-left: 4px solid var(--success);
}

.toast.error {
  border-left: 4px solid var(--error);
}

.toast.warning {
  border-left: 4px solid var(--warning);
}

.toast.info {
  border-left: 4px solid var(--accent);
}

.toast-icon {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.8rem;
  font-weight: bold;
  flex-shrink: 0;
}

.toast.success .toast-icon {
  background: var(--success-glow);
  color: var(--success);
}

.toast.error .toast-icon {
  background: var(--error-glow);
  color: var(--error);
}

.toast.warning .toast-icon {
  background: var(--warning-glow);
  color: var(--warning);
}

.toast.info .toast-icon {
  background: var(--accent-glow);
  color: var(--accent);
}

.toast-message {
  flex: 1;
  font-size: 0.9rem;
  color: var(--text-primary);
}

.toast-close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0;
  line-height: 1;
  transition: color var(--transition-fast);
}

.toast-close:hover {
  color: var(--text-primary);
}
</style>
