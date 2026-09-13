import dayjs from 'dayjs';
import { defineStore } from 'pinia';
import { ref } from 'vue';

export type ConfirmDialogOptions = {
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  confirmVariant?: 'default' | 'destructive';
};

/** 일정 모달 모드. 기존 일정은 기본 `view`(조회)로 열고, 버튼으로 `edit`으로 전환한다. */
export type ScheduleModalMode = 'add' | 'edit' | 'view';

export const useUiStore = defineStore('ui', () => {
  const leftSidebarOpen = ref(true);
  const rightSidebarOpen = ref(false);
  // 시작 직후 우측 사이드바가 오늘 일정을 보여주도록 오늘(로컬)로 초기화
  const selectedDate = ref<string | null>(dayjs().format('YYYY-MM-DD'));
  const isScheduleModalOpen = ref(false);
  const scheduleModalMode = ref<ScheduleModalMode>('add');
  const selectedScheduleId = ref<string | null>(null);

  const isConfirmModalOpen = ref(false);
  const confirmTitle = ref('');
  const confirmDescription = ref('');
  const confirmConfirmLabel = ref('');
  const confirmCancelLabel = ref('');
  const confirmVariant = ref<'default' | 'destructive'>('default');
  let confirmResolver: ((value: boolean) => void) | null = null;

  // 사이드바 토글 액션
  const toggleLeftSidebar = () => {
    leftSidebarOpen.value = !leftSidebarOpen.value;
  };

  const toggleRightSidebar = () => {
    rightSidebarOpen.value = !rightSidebarOpen.value;
  };

  const openRightSidebarWithDate = (date: string) => {
    selectedDate.value = dayjs(date).format('YYYY-MM-DD');
    rightSidebarOpen.value = true;
  };

  const openScheduleModal = (mode: ScheduleModalMode = 'add', scheduleId?: string) => {
    if (mode !== 'add' && !scheduleId) {
      return;
    }
    scheduleModalMode.value = mode;
    selectedScheduleId.value = scheduleId ?? null;
    isScheduleModalOpen.value = true;
  };

  /** 열린 모달의 모드만 바꾼다 (view ↔ edit). 닫지 않으므로 선택된 일정은 유지된다. */
  const setScheduleModalMode = (mode: Exclude<ScheduleModalMode, 'add'>) => {
    if (!isScheduleModalOpen.value || !selectedScheduleId.value) {
      return;
    }
    scheduleModalMode.value = mode;
  };

  const closeScheduleModal = () => {
    isScheduleModalOpen.value = false;
    scheduleModalMode.value = 'add';
    selectedScheduleId.value = null;
  };

  const resolveConfirm = (value: boolean) => {
    if (!confirmResolver) {
      isConfirmModalOpen.value = false;
      return;
    }
    const resolve = confirmResolver;
    confirmResolver = null;
    isConfirmModalOpen.value = false;
    resolve(value);
  };

  const openConfirm = (options: ConfirmDialogOptions): Promise<boolean> => {
    if (confirmResolver) {
      confirmResolver(false);
      confirmResolver = null;
    }

    confirmTitle.value = options.title;
    confirmDescription.value = options.description;
    confirmConfirmLabel.value = options.confirmLabel ?? '';
    confirmCancelLabel.value = options.cancelLabel ?? '';
    confirmVariant.value = options.confirmVariant ?? 'default';
    isConfirmModalOpen.value = true;

    return new Promise<boolean>((resolve) => {
      confirmResolver = resolve;
    });
  };

  return {
    leftSidebarOpen,
    rightSidebarOpen,
    selectedDate,
    isScheduleModalOpen,
    isConfirmModalOpen,
    confirmTitle,
    confirmDescription,
    confirmConfirmLabel,
    confirmCancelLabel,
    confirmVariant,
    toggleLeftSidebar,
    toggleRightSidebar,
    openRightSidebarWithDate,
    openScheduleModal,
    setScheduleModalMode,
    closeScheduleModal,
    openConfirm,
    resolveConfirm,
    scheduleModalMode,
    selectedScheduleId,
  };
});
