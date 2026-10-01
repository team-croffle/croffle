import type { AppSettings } from '../models/app-settings.js';

export interface SettingsApi {
  getAll(): Promise<AppSettings>;
  getOf(key: string): Promise<AppSettings[keyof AppSettings]>;
  update(newSettings: Partial<AppSettings>): Promise<AppSettings>;
}
