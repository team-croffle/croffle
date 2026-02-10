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
  TAG_GET = 'tag:get',
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
  PLUGIN_STORAGE_CREATE = 'pluginStorage:create',
  PLUGIN_STORAGE_UPDATE = 'pluginStorage:update',
  PLUGIN_STORAGE_DELETE = 'pluginStorage:delete',

  // Settings
  SETTINGS_GET = 'settings:get',
  SETTINGS_CREATE = 'settings:create',
  SETTINGS_UPDATE = 'settings:update',
  SETTINGS_DELETE = 'settings:delete',

  // Window
  WINDOW_CREATE = 'window:create',
  WINDOW_CLOSE = 'window:close',
  WINDOW_MINIMIZE = 'window:minimize',
  WINDOW_MAXIMIZE = 'window:maximize',
  WINDOW_RESTORE = 'window:restore',
  WINDOW_FOCUS = 'window:focus',
  WINDOW_BLUR = 'window:blur',
  WINDOW_CLOSE_ALL = 'window:closeAll',

  // OS Service
  OS_SERVICE_GET = 'osService:get',
  OS_SERVICE_CREATE = 'osService:create',
  OS_SERVICE_UPDATE = 'osService:update',
  OS_SERVICE_DELETE = 'osService:delete',

  // HTTP Service
  HTTP_SERVICE_GET = 'httpService:get',
  HTTP_SERVICE_POST = 'httpService:post',

  // Event
  EVENT_EMIT = 'event:emit',
  EVENT_ON = 'event:on',

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
