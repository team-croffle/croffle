import type { AppSettings } from '../models/app-settings.js';

/** Read-only view of the app settings. Writing them is host-only. */
export interface SettingsApi {
  getAll(): Promise<AppSettings>;
  getOf(key: string): Promise<AppSettings[keyof AppSettings]>;
}
