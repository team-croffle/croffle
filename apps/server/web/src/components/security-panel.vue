<script setup lang="ts">
  import { ref } from 'vue';

  import { api, type Me, type OtpEnrolment } from '../api';
  import PasswordForm from './password-form.vue';

  const props = defineProps<{ me: Me }>();
  const emit = defineEmits<{ changed: [] }>();

  const enrolment = ref<OtpEnrolment | null>(null);
  const recoveryCodes = ref<string[]>([]);
  const code = ref('');
  const password = ref('');
  const error = ref('');
  const busy = ref(false);

  async function run(action: () => Promise<void>) {
    busy.value = true;
    error.value = '';
    try {
      await action();
    } catch (err) {
      error.value = (err as Error).message;
    } finally {
      busy.value = false;
    }
  }

  const start = () =>
    run(async () => {
      recoveryCodes.value = [];
      enrolment.value = await api.otpEnroll();
    });

  const confirm = () =>
    run(async () => {
      recoveryCodes.value = (await api.otpConfirm(code.value)).recoveryCodes;
      enrolment.value = null;
      code.value = '';
      emit('changed');
    });

  const disable = () =>
    run(async () => {
      await api.otpDisable(password.value, code.value);
      password.value = '';
      code.value = '';
      emit('changed');
    });
</script>

<template>
  <section class="security">
    <h2>One-time codes</h2>

    <template v-if="recoveryCodes.length">
      <p class="notice">
        One-time codes are on. Keep these recovery codes somewhere safe: each signs in once without
        your authenticator, and they are not shown again.
      </p>
      <ul class="codes">
        <li v-for="recovery in recoveryCodes" :key="recovery" class="mono">{{ recovery }}</li>
      </ul>
    </template>

    <template v-else-if="props.me.otpEnabled">
      <p>
        Signing in (desktop and dashboard) asks for a code from your authenticator app. To turn this
        off, confirm with your password and a current code or a recovery code.
      </p>
      <form class="stack" @submit.prevent="disable">
        <label>
          Password
          <input v-model="password" type="password" autocomplete="current-password" required />
        </label>
        <label>
          Code
          <input v-model="code" autocomplete="one-time-code" required />
        </label>
        <button class="primary" type="submit" :disabled="busy">Turn off</button>
      </form>
    </template>

    <template v-else-if="enrolment">
      <p>
        Scan this with an authenticator app, or enter the secret by hand, then type the code it
        shows.
      </p>
      <img
        :src="enrolment.qrPng"
        alt="QR code for the authenticator app"
        width="200"
        height="200"
      />
      <p class="mono">{{ enrolment.secret }}</p>
      <form class="stack" @submit.prevent="confirm">
        <label>
          Code
          <input v-model="code" autocomplete="one-time-code" inputmode="numeric" required />
        </label>
        <button class="primary" type="submit" :disabled="busy">Confirm</button>
      </form>
    </template>

    <template v-else>
      <p>Off. Turn it on to require a code from an authenticator app at every sign-in.</p>
      <button class="primary" :disabled="busy" @click="start">Set up</button>
    </template>

    <p v-if="error" class="error">{{ error }}</p>
  </section>
  <PasswordForm :otp-enabled="props.me.otpEnabled" />
</template>
