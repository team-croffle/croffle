<script setup lang="ts">
  import { ref } from 'vue';

  import { api } from '../api';

  const props = defineProps<{ otpEnabled: boolean }>();

  const current = ref('');
  const next = ref('');
  const repeat = ref('');
  const code = ref('');
  const error = ref('');
  const done = ref(false);
  const busy = ref(false);

  async function submit() {
    error.value = '';
    done.value = false;
    if (next.value !== repeat.value) {
      error.value = 'The two new passwords do not match.';
      return;
    }
    busy.value = true;
    try {
      await api.changePassword(current.value, next.value, props.otpEnabled ? code.value : '');
      current.value = next.value = repeat.value = code.value = '';
      done.value = true;
    } catch (err) {
      error.value = (err as Error).message;
    } finally {
      busy.value = false;
    }
  }
</script>

<template>
  <section class="security">
    <h2>Password</h2>
    <p>
      Changing it signs out every other browser. This one stays signed in, and desktop devices keep
      syncing.
    </p>
    <form class="stack" @submit.prevent="submit">
      <label>
        Current password
        <input v-model="current" type="password" autocomplete="current-password" required />
      </label>
      <label>
        New password
        <input
          v-model="next"
          type="password"
          autocomplete="new-password"
          maxlength="4096"
          required
        />
      </label>
      <label>
        New password again
        <input
          v-model="repeat"
          type="password"
          autocomplete="new-password"
          maxlength="4096"
          required
        />
      </label>
      <label v-if="props.otpEnabled">
        Code
        <input v-model="code" autocomplete="one-time-code" required />
      </label>
      <button class="primary" type="submit" :disabled="busy">Change password</button>
    </form>
    <p v-if="done" class="notice">Password changed. Other browsers are signed out.</p>
    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>
