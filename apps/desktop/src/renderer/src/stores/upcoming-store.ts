import {
  selectUpcoming,
  UPCOMING_DAYS,
  type Schedule,
  type UpcomingOccurrence,
} from '@croffledev/common';
import dayjs from 'dayjs';
import { defineStore } from 'pinia';
import { computed, ref, watch } from 'vue';

import { occurrencesOnDate } from '@/utils/schedule-occurrences';

import { useScheduleStore } from './schedule-store';

/** How often "now" moves on, so a schedule that started leaves the list. */
const TICK_MS = 60_000;

/**
 * Today's and the next days' schedules, independent of the month the calendar shows.
 * Feeds the right sidebar's "upcoming" list and the left sidebar's "오늘 일정 N개".
 * Reads with the same range query the calendar uses; reloads when the calendar's schedules change.
 */
export const useUpcomingStore = defineStore('upcoming', () => {
  const scheduleStore = useScheduleStore();
  const schedules = ref<Schedule[]>([]);
  const now = ref(new Date());
  let timer: ReturnType<typeof setInterval> | null = null;
  let loadedDay = '';

  const load = async () => {
    const start = dayjs().startOf('day');
    loadedDay = start.format('YYYY-MM-DD');
    try {
      schedules.value = await croffle.calendar.schedules.getAll({
        start: start.toISOString(),
        end: start.add(UPCOMING_DAYS + 1, 'day').toISOString(),
      });
    } catch {
      // The calendar reports load errors; the sidebar keeps the last list.
    }
  };

  const upcoming = computed<UpcomingOccurrence[]>(() => selectUpcoming(schedules.value, now.value));

  const todayCount = computed(
    () => schedules.value.flatMap((schedule) => occurrencesOnDate(schedule, now.value)).length,
  );

  const start = () => {
    if (timer) {
      return;
    }
    void load();
    timer = setInterval(() => {
      now.value = new Date();
      if (dayjs(now.value).format('YYYY-MM-DD') !== loadedDay) {
        void load();
      }
    }, TICK_MS);
  };

  // Any create, update, delete or import goes through the calendar's store.
  watch(
    () => scheduleStore.schedules,
    () => {
      now.value = new Date();
      void load();
    },
    { deep: true },
  );

  return { upcoming, todayCount, start, load };
});
