export enum AppSettingLanguage {
  KO = 'ko',
  EN = 'en',
}

export enum AppSettingTheme {
  LIGHT = 'light',
  DARK = 'dark',
  SYSTEM = 'system',
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

/** What the window close button does. `ASK` shows a one-time prompt and stores the answer. */
export enum AppCloseBehavior {
  ASK = 'ask',
  TRAY = 'tray',
  QUIT = 'quit',
}

export enum ClipboardDataType {
  TEXT = 'text',
  IMAGE = 'image',
  EMPTY = 'empty',
  ERROR = 'error',
}

// Domain mutations are for extension hooks (ctx.event.on).
// UPDATE_* are for the host updater UI.
export enum AppEventType {
  // Schedule
  SCHEDULE_CREATE = 'schedule:create',
  SCHEDULE_UPDATE = 'schedule:update',
  SCHEDULE_DELETE = 'schedule:delete',
  SCHEDULE_EXPORT_TO_FILE = 'schedule:exportToFile',
  SCHEDULE_IMPORT_FROM_FILE = 'schedule:importFromFile',

  // Tag
  TAG_CREATE = 'tag:create',
  TAG_UPDATE = 'tag:update',
  TAG_DELETE = 'tag:delete',

  // Extension lifecycle
  EXTENSION_INFO_INSTALL = 'extensionInfo:install',
  EXTENSION_INFO_TOGGLE = 'extensionInfo:toggle',
  EXTENSION_INFO_UNINSTALL = 'extensionInfo:uninstall',

  // Settings
  SETTINGS_UPDATE = 'settings:update',

  // Window lifecycle
  WINDOW_SHOW = 'window:show',
  WINDOW_HIDE = 'window:hide',
  WINDOW_EXIT = 'window:exit',
  /** Close was requested while `closeBehavior` is `ask`; the host UI asks the user. */
  WINDOW_CLOSE_REQUESTED = 'window:closeRequested',

  // Electron updater (host UI)
  UPDATE_AVAILABLE = 'update:available',
  UPDATE_NOT_AVAILABLE = 'update:notAvailable',
  UPDATE_DOWNLOAD_PROGRESS = 'update:downloadProgress',
  UPDATE_DOWNLOADED = 'update:downloaded',
  UPDATE_ERROR = 'update:error',
  UPDATE_DOWNLOAD_NOW = 'update:downloadNow',
  UPDATE_DOWNLOAD_LATER = 'update:downloadLater',
  UPDATE_SKIP_THIS_VERSION = 'update:skipThisVersion',
}
