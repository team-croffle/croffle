import type {
  FiredNotification,
  NotificationItem,
  NotificationList,
} from '../../common/api/notifications';

/** How many fired reminders the popover keeps (memory only; gone when the app quits). */
export const FIRED_LIMIT = 50;
/** How far ahead "upcoming" looks. */
export const UPCOMING_WINDOW_MS = 24 * 60 * 60 * 1000;

/**
 * The bell's list. Pure: the scheduler feeds it, IPC reads it, tests drive it without Electron.
 * Reading never marks anything read; only `markAllRead` does.
 */
export class NotificationLog {
  private fired: FiredNotification[] = [];

  public add(item: NotificationItem, firedAt: number): void {
    this.fired = [{ ...item, firedAt, read: false }, ...this.fired.filter((f) => f.id !== item.id)]
      .toSorted((a, b) => b.firedAt - a.firedAt)
      .slice(0, FIRED_LIMIT);
  }

  public markAllRead(): boolean {
    const changed = this.fired.some((f) => !f.read);
    this.fired = this.fired.map((f) => (f.read ? f : { ...f, read: true }));
    return changed;
  }

  /**
   * The list as the popover shows it. `candidates` are the scheduler's pending reminders;
   * `hasFired` tells the ones already shown (or skipped) apart.
   */
  public list(
    candidates: NotificationItem[],
    now: number,
    hasFired: (item: NotificationItem) => boolean,
  ): NotificationList {
    const upcoming = candidates
      .filter((c) => c.remindAt > now && c.remindAt <= now + UPCOMING_WINDOW_MS && !hasFired(c))
      .toSorted((a, b) => a.remindAt - b.remindAt);
    return {
      fired: this.fired.map((f) => ({ ...f })),
      upcoming,
      unread: this.fired.filter((f) => !f.read).length,
    };
  }
}
