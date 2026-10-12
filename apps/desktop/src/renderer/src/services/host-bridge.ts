import type { CroffleInternalAPI } from '@croffledev/common';

/**
 * Host-only bridge. The preload hands the internal API out exactly once; this module
 * is evaluated before any extension loads, so it is the host that gets it and an
 * extension calling `window.croffleInternal.take()` later receives `undefined`.
 */
const captured: CroffleInternalAPI | undefined = window.croffleInternal?.take();

if (!captured) {
  throw new Error('[croffle] internal host bridge unavailable (already taken or preload missing)');
}

export const croffleInternal: CroffleInternalAPI = captured;
