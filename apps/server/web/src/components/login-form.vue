<script setup lang="ts">
  import { ref } from 'vue';

  import { ApiError, api } from '../api';

  const emit = defineEmits<{ signedIn: [] }>();

  const username = ref('');
  const password = ref('');
  const error = ref('');
  const busy = ref(false);

  async function submit() {
    busy.value = true;
    error.value = '';
    try {
      await api.login(username.value, password.value);
      // The password never leaves this component, and is dropped as soon as the
      // session cookie exists.
      password.value = '';
      emit('signedIn');
    } catch (err) {
      error.value =
        err instanceof ApiError && err.status === 401
          ? 'Wrong username or password.'
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
      This dashboard is read-only. Accounts, tokens and teams are managed with
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

    <p v-if="error" class="error">{{ error }}</p>
    <button class="primary" type="submit" :disabled="busy">
      {{ busy ? 'Signing in…' : 'Sign in' }}
    </button>
  </form>
</template>
