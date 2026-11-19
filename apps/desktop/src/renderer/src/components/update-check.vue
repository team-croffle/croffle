<script setup lang="ts">
  import { storeToRefs } from 'pinia';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  import { Button } from '@/components/ui/button';
  import { Icon } from '@/components/ui/icon';
  import { useUpdateStore } from '@/stores/update-store';

  /** "Check for updates": the button and its state. Same store in the version tile and Settings. */
  /** `now`: the Settings row's "지금 확인"; default: the version tile's "업데이트 확인". */
  const props = defineProps<{ label?: 'check' | 'now' }>();

  const updateStore = useUpdateStore();
  const { checkStatus } = storeToRefs(updateStore);
  const { t } = useI18n();

  /** A development build cannot check; say so before the click. */
  const isDev = import.meta.env.DEV;

  const statusText = computed(() => {
    if (isDev) {
      return t('update.check.unsupportedDevelopment');
    }
    const s = checkStatus.value;
    switch (s.kind) {
      case 'idle':
        return '';
      case 'checking':
        return t('update.check.checking');
      case 'latest':
        return t('update.check.latest');
      case 'available':
        return t('update.check.available', { version: s.version });
      case 'error':
        return t('update.check.failed', { message: s.message });
      case 'unsupported':
        return s.reason === 'development'
          ? t('update.check.unsupportedDevelopment')
          : t('update.check.unsupportedPackage');
    }
    return '';
  });

  const statusClass = computed(() =>
    checkStatus.value.kind === 'error' && !isDev ? 'text-destructive' : 'text-muted-foreground',
  );

  const onClick = () => {
    if (checkStatus.value.kind === 'available') {
      updateStore.isModalOpen = true;
      return;
    }
    void updateStore.checkNow();
  };
</script>

<template>
  <div class="flex flex-col gap-1.5">
    <Button
      type="button"
      variant="outline"
      :disabled="isDev || checkStatus.kind === 'checking'"
      @click="onClick"
    >
      <Icon
        :icon="checkStatus.kind === 'checking' ? 'lucide:loader-circle' : 'lucide:refresh-cw'"
        :class="{ 'animate-spin': checkStatus.kind === 'checking' }"
      />
      {{
        checkStatus.kind === 'available'
          ? $t('update.check.openUpdate')
          : props.label === 'now'
            ? $t('update.check.now')
            : $t('update.check.button')
      }}
    </Button>
    <p v-if="statusText" class="text-xs" :class="statusClass" role="status" aria-live="polite">
      {{ statusText }}
    </p>
  </div>
</template>
