import type { UpdateCheckStart } from '../update-status';

/**
 * Host-only window controls. Not part of `@croffledev/croffle-types`:
 * extensions never get these (exposed on the `croffleInternal` bridge only).
 */
export interface InternalWindowApi {
  minimize(): Promise<void>;
  maximize(): Promise<void>;
  close(): Promise<void>;
  exitApp(): Promise<void>;
  /** Starts a check by hand; the result arrives as the UPDATE_* events. */
  checkForUpdates(): Promise<UpdateCheckStart>;
}
