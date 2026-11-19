/**
 * State of "check for updates", shared by the version tile and Settings › General › Updates.
 * Pure so it can be tested without Electron; the renderer's update store feeds it events.
 */
export type UpdateCheckStatus =
  | { kind: 'idle' }
  | { kind: 'checking' }
  | { kind: 'latest' }
  | { kind: 'available'; version: string }
  | { kind: 'error'; message: string }
  /** This install cannot check (development build, or not an updatable package such as a bare Linux folder). */
  | { kind: 'unsupported'; reason: UpdateUnsupportedReason };

export type UpdateUnsupportedReason = 'development' | 'not-updatable';

/** What the main process answers when a check is asked for. */
export type UpdateCheckStart =
  | { status: 'started' }
  | { status: 'unsupported'; reason: UpdateUnsupportedReason };

export type UpdateCheckEvent =
  | { type: 'check'; result: UpdateCheckStart }
  | { type: 'not-available' }
  | { type: 'available'; version: string }
  | { type: 'error'; message: string };

export const UPDATE_IDLE: UpdateCheckStatus = { kind: 'idle' };

/** Next state. Results that arrive with no check running (the startup check) still update it. */
export function nextUpdateStatus(
  _status: UpdateCheckStatus,
  event: UpdateCheckEvent,
): UpdateCheckStatus {
  switch (event.type) {
    case 'check':
      return event.result.status === 'started'
        ? { kind: 'checking' }
        : { kind: 'unsupported', reason: event.result.reason };
    case 'not-available':
      return { kind: 'latest' };
    case 'available':
      return { kind: 'available', version: event.version };
    case 'error':
      return { kind: 'error', message: event.message };
  }
}

/**
 * Whether a "you are up to date" toast is shown. A check started by hand shows its result in
 * place (the tile and the settings row), so only the startup check gets the toast.
 */
export function toastsLatest(manual: boolean): boolean {
  return !manual;
}
