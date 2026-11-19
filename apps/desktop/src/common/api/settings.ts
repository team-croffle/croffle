import type { AppSettings, SettingsApi } from '@croffledev/croffle-types';

/** Main → app renderer: settings were stored. Internal; not on the extension event bus. */
export const SETTINGS_STORED_CHANNEL = 'croffle:settings:stored';

/**
 * Stored settings with main's revision, which goes up by one on every store. The app window
 * keeps only the highest revision it has seen, so the order in which replies and broadcasts
 * arrive cannot bring back an older value.
 */
export type StoredSettings = { revision: number; settings: AppSettings };

/** Public read API plus the host-only writes. */
export interface InternalSettingsApi extends SettingsApi {
  /** Stores a patch (main merges it per section) and answers with the revision it made. */
  update(newSettings: Partial<AppSettings>): Promise<StoredSettings>;
  getStored(): Promise<StoredSettings>;
  /** Subscribes to stored settings; returns the unsubscribe function. */
  onStored(callback: (stored: StoredSettings) => void): () => void;
}
