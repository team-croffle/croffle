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

export enum ClipboardDataType {
  TEXT = 'text',
  IMAGE = 'image',
  EMPTY = 'empty',
  ERROR = 'error',
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
  PLUGIN_INFO_INSTALL = 'pluginInfo:install',
  PLUGIN_INFO_TOGGLE = 'pluginInfo:toggle',
  PLUGIN_INFO_UNINSTALL = 'pluginInfo:uninstall',

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
