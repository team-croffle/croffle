import type { Tag } from './tag.js';

/**
 * Override of one occurrence of a recurring schedule, keyed by the occurrence's
 * original start. `startDate`/`endDate` null means "not moved"; `cancelled` hides it.
 */
export type ScheduleException = {
  /** Original start of the occurrence this entry overrides (all-day: local midnight). */
  occurrenceStart: Date;
  /** New start of the occurrence, or null when only `cancelled` applies. */
  startDate: Date | null;
  /** New end of the occurrence, or null when only `cancelled` applies. */
  endDate: Date | null;
  /** true = this occurrence does not happen. */
  cancelled: boolean;
};

export type Schedule = {
  id: string;
  title: string;
  description: string;
  location: string;
  startDate: Date;
  endDate: Date;
  isAllDay: boolean;
  recurrenceRule?: string;
  colorLabel: string;
  tags: Tag[];
  priority: 'low' | 'medium' | 'high';
  /**
   * Reminder offsets in minutes before the schedule starts, ascending.
   * `null` = follow the app-wide notifications.defaultReminderMinutes.
   * `[]` = no reminder at all.
   * (The 1.2.0 single-value `reminderMinutes` mirror was removed in types 2.0.)
   */
  reminders: number[] | null;
  /**
   * Per-occurrence overrides of a recurring schedule (empty or missing for one-off schedules).
   * Extensions that expand `recurrenceRule` themselves must apply these: skip each
   * `occurrenceStart`, and add `startDate`..`endDate` unless `cancelled`.
   */
  exceptions?: ScheduleException[];
  /**
   * `'todo'` = a To-Do (1.3.3): `endDate` is the deadline and `startDate` equals it; a To-Do does
   * not repeat. Missing = `'event'`.
   */
  kind?: 'event' | 'todo';
  /** When a To-Do was done; `null` or missing = not done. Always empty for an event. */
  completedAt?: Date | null;
  createdAt: Date;
  updatedAt: Date;
};
