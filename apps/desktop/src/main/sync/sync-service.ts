import {
  AppEventType,
  SYNC_STATUS_CHANNEL,
  type SyncErrorKind,
  type SyncStatus,
} from '@croffledev/common';
import { BrowserWindow } from 'electron';

import { reminderScheduler } from '../calendar/reminder-scheduler';
import { eventService } from '../event-bus/event-service';
import { logger } from '../logger';
import { backfillOutbox } from './backfill';
import { countFailed, countPending } from './outbox';
import { pull, bootstrap } from './pull';
import { push } from './push';
import {
  getServer,
  isTokenVolatile,
  loadToken,
  updateServerState,
  setEnabled as storeEnabled,
} from './sync-config';
import { checkServer, configureTls, SyncError } from './transport';

const PERIOD_MS = 5 * 60_000;
const DEBOUNCE_MS = 5_000;
const MAX_BACKOFF_MS = 30 * 60_000;

/** Local edits that should be pushed soon. The outbox already has them; this only times the push. */
const LOCAL_CHANGE_EVENTS = [
  AppEventType.SCHEDULE_CREATE,
  AppEventType.SCHEDULE_UPDATE,
  AppEventType.SCHEDULE_DELETE,
  AppEventType.SCHEDULE_IMPORT_FROM_FILE,
  AppEventType.TAG_CREATE,
  AppEventType.TAG_UPDATE,
  AppEventType.TAG_DELETE,
] as const;

class SyncService {
  private started = false;
  private running: Promise<void> | null = null;
  private periodTimer: NodeJS.Timeout | null = null;
  private debounceTimer: NodeJS.Timeout | null = null;
  private retryTimer: NodeJS.Timeout | null = null;
  private backoffMs = 0;
  private clockSkewMs: number | null = null;
  private dataChangedAt: number | null = null;
  private readonly unsubscribers: (() => void)[] = [];

  /** Starts timers and listeners. Makes no request unless a server is set, signed in and enabled. */
  start(): void {
    if (this.started) {
      return;
    }
    this.started = true;
    const server = getServer();
    configureTls(server?.url ?? null, server?.allowInsecureTls ?? false);

    for (const eventName of LOCAL_CHANGE_EVENTS) {
      const listener = () => this.scheduleSoon();
      eventService.on(eventName, listener);
      this.unsubscribers.push(() => eventService.off(eventName, listener));
    }
    this.periodTimer = setInterval(() => void this.syncNow(), PERIOD_MS);
    void this.syncNow();
  }

  stop(): void {
    this.started = false;
    for (const timer of [this.periodTimer, this.debounceTimer, this.retryTimer]) {
      if (timer) {
        clearTimeout(timer);
      }
    }
    this.periodTimer = this.debounceTimer = this.retryTimer = null;
    this.unsubscribers.splice(0).forEach((off) => off());
  }

  private canSync(): boolean {
    const server = getServer();
    return Boolean(server?.enabled && server.accountId && loadToken());
  }

  private scheduleSoon(): void {
    if (!this.started || !this.canSync()) {
      this.broadcast();
      return;
    }
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
    }
    this.debounceTimer = setTimeout(() => {
      this.debounceTimer = null;
      void this.syncNow();
    }, DEBOUNCE_MS);
  }

  /** One sync round. Concurrent calls share the round already running. */
  syncNow(): Promise<void> {
    if (!this.canSync()) {
      this.broadcast();
      return Promise.resolve();
    }
    if (!this.running) {
      this.running = this.runOnce().finally(() => {
        this.running = null;
        this.broadcast();
      });
      this.broadcast();
    }
    return this.running;
  }

  private async runOnce(): Promise<void> {
    const server = getServer();
    const token = loadToken();
    if (!server || !token) {
      return;
    }
    const ctx = { url: server.url, token };
    configureTls(server.url, server.allowInsecureTls);

    try {
      const check = await checkServer(server.url);
      this.clockSkewMs = check.clockWarning ? check.clockSkewMs : null;

      let applied = 0;
      if (!server.bootstrappedAt) {
        applied += await bootstrap(ctx);
      }
      if (!getServer()?.backfilledAt) {
        const queued = backfillOutbox();
        updateServerState({ backfilledAt: Date.now() });
        logger.info('Sync', `Queued ${queued} existing row(s) for the first push`);
      }
      await push(ctx);
      applied += await pull(ctx);

      updateServerState({ lastSyncedAt: Date.now(), lastErrorKind: null, lastError: null });
      this.backoffMs = 0;
      if (applied > 0) {
        this.dataChangedAt = Date.now();
        await reminderScheduler.rebuild();
      }
    } catch (error) {
      const kind: SyncErrorKind = error instanceof SyncError ? error.kind : 'server';
      const message = error instanceof Error ? error.message : String(error);
      updateServerState({ lastErrorKind: kind, lastError: message });
      logger.warn('Sync', `Sync failed (${kind}): ${message}`);
      // A refused token or an incompatible server needs the user; retrying will not help.
      if (kind !== 'auth' && kind !== 'protocol') {
        this.scheduleRetry();
      }
    }
  }

  private scheduleRetry(): void {
    this.backoffMs = Math.min(this.backoffMs ? this.backoffMs * 2 : 30_000, MAX_BACKOFF_MS);
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
    }
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      void this.syncNow();
    }, this.backoffMs);
  }

  setEnabled(enabled: boolean): void {
    storeEnabled(enabled);
    if (enabled) {
      void this.syncNow();
    } else {
      this.broadcast();
    }
  }

  /** Re-reads the binding after the settings changed it (server, TLS, sign-in or sign-out). */
  bindingChanged(): void {
    const server = getServer();
    configureTls(server?.url ?? null, server?.allowInsecureTls ?? false);
    this.backoffMs = 0;
    this.clockSkewMs = null;
    this.broadcast();
  }

  getStatus(): SyncStatus {
    const server = getServer();
    return {
      configured: Boolean(server),
      url: server?.url ?? null,
      allowInsecureTls: server?.allowInsecureTls ?? false,
      signedIn: Boolean(server?.accountId && loadToken()),
      accountId: server?.accountId ?? null,
      enabled: server?.enabled ?? false,
      running: this.running !== null,
      pending: countPending() - countFailed(),
      failed: countFailed(),
      lastSyncedAt: server?.lastSyncedAt ?? null,
      lastErrorKind: (server?.lastErrorKind as SyncErrorKind | null) ?? null,
      lastError: server?.lastError ?? null,
      tokenVolatile: isTokenVolatile(),
      clockSkewMs: this.clockSkewMs,
      dataChangedAt: this.dataChangedAt,
    };
  }

  private broadcast(): void {
    const status = this.getStatus();
    for (const win of BrowserWindow.getAllWindows()) {
      if (!win.isDestroyed()) {
        win.webContents.send(SYNC_STATUS_CHANNEL, status);
      }
    }
  }
}

export const syncService = new SyncService();
