import {
  NOTIFICATIONS_CHANNEL,
  type NotificationItem,
  type NotificationList,
} from '@croffledev/common';
import { BrowserWindow } from 'electron';

import { NotificationLog } from './notification-list';

/** Where pending reminders come from (the reminder scheduler registers itself). */
export type PendingReminderSource = () => NotificationItem[];

/** The bell's list for the app window: fired reminders of this run and the next 24 hours. */
class NotificationService {
  private readonly log = new NotificationLog();
  private pending: PendingReminderSource = () => [];

  public setPendingSource(source: PendingReminderSource): void {
    this.pending = source;
  }

  public recordFired(item: NotificationItem): void {
    this.log.add(item, Date.now());
    this.notifyChanged();
  }

  public list(): NotificationList {
    return this.log.list(this.pending(), Date.now(), () => false);
  }

  public markAllRead(): void {
    if (this.log.markAllRead()) {
      this.notifyChanged();
    }
  }

  /** Tells the app window to read the list again (pending reminders changed or one fired). */
  public notifyChanged(): void {
    for (const win of BrowserWindow.getAllWindows()) {
      if (!win.isDestroyed()) {
        win.webContents.send(NOTIFICATIONS_CHANNEL);
      }
    }
  }
}

export const notificationService = new NotificationService();
