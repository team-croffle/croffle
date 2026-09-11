import type { Schedule } from '@croffledev/common';
import { toFullCalendarRRule } from '@croffledev/common';
import type { EventInput } from '@fullcalendar/core';
import dayjs from 'dayjs';
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { toast } from 'vue-sonner';

import { i18n } from '@/i18n';

function toEventDuration(schedule: Schedule): EventInput['duration'] {
  const start = dayjs(schedule.startDate);
  const end = dayjs(schedule.endDate);

  if (schedule.isAllDay) {
    const startDay = start.startOf('day');
    const endExclusive = end.startOf('day').add(1, 'day');
    const days = Math.max(endExclusive.diff(startDay, 'day'), 1);
    return { days };
  }

  const ms = Math.max(end.diff(start), 5 * 60 * 1000);
  return { milliseconds: ms };
}

function toCalendarEvent(schedule: Schedule): EventInput {
  const base: EventInput = {
    id: schedule.id,
    title: schedule.title,
    allDay: schedule.isAllDay,
    backgroundColor: schedule.colorLabel,
    borderColor: schedule.colorLabel,
    textColor: '#FFFFFF',
    display: schedule.isAllDay ? 'auto' : 'block',
    extendedProps: {
      scheduleId: schedule.id,
      description: schedule.description,
      location: schedule.location,
      tags: schedule.tags,
      recurrenceRule: schedule.recurrenceRule,
      priority: schedule.priority,
    },
  };

  if (schedule.recurrenceRule?.trim()) {
    const rrule = toFullCalendarRRule(schedule.recurrenceRule, schedule.startDate, {
      allDay: schedule.isAllDay,
    });
    if (rrule) {
      return {
        ...base,
        rrule,
        duration: toEventDuration(schedule),
      };
    }
  }

  let displayEndDate: Date | string = schedule.endDate;
  if (schedule.isAllDay && schedule.endDate) {
    displayEndDate = dayjs(schedule.endDate).add(1, 'day').toDate();
  }

  return {
    ...base,
    start: schedule.startDate,
    end: displayEndDate,
  };
}

export const useScheduleStore = defineStore('schedule', () => {
  const schedules = ref<Schedule[]>([]);

  const events = computed(() => schedules.value.map(toCalendarEvent));

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
    } catch (error) {
      toast.error(String(i18n.global.t('schedule.loadFailed', { error: JSON.stringify(error) })));
    }
  };

  return {
    schedules,
    events,
    getScheduleById,
    createSchedule,
    updateScheduleById,
    removeScheduleById,
    loadSchedules,
  };
});
