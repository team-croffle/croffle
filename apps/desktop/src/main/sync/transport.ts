import {
  ERROR_CODES,
  HEADER_CLIENT_ID,
  HEADER_VERSION,
  PROTOCOL_VERSION,
  type ErrorBody,
  type HealthResponse,
  type SyncErrorKind,
} from '@croffledev/common';
import { session, type Session } from 'electron';

import { logger } from '../logger';
import { getClientId } from './client-id';

export class SyncError extends Error {
  constructor(
    readonly kind: SyncErrorKind,
    message: string,
    /** HTTP status, or 0 when no response arrived. */
    readonly status = 0,
    /** The server's error code (`ErrorBody.code`), when it sent one. */
    readonly code?: string,
  ) {
    super(message);
    this.name = 'SyncError';
  }
}

const SESSION_PARTITION = 'croffle-sync';
const REQUEST_TIMEOUT_MS = 20_000;
const RETRY_DELAYS_MS = [500, 1_000, 2_000];
/** Beyond this, a wall-clock difference is reported: LWW compares client clocks. */
export const CLOCK_SKEW_WARNING_MS = 5 * 60_000;

let syncSession: Session | null = null;
let insecureHost: string | null = null;

/**
 * Sync requests use their own session, so relaxing TLS for a self-hosted server never reaches
 * the default session that extensions' `http.*` uses.
 */
function getSession(): Session {
  if (!syncSession) {
    syncSession = session.fromPartition(SESSION_PARTITION, { cache: false });
    syncSession.setCertificateVerifyProc((request, callback) => {
      // 0 accepts the certificate, -3 keeps Chromium's own verdict.
      callback(insecureHost !== null && request.hostname === insecureHost ? 0 : -3);
    });
  }
  return syncSession;
}

/**
 * Accepts an unverified certificate for the sync server's host only, and only while the user
 * keeps the setting on. Any other host, and the default session, stay strict.
 */
export function configureTls(serverUrl: string | null, allowInsecure: boolean): void {
  insecureHost = serverUrl && allowInsecure ? new URL(serverUrl).hostname : null;
  getSession();
}

export type RequestOptions = {
  token?: string | null;
  body?: unknown;
  /** Retries network failures and 5xx answers with backoff. Off for logins (not idempotent). */
  retry?: boolean;
};

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isTlsFailure(error: unknown): boolean {
  return /ERR_CERT_|ERR_SSL_|certificate/i.test(String(error));
}

async function readError(response: Response): Promise<ErrorBody | undefined> {
  try {
    const body = (await response.json()) as Partial<ErrorBody>;
    return typeof body.code === 'string' ? { code: body.code, message: body.message } : undefined;
  } catch {
    return undefined;
  }
}

function kindOf(status: number, code: string | undefined): SyncErrorKind {
  if (status === 401 || (status === 403 && code === ERROR_CODES.accountDisabled)) {
    return 'auth';
  }
  return 'server';
}

async function attempt<T>(
  method: string,
  url: string,
  apiPath: string,
  options: RequestOptions,
): Promise<T> {
  const headers: Record<string, string> = {
    [HEADER_VERSION]: String(PROTOCOL_VERSION),
    [HEADER_CLIENT_ID]: getClientId(),
    Accept: 'application/json',
  };
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`;
  }
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  let response: Response;
  try {
    response = await getSession().fetch(url, {
      method,
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: controller.signal,
      redirect: 'error',
    });
  } catch (error) {
    // The cause can carry the URL but never headers, so it is safe to log.
    logger.info('Sync', `${method} ${apiPath} failed`);
    if (isTlsFailure(error)) {
      throw new SyncError('tls', 'The server certificate was not accepted');
    }
    throw new SyncError('offline', 'The server could not be reached');
  } finally {
    clearTimeout(timer);
  }

  logger.info('Sync', `${method} ${apiPath} ${response.status}`);
  if (response.ok) {
    return (await response.json()) as T;
  }
  const error = await readError(response);
  throw new SyncError(
    kindOf(response.status, error?.code),
    error?.message ?? `HTTP ${response.status}`,
    response.status,
    error?.code,
  );
}

function isRetryable(error: unknown): boolean {
  return error instanceof SyncError && (error.kind === 'offline' || error.status >= 500);
}

/** One JSON request to the sync server. `apiPath` is relative to the server URL (`/sync/pull`). */
export async function request<T>(
  serverUrl: string,
  method: 'GET' | 'POST',
  apiPath: string,
  options: RequestOptions = {},
): Promise<T> {
  const url = `${serverUrl}${apiPath}`;
  const delays = options.retry === false ? [] : RETRY_DELAYS_MS;
  for (let i = 0; ; i++) {
    try {
      return await attempt<T>(method, url, apiPath, options);
    } catch (error) {
      if (i >= delays.length || !isRetryable(error)) {
        throw error;
      }
      await delay(delays[i]);
    }
  }
}

export type ServerCheck = {
  health: HealthResponse;
  /** Server clock minus local clock, in milliseconds. */
  clockSkewMs: number;
  /** True when the skew is large enough to tell the user. Sync still runs. */
  clockWarning: boolean;
};

/**
 * Asks `/health` which protocol versions the server speaks before any sync request, so an
 * incompatible pair is reported in the settings instead of failing in the background.
 */
export async function checkServer(serverUrl: string): Promise<ServerCheck> {
  const health = await request<HealthResponse>(serverUrl, 'GET', '/health');
  const { min, max } = health.protocol ?? { min: 0, max: -1 };
  if (PROTOCOL_VERSION < min) {
    throw new SyncError(
      'protocol',
      `The server needs sync protocol ${min}–${max}; update Croffle (this version speaks ${PROTOCOL_VERSION})`,
    );
  }
  if (PROTOCOL_VERSION > max) {
    throw new SyncError(
      'protocol',
      `The server speaks sync protocol ${min}–${max}; update the server (this version speaks ${PROTOCOL_VERSION})`,
    );
  }
  const clockSkewMs = health.time - Date.now();
  return { health, clockSkewMs, clockWarning: Math.abs(clockSkewMs) > CLOCK_SKEW_WARNING_MS };
}
