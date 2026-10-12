import type { InternalSchedulesApi } from './calendar';
import type {
  ExtensionConfigurationApi,
  ExtensionSessionApi,
  ExtensionStorageApi,
  InternalExtensionInfoApi,
} from './extensions';
import type { InternalSettingsApi } from './settings';
import type { InternalWindowApi } from './window';

export * from './calendar';
export * from './enums';
export * from './extensions';
export * from './settings';
export * from './window';

/**
 * Host-only API exposed on the `croffleInternal` bridge. The extension loader never
 * passes any of this to an extension; the public surface is `CroffleAPI` in
 * `@croffledev/croffle-types`.
 */
/**
 * What the preload actually puts on `window.croffleInternal`: the API is handed out
 * once. The renderer host takes it at startup, so extension code calling `take()`
 * later gets `undefined`.
 */
export interface CroffleInternalBridge {
  take(): CroffleInternalAPI | undefined;
}

export interface CroffleInternalAPI {
  window: InternalWindowApi;
  settings: Pick<InternalSettingsApi, 'update'>;
  extensions: {
    info: Pick<InternalExtensionInfoApi, 'install' | 'installFromLocal' | 'toggle' | 'uninstall'>;
    configuration: ExtensionConfigurationApi;
    storage: ExtensionStorageApi;
    session: ExtensionSessionApi;
  };
  calendar: {
    schedules: Pick<InternalSchedulesApi, 'exportSchedulesToFile' | 'importScheduleFromFile'>;
  };
}
