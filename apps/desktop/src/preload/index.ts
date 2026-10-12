import type { CroffleInternalAPI, HostEnums } from '@croffledev/common';
import {
  AppCloseBehavior,
  AppEventType,
  AppSettingLanguage,
  AppSettingTheme,
  CalendarTimeFormat,
  CalendarView,
  CalendarWeekStartDay,
  ClipboardDataType,
} from '@croffledev/common';
import type { CroffleAPI } from '@croffledev/croffle-types';
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
  AppCloseBehavior,
  ClipboardDataType,
  AppEventType,
} satisfies HostEnums;

/** Public host API: exactly what `@croffledev/croffle-types` declares, nothing more. */
const api = {
  os: osApi,
  http: httpApi,
  event: eventApi,
  calendar: {
    schedules: {
      getAll: scheduleApi.getAll,
      create: scheduleApi.create,
      update: scheduleApi.update,
      remove: scheduleApi.remove,
      moveOccurrence: scheduleApi.moveOccurrence,
    },
    tags: tagApi,
    search: searchApi,
  },
  settings: {
    getAll: settingsApi.getAll,
    getOf: settingsApi.getOf,
  },
  extensions: {
    info: {
      getInstalled: extensionInfoApi.getInstalled,
      getEnabled: extensionInfoApi.getEnabled,
      getByName: extensionInfoApi.getByName,
    },
  },
  enums,
} satisfies CroffleAPI;

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
