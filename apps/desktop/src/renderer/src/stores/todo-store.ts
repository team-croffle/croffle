import type { Schedule } from '@croffledev/common';
import { defineStore } from 'pinia';
import { ref, watch } from 'vue';
import { toast } from 'vue-sonner';

import { i18n } from '@/i18n';
import { croffleInternal } from '@/services/host-bridge';

import { useScheduleStore } from './schedule-store';

/**
 * The open To-Dos of the right sidebar (1.3.3): every To-Do not done yet, overdue ones too,
 * earliest deadline first. Not limited to the month the calendar shows, so it has its own query;
 * it reloads whenever the calendar's schedules change (create, update, delete, import).
 */
export const useTodoStore = defineStore('todo', () => {
  const scheduleStore = useScheduleStore();
  const openTodos = ref<Schedule[]>([]);
  /** 'loading' until the first answer; 'error' when the last read failed. */
  const status = ref<'loading' | 'ready' | 'error'>('loading');
  let started = false;
  /** Number of the latest read; an answer to an older one is dropped (reads can overlap). */
  let latest = 0;

  const load = async () => {
    const request = ++latest;
    try {
      const todos = await croffleInternal.calendar.schedules.listOpenTodos();
      if (request === latest) {
        openTodos.value = todos;
        status.value = 'ready';
      }
    } catch {
      // Keep the last list; the next schedule change reloads it.
      if (request === latest) {
        status.value = 'error';
      }
    }
  };

  /** Marks a To-Do done (now) or not done. Goes through the calendar's store like any edit. */
  const setCompleted = async (schedule: Schedule, done: boolean) => {
    try {
      await scheduleStore.updateScheduleById(schedule.id, {
        completedAt: done ? new Date() : null,
      });
    } catch (error) {
      toast.error(String(i18n.global.t('schedule.saveFailed', { error: JSON.stringify(error) })));
    }
  };

  const start = () => {
    if (started) {
      return;
    }
    started = true;
    void load();
  };

  watch(
    () => scheduleStore.schedules,
    () => {
      void load();
    },
    { deep: true },
  );

  return { openTodos, status, start, load, setCompleted };
});
