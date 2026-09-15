/**
 * DB row shapes — must match drizzle `$inferSelect` exactly.
 * Public/API DTOs live alongside and are assembled via mappers.
 */

export type TagEntity = {
  id: string;
  name: string;
  color: string;
};

export type ScheduleEntity = {
  id: string;
  title: string;
  description: string | null;
  location: string | null;
  startDate: Date;
  endDate: Date;
  isAllDay: boolean;
  recurrenceRule: string | null;
  colorLabel: string;
  priority: 'low' | 'medium' | 'high';
  /** true = follow notifications.defaultReminderMinutes; false = use schedule_reminder rows */
  useDefaultReminder: boolean;
  createdAt: Date;
  updatedAt: Date;
};

/** One reminder offset of a schedule, in minutes before the occurrence starts. */
export type ScheduleReminderEntity = {
  id: string;
  scheduleId: string;
  minutes: number;
};

/** Persisted extension registry row (manifest body stays on disk). */
export type ExtensionInfoEntity = {
  id: string;
  name: string;
  version: string;
  author: string;
  description: string | null;
  enabled: boolean;
  main: string | null;
};

export type ExtensionStorageEntity = {
  extensionId: string;
  key: string;
  value: string;
  updatedAt: Date;
};
