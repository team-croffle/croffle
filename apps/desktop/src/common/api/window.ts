/**
 * Host-only window controls. Not part of `@croffledev/croffle-types`:
 * extensions never get these (exposed on the `croffleInternal` bridge only).
 */
export interface InternalWindowApi {
  minimize(): Promise<void>;
  maximize(): Promise<void>;
  close(): Promise<void>;
  exitApp(): Promise<void>;
  checkForUpdates(): Promise<void>;
}
