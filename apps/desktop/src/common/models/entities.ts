/**
 * DB row shapes — must match drizzle `$inferSelect` exactly.
 * Public/API DTOs live alongside and are assembled via mappers.
 */

import type { SyncColumns } from './sync';

/** Flattens `T & SyncColumns` so `AssertEqual` can compare it with drizzle `$inferSelect`. */
type WithSync<T> = { [K in keyof (T & SyncColumns)]: (T & SyncColumns)[K] };

export type TagEntity = WithSync<{
  id: string;
  name: string;
  color: string;
  createdAt: Date;
  updatedAt: Date;
}>;

export type ScheduleEntity = WithSync<{
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
}>;

/** Schedule ↔ tag link. Has its own id so it can be tombstoned and synced as a row. */
export type ScheduleTagEntity = WithSync<{
  id: string;
  scheduleId: string;
  tagId: string;
}>;

/** Link between a local schedule and one external provider event (provider-neutral). */
export type ScheduleExternalLinkEntity = WithSync<{
  id: string;
  scheduleId: string;
  provider: string;
  accountId: string;
  externalId: string;
  externalEtag: string | null;
  externalUpdatedAtMs: number | null;
}>;

/** Override of one occurrence of a recurring schedule (key = original occurrence start). */
export type ScheduleExceptionEntity = WithSync<{
  id: string;
  scheduleId: string;
  occurrenceStart: Date;
  startDate: Date | null;
  endDate: Date | null;
  cancelled: boolean;
}>;

/** One reminder offset of a schedule, in minutes before the occurrence starts. */
export type ScheduleReminderEntity = WithSync<{
  id: string;
  scheduleId: string;
  minutes: number;
}>;

/** Persisted extension registry row (manifest body stays on disk). */
export type ExtensionInfoEntity = {
  id: string;
  name: string;
  version: string;
  author: string;
  description: string | null;
  enabled: boolean;
  main: string | null;
  installedAt: Date;
  updatedAt: Date;
};

export type ExtensionStorageEntity = {
  extensionId: string;
  key: string;
  value: string;
  updatedAt: Date;
};
