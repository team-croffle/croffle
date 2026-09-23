import { randomUUID } from 'node:crypto';

import { LAST_WRITER_DEVICE_PREFIX } from '@croffledev/common';

import { readAppState, writeAppState } from '../setting/app-state';

let cached: string | null = null;

/** Stable per-installation id, generated once and kept in `app-state.json`. */
export function getClientId(): string {
  if (cached) {
    return cached;
  }
  const state = readAppState();
  if (state.clientId) {
    cached = state.clientId;
    return cached;
  }
  cached = randomUUID();
  writeAppState({ ...state, clientId: cached });
  return cached;
}

/** `lastWriterId` value for rows written by this device. */
export function deviceWriterId(): string {
  return `${LAST_WRITER_DEVICE_PREFIX}${getClientId()}`;
}
