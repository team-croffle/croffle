export declare enum AppSettingLanguage {
  KO = 'ko',
  EN = 'en',
}

export declare enum AppSettingTheme {
  LIGHT = 'light',
  DARK = 'dark',
  SYSTEM = 'system',
}

export declare enum CalendarView {
  DAY = 'day',
  WEEK = 'week',
  MONTH = 'month',
  YEAR = 'year',
}

export declare enum CalendarWeekStartDay {
  SUNDAY = 'sunday',
  MONDAY = 'monday',
}

export declare enum CalendarTimeFormat {
  H12 = '12h',
  H24 = '24h',
}

/** What the window close button does. `ASK` shows a one-time prompt and stores the answer. */
export declare enum AppCloseBehavior {
  ASK = 'ask',
  TRAY = 'tray',
  QUIT = 'quit',
}

export type AppSettings = {
  general: {
    language: AppSettingLanguage;
    theme: AppSettingTheme;
    autoUpdate: boolean;
    startOnSystemBoot: boolean;
    startMinimized: boolean;
    /** Close button behavior. Defaults to `ask` until the user answers the first-close prompt. */
    closeBehavior: AppCloseBehavior;
  };
  appearance: {
    /** Accent color hue in degrees (0–360). Drives --croffle-accent-hue. */
    accentHue: number;
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
};
