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
