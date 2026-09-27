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
  import { useChoiceStore } from '@/stores/choice-store';

  const { t } = useI18n();
  const choiceStore = useChoiceStore();
  const { isOpen, options } = storeToRefs(choiceStore);

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      choiceStore.resolveChoice(null);
    }
  };
</script>

<template>
  <Dialog :open="isOpen" @update:open="handleOpenChange">
    <DialogContent v-if="options" class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ options.title }}</DialogTitle>
        <DialogDescription>{{ options.description }}</DialogDescription>
      </DialogHeader>
      <p v-if="options.hint" class="text-muted-foreground text-xs">{{ options.hint }}</p>
      <DialogFooter class="gap-2">
        <Button variant="outline" @click="choiceStore.resolveChoice(null)">
          {{ options.cancelLabel || t('common.cancel') }}
        </Button>
        <Button
          v-for="choice in options.choices"
          :key="choice.value"
          :variant="choice.variant ?? 'default'"
          :disabled="choice.disabled"
          @click="choiceStore.resolveChoice(choice.value)"
        >
          {{ choice.label }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
