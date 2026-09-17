<script setup lang="ts">
  import { AppCloseBehavior, AppEventType } from '@croffledev/common';
  import { onMounted, onUnmounted, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { toast } from 'vue-sonner';

  import { Button } from '@/components/ui/button';
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from '@/components/ui/dialog';
  import { useAppSettingsStore } from '@/stores/app-settings-store';

  /**
   * First-close prompt. Main emits WINDOW_CLOSE_REQUESTED while `general.closeBehavior`
   * is `ask`; the answer is stored as the setting, then the close is re-issued so the
   * window follows it. Dismissing the dialog (Esc / outside click) keeps the window open
   * and leaves the setting untouched, so the next close asks again.
   */
  const { t } = useI18n();
  const appSettingsStore = useAppSettingsStore();

  const isOpen = ref(false);
  const isSaving = ref(false);
  let unsubscribe: (() => void) | null = null;

  const choose = async (closeBehavior: AppCloseBehavior.TRAY | AppCloseBehavior.QUIT) => {
    if (isSaving.value) {
      return;
    }
    isSaving.value = true;
    try {
      const general =
        appSettingsStore.settings?.general ?? (await croffle.settings.getAll()).general;
      await croffle.settings.update({ general: { ...general, closeBehavior } });
      isOpen.value = false;
      if (closeBehavior === AppCloseBehavior.QUIT) {
        await croffle.window.exitApp();
      } else {
        await croffle.window.close();
      }
    } catch (error) {
      toast.error(t('settings.errors.save', { error: JSON.stringify(error) }));
    } finally {
      isSaving.value = false;
    }
  };

  onMounted(() => {
    unsubscribe = croffle.event.on(AppEventType.WINDOW_CLOSE_REQUESTED, () => {
      isOpen.value = true;
    });
  });

  onUnmounted(() => {
    unsubscribe?.();
    unsubscribe = null;
  });
</script>

<template>
  <Dialog :open="isOpen" @update:open="(open: boolean) => (isOpen = open)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t('closeModal.title') }}</DialogTitle>
        <DialogDescription>{{ t('closeModal.description') }}</DialogDescription>
      </DialogHeader>
      <p class="text-muted-foreground text-xs">{{ t('closeModal.hint') }}</p>
      <DialogFooter class="gap-2">
        <Button variant="outline" :disabled="isSaving" @click="choose(AppCloseBehavior.QUIT)">
          {{ t('closeModal.quit') }}
        </Button>
        <Button :disabled="isSaving" @click="choose(AppCloseBehavior.TRAY)">
          {{ t('closeModal.tray') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
