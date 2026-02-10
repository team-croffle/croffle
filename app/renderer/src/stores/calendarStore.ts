import { INITIAL_SCHEDULES } from '@/data/dummySchedule';
import type { Schedule } from 'croffle';
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

export const useCalendarStore = defineStore('calendar', () => {
  // 더미 데이터 사용
  const schedules = ref<Schedule[]>(INITIAL_SCHEDULES);

  // 데이터 변환(FullCalendar 이벤트 형식에 맞게)
  const events = computed(() => {
    return schedules.value.map((schedule) => ({
      id: schedule.id,
      title: schedule.title,
      start: schedule.startDate,
      end: schedule.endDate,
      allDay: schedule.isAllDay,
      backgroundColor: schedule.colorLabel,
      borderColor: schedule.colorLabel,
      textColor: '#FFFFFF',
      display: schedule.isAllDay ? 'auto' : 'block',
      extendedProps: {
        description: schedule.description,
        location: schedule.location,
        tags: schedule.tags,
        recurrenceRule: schedule.recurrenceRule,
      },
    }));
  });

  // Actions
  const addSchedule = (newSchedule: Schedule) => {
    schedules.value.push(newSchedule);
  };

  const removeSchedule = (id: string) => {
    schedules.value = schedules.value.filter((s) => s.id !== id);
  };

  return { schedules, events, addSchedule, removeSchedule };
});
