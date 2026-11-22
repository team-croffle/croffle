import type { SyncServerRow } from '../database/schema';

/**
 * What signing out writes: nothing pulled, nothing bootstrapped, no account. `enabled` is left as
 * it was, so signing in again resumes with the switch the user had (1.3.3); without an account
 * and a token nothing syncs in the meantime.
 */
export const SIGNED_OUT_STATE = {
  accountId: null,
  teamId: null,
  lastPulledSeq: 0,
  bootstrapCursor: null,
  bootstrappedAt: null,
  backfilledAt: null,
  lastSyncedAt: null,
  lastErrorKind: null,
  lastError: null,
} satisfies Partial<SyncServerRow>;

/** What a new server URL writes: a new binding starts signed out and switched off. */
export const NEW_SERVER_STATE = {
  ...SIGNED_OUT_STATE,
  enabled: false,
} satisfies Partial<SyncServerRow>;
