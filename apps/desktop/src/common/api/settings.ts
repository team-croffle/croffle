import type { AppSettings, SettingsApi } from '@croffledev/croffle-types';

/** Public read API plus the host-only `update`. */
export interface InternalSettingsApi extends SettingsApi {
  update(newSettings: Partial<AppSettings>): Promise<AppSettings>;
}
