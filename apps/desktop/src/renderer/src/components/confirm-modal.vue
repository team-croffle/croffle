<script setup lang="ts">
  import { storeToRefs } from 'pinia';
  import { useI18n } from 'vue-i18n';

  import { Button } from '@/components/ui/button';
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from '@/components/ui/dialog';
  import { useUiStore } from '@/stores/ui-store';

  const { t } = useI18n();
  const uiStore = useUiStore();
  const {
    isConfirmModalOpen,
    confirmTitle,
    confirmDescription,
    confirmConfirmLabel,
    confirmCancelLabel,
    confirmVariant,
  } = storeToRefs(uiStore);

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      uiStore.resolveConfirm(false);
    }
  };
</script>

<template>
  <Dialog :open="isConfirmModalOpen" @update:open="handleOpenChange">
    <!-- Above any other dialog: a confirmation can be asked from inside one (settings). -->
    <DialogContent class="z-[60] sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ confirmTitle }}</DialogTitle>
        <DialogDescription>{{ confirmDescription }}</DialogDescription>
      </DialogHeader>
      <DialogFooter class="gap-2">
        <Button variant="outline" class="min-w-20" @click="uiStore.resolveConfirm(false)">
          {{ confirmCancelLabel || t('common.cancel') }}
        </Button>
        <Button :variant="confirmVariant" class="min-w-20" @click="uiStore.resolveConfirm(true)">
          {{ confirmConfirmLabel || t('common.delete') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
