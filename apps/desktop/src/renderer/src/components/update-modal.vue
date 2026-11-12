<script setup lang="ts">
  import { storeToRefs } from 'pinia';

  import { Button } from '@/components/ui/button';
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from '@/components/ui/dialog';
  import { useUpdateStore } from '@/stores/update-store';

  const updateStore = useUpdateStore();
  const { isModalOpen, updateInfo, isDownloading, downloadProgress } = storeToRefs(updateStore);
  const { skipUpdate, downloadLater, downloadNow } = updateStore;
</script>

<template>
  <Dialog :open="isModalOpen">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{ $t('update.available') }}</DialogTitle>
      </DialogHeader>
      <DialogDescription>
        {{ $t('update.availableDescription', { version: updateInfo?.version }) }}
      </DialogDescription>

      <div
        v-if="updateInfo?.releaseNotes"
        class="text-muted-foreground max-h-32 overflow-y-auto rounded-md border p-3 text-sm"
        v-html="updateInfo.releaseNotes"
      />
      <a
        v-else-if="updateInfo?.version"
        :href="`https://github.com/team-croffle/croffle/releases/tag/desktop-v${updateInfo.version}`"
        target="_blank"
        rel="noopener noreferrer"
        class="text-sm underline underline-offset-2"
      >
        {{ $t('update.viewReleaseNotes') }}
      </a>

      <div v-if="isDownloading">
        <p>{{ $t('update.downloading', { percent: Math.round(downloadProgress) }) }}</p>
      </div>

      <DialogFooter class="gap-2">
        <Button variant="ghost" class="min-w-20" :disabled="isDownloading" @click="skipUpdate">
          {{ $t('update.skip') }}
        </Button>
        <Button variant="outline" class="min-w-20" :disabled="isDownloading" @click="downloadLater">
          {{ $t('update.later') }}
        </Button>
        <Button class="min-w-20" :disabled="isDownloading" @click="downloadNow">{{
          $t('update.now')
        }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
