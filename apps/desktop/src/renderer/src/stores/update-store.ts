import {
  AppEventType,
  applyCheckAnswer,
  nextUpdateStatus,
  toastsLatest,
  UPDATE_IDLE,
  type UpdateCheckEvent,
  type UpdateCheckStatus,
} from '@croffledev/common';
import { defineStore } from 'pinia';
import { ref } from 'vue';
import { toast } from 'vue-sonner';

import { i18n } from '@/i18n';
import { croffleInternal } from '@/services/host-bridge';
import type { UpdateInfo } from '@/types/update';

function t(key: string) {
  return String(i18n.global.t(key));
}

export const useUpdateStore = defineStore('update', () => {
  const isModalOpen = ref<boolean>(false);
  const updateInfo = ref<UpdateInfo | null>(null);
  const isDownloading = ref<boolean>(false);
  const downloadProgress = ref<number>(0);
  const updateError = ref<string | null>(null);
  /** "Check for updates" state, shared by the version tile and Settings › General. */
  const checkStatus = ref<UpdateCheckStatus>(UPDATE_IDLE);
  /** The running check was started by hand (its result shows in place, without a toast). */
  let manualCheck = false;
  let unsubscribers: (() => void)[] = [];

  const isChecking = () => checkStatus.value.kind === 'checking';

  const apply = (event: UpdateCheckEvent) => {
    checkStatus.value = nextUpdateStatus(checkStatus.value, event);
  };

  /**
   * Starts a check by hand. "Checking" is shown before asking main, so a second click is ignored;
   * a result event that arrives before main's answer is kept (see applyCheckAnswer).
   */
  async function checkNow() {
    if (isChecking()) {
      return;
    }
    manualCheck = true;
    apply({ type: 'check', result: { status: 'started' } });
    try {
      const answer = await croffleInternal.window.checkForUpdates();
      checkStatus.value = applyCheckAnswer(checkStatus.value, answer);
    } catch (error) {
      apply({ type: 'error', message: error instanceof Error ? error.message : String(error) });
    }
    // An install that cannot check answers at once; no events follow.
    if (!isChecking()) {
      manualCheck = false;
    }
  }

  function init() {
    unsubscribers = [
      croffle.event.on(AppEventType.UPDATE_AVAILABLE, (payload) => {
        updateInfo.value = payload as UpdateInfo;
        updateError.value = null;
        isModalOpen.value = true;
        manualCheck = false;
        apply({ type: 'available', version: updateInfo.value.version });
      }),
      croffle.event.on(AppEventType.UPDATE_NOT_AVAILABLE, () => {
        apply({ type: 'not-available' });
        if (toastsLatest(manualCheck)) {
          toast(t('update.alreadyLatest'));
        }
        manualCheck = false;
      }),
      croffle.event.on(AppEventType.UPDATE_DOWNLOAD_PROGRESS, (payload) => {
        isDownloading.value = true;
        const { percent } = payload as { percent: number; transferred: number };
        downloadProgress.value = percent;
      }),
      croffle.event.on(AppEventType.UPDATE_DOWNLOADED, () => {
        isDownloading.value = false;
        downloadProgress.value = 0;
      }),
      croffle.event.on(AppEventType.UPDATE_ERROR, (payload) => {
        const err = payload as Error;
        const message = err?.message ?? t('update.errorFallback');
        isDownloading.value = false;
        downloadProgress.value = 0;
        isModalOpen.value = false;
        updateError.value = message;
        apply({ type: 'error', message });
        manualCheck = false;
        toast.error(t('update.failed'), { description: message });
      }),
    ];
  }

  function dispose() {
    unsubscribers.forEach((unsubscribe) => unsubscribe());
    unsubscribers = [];
  }

  function downloadNow() {
    updateError.value = null;
    isDownloading.value = true;
    croffle.event.emit(AppEventType.UPDATE_DOWNLOAD_NOW, null);
  }

  function downloadLater() {
    updateError.value = null;
    isModalOpen.value = false;
    isDownloading.value = true;
    downloadProgress.value = 0;
    croffle.event.emit(AppEventType.UPDATE_DOWNLOAD_LATER, null);
  }

  function skipUpdate() {
    croffle.event.emit(AppEventType.UPDATE_SKIP_THIS_VERSION, null);
    isDownloading.value = false;
    downloadProgress.value = 0;
    updateError.value = null;
    isModalOpen.value = false;
  }

  return {
    checkStatus,
    checkNow,
    isModalOpen,
    updateInfo,
    isDownloading,
    downloadProgress,
    updateError,
    init,
    dispose,
    downloadNow,
    downloadLater,
    skipUpdate,
  };
});
