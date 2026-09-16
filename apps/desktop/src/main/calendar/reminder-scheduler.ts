import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

import {
  AppEventType,
  buildReminderCandidates,
  formatReminderBody,
  legacyReminderDedupKey,
  reminderDedupKey,
  type ReminderCandidate,
} from '@croffledev/common';
import { app } from 'electron';

import { eventService } from '../event-bus/event-service';
import { logger } from '../logger';
import { settingService } from '../setting/setting-service';
import { showNotification } from '../window/os-notification';
import { windowService } from '../window/window-service';
import { getSchedules } from './schedule-service';

const HORIZON_DAYS = 7;
const SAFETY_POLL_MS = 30_000;
const GRACE_MS = 2 * 60_000;
/** setTimeout delay clamp — re-arm periodically for long waits / sleep recovery */
const MAX_TIMER_MS = 60 * 60_000;
const FIRED_RETENTION_MS = (HORIZON_DAYS + 1) * 24 * 60 * 60_000;

type FiredStore = {
  keys: string[];
};

class ReminderScheduler {
  private started = false;
  private candidates: ReminderCandidate[] = [];
  private fired = new Set<string>();
  private nextTimer: ReturnType<typeof setTimeout> | null = null;
  private pollTimer: ReturnType<typeof setInterval> | null = null;
  private readonly firedPath: string;
  private unsubscribers: (() => void)[] = [];

  constructor() {
    this.firedPath = path.join(app.getPath('userData'), 'reminder-fired.json');
  }

  public async start(): Promise<void> {
    if (this.started) {
      return;
    }
    this.started = true;
    this.loadFired();
    this.registerListeners();
    this.pollTimer = setInterval(() => {
      void this.tick();
    }, SAFETY_POLL_MS);
    await this.rebuild();
    logger.info('ReminderScheduler', 'Started');
  }

  public stop(): void {
    if (!this.started) {
      return;
    }
    this.started = false;
    this.clearNextTimer();
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
    for (const off of this.unsubscribers) {
      off();
    }
    this.unsubscribers = [];
    this.saveFired();
    logger.info('ReminderScheduler', 'Stopped');
  }

  public async rebuild(): Promise<void> {
    if (!this.started) {
      return;
    }

    const settings = settingService.get();
    if (!settings.notifications.enabled) {
      this.candidates = [];
      this.clearNextTimer();
      logger.debug('ReminderScheduler', 'Notifications disabled — queue cleared');
      return;
    }

    const now = new Date();
    const horizonEnd = new Date(now.getTime() + HORIZON_DAYS * 24 * 60 * 60_000);
    const defaultMinutes = settings.notifications.defaultReminderMinutes;

    // Pull schedules that may fire in the horizon (include past buffer for remindAt lag)
    const queryStart = new Date(now.getTime() - Math.max(defaultMinutes, 60) * 60_000 - GRACE_MS);
    const rows = await getSchedules({ start: queryStart, end: horizonEnd });

    this.candidates = buildReminderCandidates(rows, {
      now,
      horizonEnd,
      defaultReminderMinutes: defaultMinutes,
    });

    this.pruneFired(now);
    await this.tick();
    this.armNextTimer();
    logger.debug('ReminderScheduler', `Queue rebuilt: ${this.candidates.length} candidate(s)`);
  }

  private registerListeners(): void {
    const onRebuild = () => {
      void this.rebuild();
    };

    const events = [
      AppEventType.SCHEDULE_CREATE,
      AppEventType.SCHEDULE_UPDATE,
      AppEventType.SCHEDULE_DELETE,
      AppEventType.SETTINGS_UPDATE,
    ] as const;

    for (const eventName of events) {
      const listener = () => onRebuild();
      eventService.on(eventName, listener);
      this.unsubscribers.push(() => eventService.off(eventName, listener));
    }
  }

