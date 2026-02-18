import { INITIAL_SCHEDULES } from '@/data/dummySchedule';
import type { Schedule } from 'croffle';
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import dayjs from 'dayjs';

export const useScheduleStore = defineStore('schedule', () => {
  // 더미 데이터 사용
  const schedules = ref<Schedule[]>(INITIAL_SCHEDULES);

  // 데이터 변환(FullCalendar 이벤트 형식에 맞게)
  const events = computed(() => {
    return schedules.value.map((schedule) => {
      let displayEndDate = schedule.endDate;

      // 종일 일정의 경우, FullCalendar는 end 날짜를 포함하지 않으므로 하루 더해줌
      if (schedule.isAllDay && schedule.endDate) {
        displayEndDate = dayjs(schedule.endDate).add(1, 'day').format('YYYY-MM-DD');
      }

      return {
        id: schedule.id,
        title: schedule.title,
        start: schedule.startDate,
        end: displayEndDate,
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
      };
    });
  });

  // Actions
  // 일정 추가
  const addSchedule = (newSchedule: Schedule) => {
    schedules.value.push(newSchedule); // createSchedules()에서 호출 예정
  };

  // 일정 조회
  const getScheduleById = (id: string) => {
    return schedules.value.find((s) => s.id === id); // getSchedules()에서 호출 예정
  };

  // 일정 수정
  const updateSchedule = (updatedSchedule: Schedule) => {
    const index = schedules.value.findIndex((s) => s.id === updatedSchedule.id); // updateSchedule(ScheduleData)에서 호출 예정
    if (index !== -1) {
      schedules.value[index] = updatedSchedule;
    }
  };

  // 일정 제거
  const deleteSchedule = (id: string) => {
    schedules.value = schedules.value.filter((s) => s.id !== id); // deleteSchedule()에서 호출 예정
  };

  return { schedules, events, addSchedule, getScheduleById, updateSchedule, deleteSchedule };
});
