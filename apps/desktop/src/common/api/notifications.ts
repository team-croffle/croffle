/** Main → app renderer: the notification list changed. Internal; not on the extension event bus. */
export const NOTIFICATIONS_CHANNEL = 'croffle:notifications:changed';

/** A reminder in the bell's popover. Times are epoch milliseconds (they cross IPC as numbers). */
export type NotificationItem = {
  /** The reminder's dedup key: one per schedule occurrence and offset. */
  id: string;
  scheduleId: string;
  title: string;
  occurrenceStart: number;
  remindAt: number;
  reminderMinutes: number;
  isAllDay: boolean;
};

/** A reminder that fired while the app was running (kept in memory only). */
export type FiredNotification = NotificationItem & {
  firedAt: number;
  read: boolean;
};

export type NotificationList = {
  /** Fired during this run, newest first. */
  fired: FiredNotification[];
  /** Due within the next 24 hours, soonest first. */
  upcoming: NotificationItem[];
  unread: number;
};

/** Host-only notification list (`croffleInternal.notifications`). Never part of the public types. */
export interface InternalNotificationsApi {
  list(): Promise<NotificationList>;
  markAllRead(): Promise<void>;
  /** Subscribes to changes; returns the unsubscribe function. */
  onChanged(callback: () => void): () => void;
}
