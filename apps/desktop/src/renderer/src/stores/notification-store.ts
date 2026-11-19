import type { NotificationItem, NotificationList, Schedule } from '@croffledev/common';
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

import { croffleInternal } from '@/services/host-bridge';

import type { SelectedOccurrence } from './ui-store';

const EMPTY: NotificationList = { fired: [], upcoming: [], unread: 0 };
const REFRESH_MS = 60_000;

/** The bell's popover: reminders fired during this run and those due in the next 24 hours. */
export const useNotificationStore = defineStore('notifications', () => {
  const list = ref<NotificationList>(EMPTY);
  let unsubscribe: (() => void) | null = null;
  let timer: ReturnType<typeof setInterval> | null = null;

  const refresh = async () => {
    try {
      list.value = await croffleInternal.notifications.list();
    } catch {
      // Keep the last list; the bell is not worth an error toast.
    }
  };

  const init = () => {
    if (unsubscribe || timer) {
      return;
    }
    unsubscribe = croffleInternal.notifications.onChanged(() => {
      void refresh();
    });
    // Time alone moves reminders into the next 24 hours (and past ones out, after sleep).
    timer = setInterval(() => void refresh(), REFRESH_MS);
    void refresh();
  };

  const markAllRead = async () => {
    await croffleInternal.notifications.markAllRead();
    await refresh();
  };

  const unread = computed(() => list.value.unread);

  return { list, unread, init, refresh, markAllRead };
});

/**
 * The occurrence a reminder points at, for the schedule viewer. One-off schedules need none;
 * a recurring one gets its start, its end (moved occurrences keep their own) and whether it moved.
 */
export function occurrenceOf(
  item: NotificationItem,
  schedule: Schedule | undefined,
): SelectedOccurrence | undefined {
  if (!schedule?.recurrenceRule?.trim()) {
    return undefined;
  }
  const start = new Date(item.occurrenceStart);
  const moved = schedule.exceptions?.find(
    (e) => !e.cancelled && e.startDate && new Date(e.startDate).getTime() === item.occurrenceStart,
  );
  if (moved?.endDate) {
    return { start, end: new Date(moved.endDate), isOverride: true };
  }
  const duration = new Date(schedule.endDate).getTime() - new Date(schedule.startDate).getTime();
  return { start, end: new Date(start.getTime() + Math.max(duration, 0)), isOverride: false };
}
