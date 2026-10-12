import type { CroffleInternalAPI, HostEnums } from '@croffledev/common';
import {
  AppEventType,
  AppSettingLanguage,
  AppSettingTheme,
  CalendarTimeFormat,
  CalendarView,
  CalendarWeekStartDay,
  ClipboardDataType,
} from '@croffledev/common';
import { contextBridge } from 'electron';

import { scheduleApi } from './api/calendar/schedules';
import { searchApi } from './api/calendar/search';
import { tagApi } from './api/calendar/tags';
import { eventApi } from './api/event';
import { extensionConfigurationApi } from './api/extensions/configuration';
import { extensionInfoApi } from './api/extensions/info';
import { extensionSessionApi } from './api/extensions/session';
import { extensionStorageApi } from './api/extensions/storage';
import { httpApi } from './api/http';
import { osApi } from './api/os';
import { settingsApi } from './api/settings';
import { windowApi } from './api/window';

/** Only the enum objects — never the rest of `@croffledev/common`. */
const enums = {
  AppSettingLanguage,
  AppSettingTheme,
  CalendarView,
  CalendarWeekStartDay,
  CalendarTimeFormat,
  ClipboardDataType,
  AppEventType,
} satisfies HostEnums;

/** Host API reachable by extensions (narrowed to the public surface in 2.0). */
const api = {
  window: windowApi,
  os: osApi,
  http: httpApi,
  event: eventApi,
  calendar: {
    schedules: scheduleApi,
    tags: tagApi,
    search: searchApi,
  },
  settings: settingsApi,
  extensions: {
    info: extensionInfoApi,
    configuration: extensionConfigurationApi,
    storage: extensionStorageApi,
    session: extensionSessionApi,
  },
  enums,
};

/** Host-only API. The extension loader never hands this to an extension. */
const internalApi = {
  window: windowApi,
  settings: {
    update: settingsApi.update,
  },
  extensions: {
    info: {
      install: extensionInfoApi.install,
      installFromLocal: extensionInfoApi.installFromLocal,
      toggle: extensionInfoApi.toggle,
      uninstall: extensionInfoApi.uninstall,
    },
    configuration: extensionConfigurationApi,
    storage: extensionStorageApi,
    session: extensionSessionApi,
  },
  calendar: {
    schedules: {
      exportSchedulesToFile: scheduleApi.exportSchedulesToFile,
      importScheduleFromFile: scheduleApi.importScheduleFromFile,
    },
  },
} satisfies CroffleInternalAPI;

if (process.contextIsolated) {
  try {
    contextBridge.exposeInMainWorld('croffle', api);
    contextBridge.exposeInMainWorld('croffleInternal', internalApi);
  } catch {
    // oxlint-disable-next-line no-console
    console.error('Failed to expose API to main world');
  }
} else {
  window.croffle = api;
  window.croffleInternal = internalApi;
}
