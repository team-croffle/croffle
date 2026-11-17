<script setup lang="ts">
  import { ref } from 'vue';

  import { formatTime, type IssuedToken } from '../api';

  defineProps<{ issued: IssuedToken; username: string }>();
  const emit = defineEmits<{ close: [] }>();

  const copied = ref(false);

  async function copy(token: string) {
    try {
      await navigator.clipboard.writeText(token);
      copied.value = true;
    } catch {
      // Clipboard access can be refused (plain HTTP, permissions); the token
      // is selectable on screen either way.
      copied.value = false;
    }
  }
</script>

<template>
  <div class="overlay" role="dialog" aria-modal="true" aria-labelledby="token-title">
    <div class="dialog">
      <h2 id="token-title">Token for {{ username }}</h2>
      <p class="notice">
        Shown once. Paste it into the desktop app (Settings → Sync → device token). Closing this
        window is the last time the server shows it.
      </p>
      <pre class="token">{{ issued.token }}</pre>
      <p class="meta">
        Device <code>{{ issued.deviceId }}</code> ·
        {{
          issued.tokenExpiresAt ? `expires ${formatTime(issued.tokenExpiresAt)}` : 'never expires'
        }}
      </p>
      <div class="actions">
        <button class="primary" @click="copy(issued.token)">
          {{ copied ? 'Copied' : 'Copy' }}
        </button>
        <button class="link" @click="emit('close')">Close</button>
      </div>
    </div>
  </div>
</template>
