import { existsSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import * as path from 'node:path';

import { is } from '@electron-toolkit/utils';
import { eq } from 'drizzle-orm';
import { app, safeStorage } from 'electron';

import { databaseManager } from '../database';
import { syncServer, type SyncServerRow } from '../database/schema';
import { logger } from '../logger';

const SERVER_ROW_ID = 'default';

/** Fields the sync engine updates as it goes. */
export type SyncServerState = Partial<
  Pick<
    SyncServerRow,
    | 'accountId'
    | 'teamId'
    | 'lastPulledSeq'
    | 'bootstrapCursor'
    | 'bootstrappedAt'
    | 'backfilledAt'
    | 'lastSyncedAt'
    | 'lastErrorKind'
    | 'lastError'
  >
>;

/** What a binding starts from: nothing pulled, nothing bootstrapped, signed out. */
const UNBOUND_STATE = {
  accountId: null,
  teamId: null,
  enabled: false,
  lastPulledSeq: 0,
  bootstrapCursor: null,
  bootstrappedAt: null,
  backfilledAt: null,
  lastSyncedAt: null,
  lastErrorKind: null,
  lastError: null,
} satisfies Partial<SyncServerRow>;

export class InvalidServerUrlError extends Error {}

/**
 * Bumped whenever the binding changes (server, account, sign-out). A sync round or a sign-in
 * started under an older generation must not write anything: its answers belong to a binding
 * that no longer exists.
 */
let generation = 0;

export class BindingChangedError extends Error {
  constructor() {
    super('The sync binding changed while a request was running');
    this.name = 'BindingChangedError';
  }
}

export function bindingGeneration(): number {
  return generation;
}

/** Throws unless the binding is still the one `expected` was read from. */
export function assertBinding(expected: number): void {
  if (expected !== generation) {
    throw new BindingChangedError();
  }
}

/** `https://host[:port][/base]` without a trailing slash. Only http and https are accepted. */
export function normalizeServerUrl(input: string): string {
  let url: URL;
  try {
    url = new URL(input.trim());
  } catch {
    throw new InvalidServerUrlError('Not a URL');
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new InvalidServerUrlError('Only http and https are supported');
  }
  if (url.username || url.password || url.search || url.hash) {
    throw new InvalidServerUrlError('The URL must not carry credentials, a query or a fragment');
  }
  return `${url.origin}${url.pathname.replace(/\/+$/, '')}`;
}

export function getServer(): SyncServerRow | undefined {
  const [row] = databaseManager
    .getDb()
    .select()
    .from(syncServer)
    .where(eq(syncServer.id, SERVER_ROW_ID))
    .all();
  return row;
}

/**
 * Binds this device to a server. Changing the URL starts a new binding: the token, cursors and
 * account are reset. Queued local changes are kept, so nothing written offline is lost.
 */
export function setServer(input: string): SyncServerRow {
  const url = normalizeServerUrl(input);
  const current = getServer();
  if (current?.url === url) {
    return current;
  }

  generation++;
  const db = databaseManager.getDb();
  if (current) {
    db.update(syncServer)
      .set({ url, ...UNBOUND_STATE })
      .where(eq(syncServer.id, SERVER_ROW_ID))
      .run();
  } else {
    db.insert(syncServer).values({ id: SERVER_ROW_ID, url }).run();
  }
  clearToken();
  return getServer()!;
}

/** Forgets the server entirely (token and binding). */
export function removeServer(): void {
  generation++;
  databaseManager.getDb().delete(syncServer).where(eq(syncServer.id, SERVER_ROW_ID)).run();
  clearToken();
}

/** Signs out: the token and the account binding go, the server URL and TLS choice stay. */
export function resetBinding(): void {
  generation++;
  databaseManager
    .getDb()
    .update(syncServer)
    .set(UNBOUND_STATE)
    .where(eq(syncServer.id, SERVER_ROW_ID))
    .run();
  clearToken();
}

export function setAllowInsecureTls(allow: boolean): void {
  databaseManager
    .getDb()
    .update(syncServer)
    .set({ allowInsecureTls: allow })
    .where(eq(syncServer.id, SERVER_ROW_ID))
    .run();
}

export function setEnabled(enabled: boolean): void {
  databaseManager
    .getDb()
    .update(syncServer)
    .set({ enabled })
    .where(eq(syncServer.id, SERVER_ROW_ID))
    .run();
}

export function updateServerState(state: SyncServerState): void {
  databaseManager
    .getDb()
    .update(syncServer)
    .set(state)
    .where(eq(syncServer.id, SERVER_ROW_ID))
    .run();
}

// --- token ------------------------------------------------------------------------------------

/** Held only in memory when the OS offers no keyring; lost on restart. */
let volatileToken: string | null = null;

function tokenPath(): string {
  return is.dev
    ? path.join(process.cwd(), 'dev/sync-token.bin')
    : path.join(app.getPath('userData'), 'sync-token.bin');
}

/** True when the token cannot be stored safely and lives in memory only. */
export function isTokenVolatile(): boolean {
  return !safeStorage.isEncryptionAvailable();
}

/** Stores the token encrypted, or in memory when encryption is unavailable. Never in plain text. */
export function saveToken(token: string): void {
  if (isTokenVolatile()) {
    volatileToken = token;
    rmSync(tokenPath(), { force: true });
    logger.warn('Sync', 'No OS keyring: the sync token is kept in memory until the app quits.');
    return;
  }
  volatileToken = null;
  writeFileSync(tokenPath(), safeStorage.encryptString(token), { mode: 0o600 });
}

export function loadToken(): string | null {
  if (volatileToken) {
    return volatileToken;
  }
  const file = tokenPath();
  if (isTokenVolatile() || !existsSync(file)) {
    return null;
  }
  try {
    return safeStorage.decryptString(readFileSync(file));
  } catch {
    logger.warn('Sync', 'The stored sync token could not be decrypted; sign in again.');
    return null;
  }
}

export function clearToken(): void {
  volatileToken = null;
  rmSync(tokenPath(), { force: true });
}
