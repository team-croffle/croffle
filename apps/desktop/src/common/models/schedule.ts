import type { Tag } from './tag';

/** App / IPC schedule DTO — dates match drizzle timestamp columns. */
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
  priority: 'low' | 'medium' | 'high';
  /**
   * Reminder offsets in minutes before the schedule starts, ascending.
   * `null` = follow the app-wide notifications.defaultReminderMinutes.
   * `[]` = no reminder at all.
   */
  reminders: number[] | null;
  /**
   * @deprecated Mirror of the first entry in `reminders`, kept for extensions
   * written against 1.2.0. Removed in 1.3 — read `reminders` instead.
   */
  reminderMinutes?: number | null;
  tags: Tag[];
  createdAt: Date;
  updatedAt: Date;
};
