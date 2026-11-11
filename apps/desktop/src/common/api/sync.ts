/** Main → app renderer status channel. Internal; not on the extension event bus. */
export const SYNC_STATUS_CHANNEL = 'croffle:sync:status';

/** Why a sync request failed, as the settings screen explains it. */
export type SyncErrorKind = 'auth' | 'offline' | 'tls' | 'server' | 'protocol' | 'clock';

export type SyncStatus = {
  /** A server URL is set. */
  configured: boolean;
  url: string | null;
  allowInsecureTls: boolean;
  signedIn: boolean;
  accountId: string | null;
  enabled: boolean;
  running: boolean;
  /** Queued local changes still being retried. */
  pending: number;
  /** Queued changes that stopped being retried until they are edited again. */
  failed: number;
  lastSyncedAt: number | null;
  lastErrorKind: SyncErrorKind | null;
  lastError: string | null;
  /** The token lives in memory only (no OS keyring); it is gone after a restart. */
  tokenVolatile: boolean;
  /** Server clock minus local clock, when the difference is worth a warning. */
  clockSkewMs: number | null;
  /** Bumped whenever rows from the server changed local data, so views can reload. */
  dataChangedAt: number | null;
};

/**
 * Result of an action the settings screen reports on. A failure keeps the server's error code
 * (`otp_required`, `unauthorized`, …), which an exception thrown across IPC would lose.
 */
export type SyncActionResult =
  | { ok: true; accountId?: string }
  | { ok: false; kind: SyncErrorKind | 'invalid'; code?: string; message: string };

export type SyncPasswordSignIn = { username: string; password: string; otp?: string };

/**
 * Host-only sync controls (`croffleInternal.sync`). Never part of `@croffledev/croffle-types`:
 * extensions cannot reach the sync server, its token or the outbox.
 */
export interface InternalSyncApi {
  getStatus(): Promise<SyncStatus>;
  /** Subscribes to status changes; returns the unsubscribe function. */
  onStatus(callback: (status: SyncStatus) => void): () => void;
  /** Sets or changes the server. A new URL resets the sign-in; queued local changes stay. */
  setServer(url: string): Promise<SyncActionResult>;
  setAllowInsecureTls(allow: boolean): Promise<void>;
  signInWithPassword(input: SyncPasswordSignIn): Promise<SyncActionResult>;
  signInWithToken(token: string): Promise<SyncActionResult>;
  signOut(): Promise<void>;
  setEnabled(enabled: boolean): Promise<void>;
  syncNow(): Promise<void>;
}