  private async tick(): Promise<void> {
    if (!this.started) {
      return;
    }

    const settings = settingService.get();
    if (!settings.notifications.enabled) {
      return;
    }

    const now = Date.now();
    let dirty = false;

    for (const candidate of this.candidates) {
      const key = this.keyOf(candidate);
      if (this.hasFired(candidate)) {
        continue;
      }

      const remindAt = candidate.remindAt.getTime();
      if (remindAt > now) {
        break; // sorted ascending
      }

      const age = now - remindAt;
      if (age > GRACE_MS) {
        // Too old to catch up — mark skipped so we don't retry forever
        this.fired.add(key);
        dirty = true;
        continue;
      }

      this.fire(candidate, key);
      dirty = true;
    }

    if (dirty) {
      this.saveFired();
      this.armNextTimer();
    }
  }

  private fire(candidate: ReminderCandidate, key: string): void {
    this.fired.add(key);
    try {
      const language = settingService.get().general.language;
      showNotification(candidate.title, formatReminderBody(candidate, language), () => {
        windowService.showWindow();
      });
      logger.info(
        'ReminderScheduler',
        `Fired reminder for ${candidate.scheduleId} @ ${candidate.occurrenceStart.toISOString()} (${candidate.reminderMinutes}m before)`,
      );
    } catch (error) {
      logger.error('ReminderScheduler', 'Failed to show notification', error);
    }
  }

  private armNextTimer(): void {
    this.clearNextTimer();

    const settings = settingService.get();
    if (!settings.notifications.enabled) {
      return;
    }

    const now = Date.now();
    const next = this.candidates.find((c) => !this.hasFired(c) && c.remindAt.getTime() > now);

    if (!next) {
      return;
    }

    const delay = Math.min(Math.max(next.remindAt.getTime() - now, 0), MAX_TIMER_MS);
    this.nextTimer = setTimeout(() => {
      void this.tick().then(() => this.armNextTimer());
    }, delay);
  }

  private keyOf(candidate: ReminderCandidate): string {
    return reminderDedupKey(
      candidate.scheduleId,
      candidate.occurrenceStart,
      candidate.reminderMinutes,
    );
  }

  /** New key, or the pre-1.2.1 key persisted before offsets were part of the key. */
  private hasFired(candidate: ReminderCandidate): boolean {
    return (
      this.fired.has(this.keyOf(candidate)) ||
      this.fired.has(legacyReminderDedupKey(candidate.scheduleId, candidate.occurrenceStart))
    );
  }

  private clearNextTimer(): void {
    if (this.nextTimer) {
      clearTimeout(this.nextTimer);
      this.nextTimer = null;
    }
  }

  private loadFired(): void {
    try {
      if (!existsSync(this.firedPath)) {
        return;
      }
      const raw = JSON.parse(readFileSync(this.firedPath, 'utf8')) as FiredStore;
      if (Array.isArray(raw.keys)) {
        this.fired = new Set(raw.keys.filter((k) => typeof k === 'string'));
      }
    } catch (error) {
      logger.warn('ReminderScheduler', 'Failed to load fired keys', error);
      this.fired = new Set();
    }
  }

  private saveFired(): void {
    try {
      const dir = path.dirname(this.firedPath);
      if (!existsSync(dir)) {
        mkdirSync(dir, { recursive: true });
      }
      const payload: FiredStore = { keys: [...this.fired] };
      writeFileSync(this.firedPath, JSON.stringify(payload), 'utf8');
    } catch (error) {
      logger.warn('ReminderScheduler', 'Failed to persist fired keys', error);
    }
  }

  private pruneFired(now: Date): void {
    const cutoff = now.getTime() - FIRED_RETENTION_MS;
    this.fired.forEach((key) => {
      const ts = Number(key.split('|')[1]);
      if (!Number.isFinite(ts) || ts < cutoff) {
        this.fired.delete(key);
      }
    });
  }
}

export const reminderScheduler = new ReminderScheduler();
