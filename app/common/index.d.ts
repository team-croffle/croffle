declare module 'croffle' {
  export interface Schedule {
    id: string;
    title: string;
    description: string;
    location: string;
    startDate: string; // ISO 8601 format
    endDate: string; // ISO 8601 format
    isAllDay: boolean;
    recurrenceRule?: string;
    colorLabel: string;
    tags: Tag[];
    createdAt: string; // ISO 8601 format
    updatedAt: string; // ISO 8601 format
  }

  export interface Tag {
    id: string;
    name: string;
    color: string;
  }

  export interface SearchQuery {
    text?: string;
    dateRange?: {
      start: string; // ISO 8601 format
      end: string; // ISO 8601 format
    };
    tags?: Tag[];
  }

  export interface PluginInfo {
    id: string;
    name: string;
    version: string;
    author: string;
    description: string;
    enabled: boolean;
  }

  export enum AppSettingLanguage {
    KO = 'ko',
    EN = 'en',
  }

  export enum AppSettingTheme {
    LIGHT = 'light',
    DARK = 'dark',
    SYSTEM = 'system',
  }

  export enum AppSettingStartupBehavior {
    OPEN_LAST_SESSION = 'openLastSession',
    OPEN_NEW_WINDOW = 'openNewWindow',
    DO_NOTHING = 'doNothing',
  }

  export enum CalendarView {
    DAY = 'day',
    WEEK = 'week',
    MONTH = 'month',
    YEAR = 'year',
  }

  export enum CalendarWeekStartDay {
    SUNDAY = 'sunday',
    MONDAY = 'monday',
  }

  export enum CalendarTimeFormat {
    H12 = '12h',
    H24 = '24h',
  }

  export interface AppSettings {
    general: {
      language: AppSettingLanguage;
      theme: AppSettingTheme;
      autoUpdate: boolean;
      startupBehavior: AppSettingStartupBehavior;
      startOnSystemBoot: boolean;
      startMinimized: boolean;
    };
    calendar: {
      defaultView: CalendarView;
      weekStartDay: CalendarWeekStartDay;
      showWeekNumbers: boolean;
      timeFormat: CalendarTimeFormat;
    };
    notifications: {
      enabled: boolean;
      defaultReminderMinutes: number;
    };
  }

  export enum ClipboardDataType {
    TEXT = 'text',
    IMAGE = 'image',
    EMPTY = 'empty',
    ERROR = 'error',
  }

  export interface ClipboardResult {
    type: ClipboardDataType;
    value: string | Buffer | null;
  }

  export interface ClipboardTextData {
    type: ClipboardDataType.TEXT;
    value: string;
  }

  export interface ClipboardImageData {
    type: ClipboardDataType.IMAGE;
    value: Buffer;
  }

  export interface HttpResponse<T = unknown> {
    ok: boolean;
    status: number;
    data: T | null;
  }

  // IPC Event Types in main process
  export enum AppEventType {
    // Schedule
    SCHEDULE_GET = 'schedule:get',
    SCHEDULE_CREATE = 'schedule:create',
    SCHEDULE_UPDATE = 'schedule:update',
    SCHEDULE_DELETE = 'schedule:delete',

    // Tag
    TAG_GET = 'tag:getAll',
    TAG_GET_BY_NAME = 'tag:getByName',
    TAG_CREATE = 'tag:create',
    TAG_UPDATE = 'tag:update',
    TAG_DELETE = 'tag:delete',

    // Plugin Info
    PLUGIN_INFO_GET_INSTALLED = 'pluginInfo:getInstalled',
    PLUGIN_INFO_GET_ENABLED = 'pluginInfo:getEnabled',
    PLUGIN_INFO_GET_BY_NAME = 'pluginInfo:getByName',
    PLUGIN_INFO_CREATE = 'pluginInfo:create',
    PLUGIN_INFO_UPDATE = 'pluginInfo:update',
    PLUGIN_INFO_DELETE = 'pluginInfo:delete',

    // Plugin Storage
    PLUGIN_STORAGE_GET = 'pluginStorage:get',
    PLUGIN_STORAGE_SET = 'pluginStorage:set',
    PLUGIN_STORAGE_DELETE = 'pluginStorage:delete',

    // Settings
    SETTINGS_GET = 'settings:get',
    SETTINGS_GET_OF = 'setttings:getOf',
    SETTINGS_UPDATE = 'settings:update',

    // Window
    WINDOW_MINIMIZE = 'window:minimize',
    WINDOW_RESTORE = 'window:restore',
    WINDOW_MAXIMIZE = 'window:maximize',
    WINDOW_CLOSE = 'window:close',
    WINDOW_EXIT = 'window:exit',
    WINDOW_CHECK_FOR_UPDATES = 'window:checkForUpdates',
    WINDOW_SHOW = 'window:show',
    WINDOW_HIDE = 'window:hide',

    // OS Service
    NATIVE_OS_NOTIFICATION = 'nativeOs:notification',
    NATIVE_OS_CLIPBOARD_GET = 'nativeOs:clipboard:get',
    NATIVE_OS_CLIPBOARD_SET = 'nativeOs:clipboard:set',

    // HTTP Service
    HTTP_SERVICE_GET = 'httpService:get',
    HTTP_SERVICE_POST = 'httpService:post',

    // Plugin
    PLUGIN_INSTALL = 'plugin:install',
    PLUGIN_TOGGLE = 'plugin:toggle',
    PLUGIN_UNINSTALL = 'plugin:uninstall',

    // Plugin Session Storage
    PLUGIN_SESSION_STORAGE_GET = 'sessionStorage:get',
    PLUGIN_SESSION_STORAGE_CREATE = 'sessionStorage:create',
    PLUGIN_SESSION_STORAGE_UPDATE = 'sessionStorage:update',
    PLUGIN_SESSION_STORAGE_DELETE = 'sessionStorage:delete',

    // Background works
    SCHEDULER_REGISTER = 'scheduler:register',
    SCHEDULER_UNREGISTER = 'scheduler:unregister',

    // UI Extensions
    UI_ADD_MENU_ITEM = 'ui:addMenuItem',
    UI_CONTEXT_MENU_ADD_ITEM = 'ui:contextMenu:addItem',
  }

  export namespace windows {
    export function minimize(): Promise<void>;
    export function maximize(): Promise<void>;
    export function close(): Promise<void>;
    export function exitApp(): Promise<void>;
    export function checkForUpdates(): Promise<void>;
    export function setCloseToTrayMode(enabled: boolean): Promise<void>;
  }

  export namespace tags {
    export function getAll(): Promise<Tag[]>;
    export function getByName(name: string): Promise<Tag | null>;
    export function create(name: string, color: string): Promise<Tag>;
    export function modify(id: string, name: string, color: string): Promise<Tag>;
    export function remove(id: string): Promise<boolean>;
  }

  export namespace schedules {
    export function getAll(period: { start: string; end: string }): Promise<Schedule[]>;
    export function create(data: Partial<Schedule>): Promise<Schedule>;
    export function update(id: string, data: Partial<Schedule>): Promise<Schedule>;
    export function remove(id: string): Promise<boolean>;
  }

  export namespace pluginInfo {
    export function getInstalled(): Promise<PluginInfo[]>;
    export function getEnabled(): Promise<PluginInfo[]>;
    export function getByName(name: string): Promise<PluginInfo | null>;
    export function install(data: Partial<PluginInfo>): Promise<PluginInfo>;
    export function toggle(name: string, enable: boolean): Promise<PluginInfo | null>;
    export function uninstall(name: string): Promise<boolean>;
  }

  export namespace settings {
    export function getAll(): Promise<AppSettings>;
    export function getOf(key: string): Promise<AppSettings[keyof AppSettings]>;
    export function update(newSettings: Partial<AppSettings>): Promise<AppSettings>;
  }

  export namespace pluginStorage {
    export function get(pluginId: string, key: string): Promise<string | null>;
    export function set(pluginId: string, key: string, value: string): Promise<void>;
  }

  export namespace os {
    export function showNotification(title: string, body: string): Promise<void>;
    export function getClipboard(): Promise<ClipboardResult>;
    export function setClipboard(data: ClipboardTextData | ClipboardImageData): Promise<void>;
  }

  export namespace http {
    export function get(
      url: string,
      params?: Record<string, string>,
      headers?: Record<string, string>
    ): Promise<HttpResponse>;
    export function post(
      url: string,
      body?: unknown,
      headers?: Record<string, string>
    ): Promise<HttpResponse>;
  }

  export namespace event {
    /**
     * Send event to all windows
     * @param type event type
     * @param payload event payload
     */
    export function emit(type: string, payload: unknown): void;
    /**
     * Listen to event from all windows
     * @param type event type
     * @param callback callback function
     * @returns unsubscribe function
     */
    export function on(type: string, callback: (payload: unknown) => void): () => void;
  }

  export const base: {
    windows: typeof windows;
    tags: typeof tags;
    schedules: typeof schedules;
    pluginInfo: typeof pluginInfo;
    settings: typeof settings;
  };

  export const app: {
    os: typeof os;
    http: typeof http;
    storage: typeof pluginStorage;
    event: typeof event;
  };
}
