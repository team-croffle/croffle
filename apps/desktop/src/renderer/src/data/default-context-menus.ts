import type { FeatureContextMenu } from '@croffledev/common';
import dayjs from 'dayjs';
import { toast } from 'vue-sonner';

import { i18n } from '@/i18n';
import { useScheduleStore } from '@/stores/schedule-store';

import { useUiStore } from '../stores/ui-store';

const isDateElement = (target: HTMLElement | null) => {
  return !!target?.closest('[data-date], .fc-daygrid-day') && !target?.closest('.fc-event');
};

const getClickedDateFromTarget = (target: HTMLElement): string | null => {
  const dayCell = target.closest('.fc-daygrid-day');
  return dayCell ? dayCell.getAttribute('data-date') : null;
};

function t(key: string) {
  return String(i18n.global.t(key));
}

export const defaultMenus: FeatureContextMenu[] = [
  {
    id: 'add-schedule',
    label: 'contextMenu.addSchedule',
    action: (targetElement: HTMLElement | null) => {
      if (!targetElement) {
        return;
      }

      const uiStore = useUiStore();
      const date = getClickedDateFromTarget(targetElement);
      uiStore.selectedDate = date ?? dayjs().format('YYYY-MM-DD');
      uiStore.openScheduleModal('add');
    },
    condition: isDateElement,
    targetView: ['calendar'],
  },
  {
    id: 'view-schedule',
    label: 'contextMenu.viewDay',
    action: (targetElement: HTMLElement | null) => {
      if (!targetElement) {
        return;
      }
      const date = getClickedDateFromTarget(targetElement);
      if (!date) {
        return;
      }
      useUiStore().openRightSidebarWithDate(date);
    },
    condition: isDateElement,
    targetView: ['calendar'],
  },
  {
    id: 'edit-schedule',
    label: 'contextMenu.editSchedule',
    action: (targetElement: HTMLElement | null) => {
      if (!targetElement) {
        return;
      }
      const eventId = targetElement.closest('.fc-event')?.getAttribute('data-event-id');
      if (!eventId || eventId === 'undefined') {
        return;
      }
      useUiStore().openScheduleModal('edit', eventId);
    },
    condition: (target) => !!target?.closest('.fc-event'),
    targetView: ['calendar'],
  },
  {
    id: 'delete-schedule',
    label: 'contextMenu.deleteSchedule',
    action: async (targetElement: HTMLElement | null) => {
      if (!targetElement) {
        return;
      }
      const eventId = targetElement?.closest('.fc-event')?.getAttribute('data-event-id');
      if (!eventId || eventId === 'undefined') {
        return;
      }

      const confirmed = await useUiStore().openConfirm({
        title: t('contextMenu.deleteSchedule'),
        description: t('contextMenu.deleteConfirm'),
        confirmLabel: t('common.delete'),
        cancelLabel: t('common.cancel'),
        confirmVariant: 'destructive',
      });
      if (!confirmed) {
        return;
      }

      try {
        const isSuccess = await useScheduleStore().removeScheduleById(eventId);
        if (isSuccess) {
          toast.success(t('contextMenu.deleteSuccess'));
        } else {
          toast.error(t('contextMenu.deleteFailed'));
        }
      } catch {
        toast.error(t('contextMenu.deleteError'));
      }
    },
    condition: (target) => !!target?.closest('.fc-event'),
    targetView: ['calendar'],
  },
];
