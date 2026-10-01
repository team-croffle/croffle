import type { CalendarApi } from './api/calendar.js';
import type { EventApi } from './api/event.js';
import type {
  ExtensionScopedConfigurationApi,
  ExtensionScopedSessionApi,
  ExtensionScopedStorageApi,
  ExtensionsApi,
} from './api/extensions.js';
import type { HttpApi } from './api/http.js';
import type { OsApi } from './api/os.js';
import type { SettingsApi } from './api/settings.js';
import type { UiApi } from './api/ui.js';
import type { WindowApi } from './api/window.js';
import type { AppEventType } from './enums.js';
import type {
  AppSettingLanguage,
  AppSettingTheme,
  CalendarTimeFormat,
  CalendarView,
  CalendarWeekStartDay,
} from './models/app-settings.js';
import type { ClipboardDataType } from './models/clipboard.js';

export interface EnumsApi {
  AppSettingLanguage: typeof AppSettingLanguage;
  AppSettingTheme: typeof AppSettingTheme;
  CalendarView: typeof CalendarView;
  CalendarWeekStartDay: typeof CalendarWeekStartDay;
  CalendarTimeFormat: typeof CalendarTimeFormat;
  ClipboardDataType: typeof ClipboardDataType;
  AppEventType: typeof AppEventType;
}

/** Preload에 노출되는 호스트 API (ui 제외) */
export interface CroffleAPI {
  window: WindowApi;
  os: OsApi;
  http: HttpApi;
  event: EventApi;
  calendar: CalendarApi;
  settings: SettingsApi;
  extensions: ExtensionsApi;
  enums: EnumsApi;
}

/** Extension activated(context)에 전달되는 API (+ ui + id-bound storage) */
export type ExtensionContext = CroffleAPI & {
  ui: UiApi;
  /** Prefer over `extensions.storage` — id is fixed to this extension */
  storage: ExtensionScopedStorageApi;
  session: ExtensionScopedSessionApi;
  configuration: ExtensionScopedConfigurationApi;
};
