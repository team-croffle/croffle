import type { SyncActionResult, SyncPasswordSignIn, SyncStatus } from '@croffledev/common';
import { defineStore } from 'pinia';
import { ref, watch } from 'vue';

import { croffleInternal } from '@/services/host-bridge';
import { useScheduleStore } from '@/stores/schedule-store';
import { useTagStore } from '@/stores/tag-store';

/** Sync status from main, plus the actions the settings panel calls. */
export const useSyncStore = defineStore('sync', () => {
  const status = ref<SyncStatus | null>(null);
  let unsubscribe: (() => void) | null = null;

  async function refresh(): Promise<void> {
    status.value = await croffleInternal.sync.getStatus();
  }

  function init(): void {
    if (unsubscribe) {
      return;
    }
    unsubscribe = croffleInternal.sync.onStatus((next) => {
      status.value = next;
    });
    void refresh();

    // Rows from the server changed local data: reload what the views show.
    const scheduleStore = useScheduleStore();
    const tagStore = useTagStore();
    watch(
      () => status.value?.dataChangedAt,
      (changedAt, previous) => {
        if (changedAt && changedAt !== previous) {
          void scheduleStore.reload();
          void tagStore.loadTags(true);
        }
      },
    );
  }

  function dispose(): void {
    unsubscribe?.();
    unsubscribe = null;
  }

  async function after<T>(action: Promise<T>): Promise<T> {
    const result = await action;
    await refresh();
    return result;
  }

  return {
    status,
    init,
    dispose,
    refresh,
    setServer: (url: string): Promise<SyncActionResult> =>
      after(croffleInternal.sync.setServer(url)),
    setAllowInsecureTls: (allow: boolean) => after(croffleInternal.sync.setAllowInsecureTls(allow)),
    signInWithPassword: (input: SyncPasswordSignIn): Promise<SyncActionResult> =>
      after(croffleInternal.sync.signInWithPassword(input)),
    signInWithToken: (token: string): Promise<SyncActionResult> =>
      after(croffleInternal.sync.signInWithToken(token)),
    signOut: () => after(croffleInternal.sync.signOut()),
    setEnabled: (enabled: boolean) => after(croffleInternal.sync.setEnabled(enabled)),
    syncNow: () => after(croffleInternal.sync.syncNow()),
  };
});
