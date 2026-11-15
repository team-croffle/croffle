<script setup lang="ts">
  import { ref } from 'vue';

  import { ApiError, api } from '../api';

  const emit = defineEmits<{ signedIn: [] }>();

  const username = ref('');
  const password = ref('');
  const otp = ref('');
  // Shown once the server answers otp_required: the account has one-time codes.
  const needsOtp = ref(false);
  const error = ref('');
  const busy = ref(false);

  async function submit() {
    busy.value = true;
    error.value = '';
    try {
      await api.login(username.value, password.value, otp.value);
      // The password never leaves this component, and is dropped as soon as the
      // session cookie exists.
      password.value = '';
      otp.value = '';
      emit('signedIn');
    } catch (err) {
      if (err instanceof ApiError && err.code === 'otp_required') {
        needsOtp.value = true;
        return;
      }
      if (err instanceof ApiError && err.status === 429) {
        error.value = 'Too many attempts. Wait a little and try again.';
        return;
      }
      error.value =
        err instanceof ApiError && err.status === 401
          ? needsOtp.value
            ? 'Wrong username, password or code.'
            : 'Wrong username or password.'
          : (err as Error).message;
    } finally {
      busy.value = false;
    }
  }
</script>

<template>
  <form class="login" @submit.prevent="submit">
    <h1>Croffle Server</h1>
    <p class="notice">
      Shows this account's own data. Accounts, other people's devices and teams are managed with
      <code>croffle-server</code> commands.
    </p>

    <label>
      Username
      <input v-model="username" autocomplete="username" autofocus required />
    </label>
    <label>
      Password
      <input v-model="password" type="password" autocomplete="current-password" required />
    </label>
    <label v-if="needsOtp">
      One-time code (or a recovery code)
      <input v-model="otp" autocomplete="one-time-code" inputmode="text" required />
    </label>

    <p v-if="error" class="error">{{ error }}</p>
    <button class="primary" type="submit" :disabled="busy">
      {{ busy ? 'Signing in…' : 'Sign in' }}
    </button>
  </form>
</template>
