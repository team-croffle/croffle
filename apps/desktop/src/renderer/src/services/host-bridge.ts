import type { CroffleInternalAPI } from '@croffledev/common';

/**
 * Host-only bridge. Captured once at startup (this module is evaluated before any
 * extension loads) and removed from `window` so extension code cannot reach it by
 * name. If the contextBridge property cannot be deleted, the boundary is only
 * enforced by types and by the extension context (see extension-loader).
 */
const captured: CroffleInternalAPI = window.croffleInternal;

try {
  delete (window as { croffleInternal?: CroffleInternalAPI }).croffleInternal;
} catch {
  // Non-configurable bridge property — nothing more to do here.
}

if (import.meta.env.DEV && 'croffleInternal' in window) {
  // oxlint-disable-next-line no-console
  console.warn('[croffle] window.croffleInternal could not be removed from the main world');
}

export const croffleInternal = captured;
