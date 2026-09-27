import type { Schedule } from '@croffledev/common';
import dayjs from 'dayjs';
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { toast } from 'vue-sonner';

import { i18n } from '@/i18n';
import { toCalendarEvents } from '@/utils/schedule-events';

export const useScheduleStore = defineStore('schedule', () => {
  const schedules = ref<Schedule[]>([]);
  /** 마지막으로 불러온 범위. 가져오기 등 외부 변경 뒤 같은 범위로 다시 읽을 때 쓴다. */
  let lastRange: { start: string; end: string } | null = null;

  const events = computed(() => schedules.value.flatMap(toCalendarEvents));

  const getScheduleById = (id: string) => {
    return schedules.value.find((s) => s.id === id);
  };

  const upsertSchedule = (schedule: Schedule) => {
    const index = schedules.value.findIndex((s) => s.id === schedule.id);
    if (index === -1) {
      schedules.value.push(schedule);
      return;
    }
    schedules.value[index] = schedule;
  };

  const createSchedule = async (payload: Partial<Schedule>) => {
    const created = await croffle.calendar.schedules.create(payload);
    upsertSchedule(created);
    return created;
  };

  const updateScheduleById = async (id: string, payload: Partial<Schedule>) => {
    const updated = await croffle.calendar.schedules.update(id, payload);
    upsertSchedule(updated);
    return updated;
  };

  /** Move one occurrence of a recurring schedule (series unchanged). */
  const moveOccurrence = async (id: string, occurrenceStart: Date, newStart: Date) => {
    const updated = await croffle.calendar.schedules.moveOccurrence(id, occurrenceStart, newStart);
    upsertSchedule(updated);
    return updated;
  };

  const removeScheduleById = async (id: string) => {
    const ok = await croffle.calendar.schedules.remove(id);
    if (ok) {
      schedules.value = schedules.value.filter((s) => s.id !== id);
    }
    return ok;
  };

  const loadSchedules = async (startDate?: string, endDate?: string) => {
    try {
      const now = dayjs();
      const start = startDate || now.subtract(1, 'month').startOf('month').toISOString();
      const end = endDate || now.add(1, 'month').endOf('month').toISOString();

      const result = await croffle.calendar.schedules.getAll({ start, end });
      schedules.value = result;
      lastRange = { start, end };
    } catch (error) {
      toast.error(String(i18n.global.t('schedule.loadFailed', { error: JSON.stringify(error) })));
    }
  };

  /** 마지막 범위로 다시 불러온다 (없으면 기본 범위). */
  const reload = async () => {
    if (lastRange) {
      await loadSchedules(lastRange.start, lastRange.end);
      return;
    }
    await loadSchedules();
  };

  return {
    schedules,
    events,
    getScheduleById,
    createSchedule,
    updateScheduleById,
    moveOccurrence,
    removeScheduleById,
    loadSchedules,
    reload,
  };
});
