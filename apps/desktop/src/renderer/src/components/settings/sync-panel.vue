<script setup lang="ts">
  import type { SyncActionResult } from '@croffledev/common';
  import { computed, onMounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { toast } from 'vue-sonner';

  import { Button } from '@/components/ui/button';
  import { Icon } from '@/components/ui/icon';
  import { Label } from '@/components/ui/label';
  import { Separator } from '@/components/ui/separator';
  import { Switch } from '@/components/ui/switch';
  import { useSyncStore } from '@/stores/sync-store';
  import { useUiStore } from '@/stores/ui-store';

  const { t, locale } = useI18n();
  const syncStore = useSyncStore();
  const uiStore = useUiStore();

  const status = computed(() => syncStore.status);
  const serverUrl = ref('');
  const authMode = ref<'password' | 'token'>('password');
  const username = ref('');
  const password = ref('');
  const otp = ref('');
  const otpRequired = ref(false);
  const token = ref('');
  const busy = ref(false);

  const inputClass =
    'w-full rounded-lg border border-neutral-300 bg-transparent px-3 py-2 text-sm placeholder:text-neutral-400 focus:border-[#A68A64] focus:outline-none focus:ring-1 focus:ring-[#A68A64] dark:border-neutral-700';
  const primaryButton = 'h-9 gap-2 border-none bg-[#A68A64] text-white hover:bg-[#8E7554]';
  const secondaryButton =
    'h-9 gap-2 border-none bg-neutral-200 text-foreground hover:bg-neutral-300 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700';

  onMounted(() => {
    void syncStore.refresh();
  });

  watch(
    () => status.value?.url,
    (url) => {
      serverUrl.value = url ?? '';
    },
    { immediate: true },
  );

  const lastSynced = computed(() =>
    status.value?.lastSyncedAt
      ? new Date(status.value.lastSyncedAt).toLocaleString(locale.value)
      : t('settings.sync.never'),
  );

  const errorText = computed(() => {
    const kind = status.value?.lastErrorKind;
    return kind ? t(`settings.sync.errors.${kind}`) : '';
  });

  const clockSkewMinutes = computed(() =>
    status.value?.clockSkewMs ? Math.round(Math.abs(status.value.clockSkewMs) / 60_000) : 0,
  );

  function report(result: SyncActionResult): boolean {
    if (result.ok) {
      return true;
    }
    const key =
      result.kind === 'invalid'
        ? 'settings.sync.errors.invalidUrl'
        : `settings.sync.errors.${result.kind}`;
    toast.error(t(key), { description: result.message });
    return false;
  }

  async function run(action: () => Promise<void>): Promise<void> {
    busy.value = true;
    try {
      await action();
    } finally {
      busy.value = false;
    }
  }

  const onSaveServer = () =>
    run(async () => {
      const next = serverUrl.value.trim();
      if (!next || next === status.value?.url) {
        return;
      }
      if (status.value?.url) {
        const confirmed = await uiStore.openConfirm({
          title: t('settings.sync.changeServerTitle'),
          description: t('settings.sync.changeServerConfirm'),
          confirmLabel: t('settings.sync.changeServerAction'),
          confirmVariant: 'destructive',
        });
        if (!confirmed) {
          serverUrl.value = status.value.url;
          return;
        }
      }
      if (report(await syncStore.setServer(next))) {
        toast.success(t('settings.sync.serverSaved'));
      }
    });

  const onSignIn = () =>
    run(async () => {
      const result =
        authMode.value === 'password'
          ? await syncStore.signInWithPassword({
              username: username.value.trim(),
              password: password.value,
              ...(otpRequired.value && otp.value ? { otp: otp.value.trim() } : {}),
            })
          : await syncStore.signInWithToken(token.value);

      if (!result.ok && result.code === 'otp_required') {
        otpRequired.value = true;
        toast(t('settings.sync.otpRequired'));
        return;
      }
      // Secrets never stay in the form, whatever the outcome.
      password.value = '';
      token.value = '';
      otp.value = '';
      if (result.ok) {
        otpRequired.value = false;
        toast.success(t('settings.sync.signedInAs', { account: result.accountId ?? '' }));
      } else {
        report(result);
      }
    });

  const onSignOut = () => run(() => syncStore.signOut());

  const onToggleEnabled = (enabled: boolean) => run(() => syncStore.setEnabled(enabled));

  const onToggleInsecureTls = (allow: boolean) => run(() => syncStore.setAllowInsecureTls(allow));

  const onSyncNow = () => run(() => syncStore.syncNow());
</script>

<template>
  <div v-if="status" class="space-y-6">
    <p class="text-muted-foreground text-sm">{{ $t('settings.sync.intro') }}</p>

    <div
      class="flex items-start gap-2 rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-100"
    >
      <Icon icon="lucide:flask-conical" class="mt-0.5 h-4 w-4 shrink-0" />
      <span>{{ $t('settings.sync.beta') }}</span>
    </div>

    <!-- 서버 -->
    <div class="space-y-2">
      <Label for="settings-sync-server" class="text-sm font-semibold">
        {{ $t('settings.sync.serverUrl') }}
      </Label>
      <div class="flex items-center gap-3">
        <input
          id="settings-sync-server"
          v-model="serverUrl"
          type="url"
          :placeholder="$t('settings.sync.serverUrlPlaceholder')"
          :class="inputClass"
          @keydown.enter="onSaveServer"
        />
        <Button
          type="button"
          :class="primaryButton"
          :disabled="busy || !serverUrl.trim() || serverUrl.trim() === status.url"
          @click="onSaveServer"
        >
          {{ $t('settings.sync.save') }}
        </Button>
      </div>
    </div>

    <!-- 인증 -->
    <div v-if="status.configured" class="space-y-3">
      <Label class="text-sm font-semibold">{{ $t('settings.sync.account') }}</Label>

      <div v-if="status.signedIn" class="flex items-center justify-between gap-3">
        <p class="text-sm">
          {{ $t('settings.sync.signedInAs', { account: status.accountId ?? '' }) }}
        </p>
        <Button type="button" :class="secondaryButton" :disabled="busy" @click="onSignOut">
          {{ $t('settings.sync.signOut') }}
        </Button>
      </div>

      <template v-else>
        <div class="flex gap-2" role="tablist">
          <Button
            v-for="mode in ['password', 'token'] as const"
            :key="mode"
            type="button"
            role="tab"
            :aria-selected="authMode === mode"
            :class="authMode === mode ? primaryButton : secondaryButton"
            @click="authMode = mode"
          >
            {{ $t(mode === 'password' ? 'settings.sync.authPassword' : 'settings.sync.authToken') }}
          </Button>
        </div>

        <form class="space-y-2" @submit.prevent="onSignIn">
          <template v-if="authMode === 'password'">
            <input
              v-model="username"
              type="text"
              autocomplete="username"
              :placeholder="$t('settings.sync.username')"
              :aria-label="$t('settings.sync.username')"
              :class="inputClass"
            />
            <input
              v-model="password"
              type="password"
              autocomplete="current-password"
              :placeholder="$t('settings.sync.password')"
              :aria-label="$t('settings.sync.password')"
              :class="inputClass"
            />
            <input
              v-if="otpRequired"
              v-model="otp"
              type="text"
              inputmode="numeric"
              autocomplete="one-time-code"
              :placeholder="$t('settings.sync.otp')"
              :aria-label="$t('settings.sync.otp')"
              :class="inputClass"
            />
          </template>
          <input
            v-else
            v-model="token"
            type="password"
            autocomplete="off"
            :placeholder="$t('settings.sync.token')"
            :aria-label="$t('settings.sync.token')"
            :class="inputClass"
          />
          <Button
            type="submit"
            :class="primaryButton"
            :disabled="
              busy || (authMode === 'password' ? !username.trim() || !password : !token.trim())
            "
          >
            <Icon v-if="busy" icon="lucide:loader-2" class="h-4 w-4 animate-spin" />
            {{ $t('settings.sync.signIn') }}
          </Button>
        </form>
      </template>

      <p v-if="status.tokenVolatile" class="text-xs text-amber-700 dark:text-amber-300">
        {{ $t('settings.sync.volatileToken') }}
      </p>
    </div>

    <!-- 사용 -->
    <div
      v-if="status.configured"
      class="flex items-center justify-between border-b border-neutral-100 py-2"
    >
      <div class="min-w-0 space-y-0.5">
        <Label class="text-sm font-semibold">{{ $t('settings.sync.enable') }}</Label>
        <p class="text-muted-foreground text-xs">
          {{ status.signedIn ? $t('settings.sync.enableHint') : $t('settings.sync.signInFirst') }}
        </p>
      </div>
      <Switch
        :model-value="status.enabled"
        :disabled="busy || !status.signedIn"
        :aria-label="$t('settings.sync.enable')"
        @update:model-value="onToggleEnabled"
      />
    </div>

    <!-- 상태 -->
    <div v-if="status.configured" class="space-y-2">
      <Label class="text-sm font-semibold">{{ $t('settings.sync.status') }}</Label>
      <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
        <dt class="text-muted-foreground">{{ $t('settings.sync.lastSyncedAt') }}</dt>
        <dd>{{ lastSynced }}</dd>
        <dt class="text-muted-foreground">{{ $t('settings.sync.pending') }}</dt>
        <dd>
          {{ $t('settings.sync.pendingCount', { count: status.pending }) }}
          <span v-if="status.failed" class="text-red-600 dark:text-red-400">
            · {{ $t('settings.sync.failedCount', { count: status.failed }) }}
          </span>
        </dd>
      </dl>
      <p v-if="errorText" class="text-xs text-red-600 dark:text-red-400" role="alert">
        {{ errorText }}
      </p>
      <p v-if="clockSkewMinutes" class="text-xs text-amber-700 dark:text-amber-300">
        {{ $t('settings.sync.clockSkew', { minutes: clockSkewMinutes }) }}
      </p>
      <Button
        type="button"
        :class="secondaryButton"
        :disabled="busy || status.running || !status.enabled || !status.signedIn"
        @click="onSyncNow"
      >
        <Icon icon="lucide:refresh-cw" :class="['h-4 w-4', status.running ? 'animate-spin' : '']" />
        {{ status.running ? $t('settings.sync.syncing') : $t('settings.sync.syncNow') }}
      </Button>
    </div>

    <!-- 고급 -->
    <template v-if="status.configured">
      <Separator />
      <div class="flex items-center justify-between py-2">
        <div class="min-w-0 space-y-0.5">
          <Label class="text-sm font-semibold">{{ $t('settings.sync.allowInsecureTls') }}</Label>
          <p class="text-xs text-amber-700 dark:text-amber-300">
            {{ $t('settings.sync.allowInsecureTlsWarning') }}
          </p>
        </div>
        <Switch
          :model-value="status.allowInsecureTls"
          :disabled="busy"
          :aria-label="$t('settings.sync.allowInsecureTls')"
          @update:model-value="onToggleInsecureTls"
        />
      </div>
    </template>
  </div>
</template>
