import { shiftRecurrenceRule } from '@croffledev/common';
import type { EventDropArg } from '@fullcalendar/core';
import dayjs from 'dayjs';
import { toast } from 'vue-sonner';

import { i18n } from '@/i18n';
import { useChoiceStore } from '@/stores/choice-store';
import { useScheduleStore } from '@/stores/schedule-store';

type RecurringChoice = 'all' | 'this';

/** Same wall-clock time, `days` calendar days later. */
const addDays = (date: Date | string, days: number): Date => dayjs(date).add(days, 'day').toDate();

/**
 * Calendar drag in month / year views: only the date changes, time and length stay.
 * Recurring schedules ask whether to move the whole series or just this occurrence.
 */
export function useScheduleDrag() {
  const scheduleStore = useScheduleStore();
  const choiceStore = useChoiceStore();
  const t = (key: string) => String(i18n.global.t(key));

  const askRecurring = (canMoveAll: boolean) =>
    choiceStore.openChoice<RecurringChoice>({
      title: t('schedule.drag.recurringTitle'),
      description: t('schedule.drag.recurringDescription'),
      hint: canMoveAll ? undefined : t('schedule.drag.ruleNotShiftable'),
      choices: [
        { value: 'this', label: t('schedule.drag.thisOccurrence'), variant: 'outline' },
        { value: 'all', label: t('schedule.drag.allOccurrences'), disabled: !canMoveAll },
      ],
    });

  const applyDrop = async (info: EventDropArg): Promise<boolean> => {
    const { event, oldEvent } = info;
    if (!event.start || !oldEvent.start) {
      return false;
    }
    const days = dayjs(event.start)
      .startOf('day')
      .diff(dayjs(oldEvent.start).startOf('day'), 'day');
    if (days === 0) {
      return false;
    }

    const scheduleId = (event.extendedProps.scheduleId as string | undefined) ?? event.id;
    const schedule = scheduleStore.getScheduleById(scheduleId);
    if (!schedule) {
      return false;
    }

    // A moved occurrence keeps its original key; only its override moves again.
    if (event.extendedProps.isOverride) {
      await scheduleStore.moveOccurrence(
        scheduleId,
        event.extendedProps.occurrenceStart as Date,
        addDays(oldEvent.start, days),
      );
      return true;
    }

    const shiftSeries = () => ({
      startDate: addDays(schedule.startDate, days),
      endDate: addDays(schedule.endDate, days),
    });

    if (!schedule.recurrenceRule?.trim()) {
      await scheduleStore.updateScheduleById(scheduleId, shiftSeries());
      return true;
    }

    const shiftedRule = shiftRecurrenceRule(schedule.recurrenceRule, days);
    const choice = await askRecurring(shiftedRule !== null);
    if (choice === 'this') {
      await scheduleStore.moveOccurrence(scheduleId, oldEvent.start, addDays(oldEvent.start, days));
      return true;
    }
    if (choice === 'all' && shiftedRule !== null) {
      // Exceptions follow the series start on the main side.
      await scheduleStore.updateScheduleById(scheduleId, {
        ...shiftSeries(),
        recurrenceRule: shiftedRule,
      });
      return true;
    }
    return false;
  };

  const handleEventDrop = async (info: EventDropArg) => {
    try {
      const saved = await applyDrop(info);
      if (!saved) {
        info.revert();
      }
    } catch (error) {
      info.revert();
      toast.error(
        String(
          i18n.global.t('schedule.drag.moveFailed', {
            error: error instanceof Error ? error.message : String(error),
          }),
        ),
      );
    }
  };

  return { handleEventDrop };
}
