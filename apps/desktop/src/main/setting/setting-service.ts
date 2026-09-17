import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

import {
  AppCloseBehavior,
  AppSettingLanguage,
  AppSettingTheme,
  CalendarTimeFormat,
  CalendarView,
  CalendarWeekStartDay,
} from '@croffledev/common';
import type { AppSettings } from '@croffledev/croffle-types';
import { is } from '@electron-toolkit/utils';
import { app } from 'electron';

import { logger } from '../logger';
import { applyPersisted } from './setting-applies';

const DEFAULT_SETTINGS: AppSettings = {
  general: {
    language: AppSettingLanguage.EN,
    theme: AppSettingTheme.SYSTEM,
    autoUpdate: true,
    startOnSystemBoot: false,
    startMinimized: false,
    closeBehavior: AppCloseBehavior.ASK,
  },
  appearance: {
    accentHue: 69.8,
  },
  calendar: {
    defaultView: CalendarView.MONTH,
    weekStartDay: CalendarWeekStartDay.SUNDAY,
    showWeekNumbers: false,
    timeFormat: CalendarTimeFormat.H24,
  },
  notifications: {
    enabled: true,
    defaultReminderMinutes: 10,
  },
};

class SettingService {
  private settings: AppSettings;
  private filePath: string;

  constructor() {
    this.filePath = is.dev
      ? path.join(process.cwd(), 'dev/settings.json')
      : path.join(app.getPath('userData'), 'settings.json');
    this.settings = this.loadSettings();
  }

  private loadSettings(): AppSettings {
    try {
      if (!existsSync(this.filePath)) {
        logger.info('Settings', 'Settings file does not exist. Creating default settings.');
        this.saveSettings(DEFAULT_SETTINGS);
        return DEFAULT_SETTINGS;
      }

      const data = readFileSync(this.filePath, 'utf-8');
      const parsed = JSON.parse(data) as Partial<AppSettings>;

      const merged: AppSettings = {
        ...DEFAULT_SETTINGS,
        ...parsed,
        general: { ...DEFAULT_SETTINGS.general, ...parsed.general },
        appearance: { ...DEFAULT_SETTINGS.appearance, ...parsed.appearance },
        calendar: { ...DEFAULT_SETTINGS.calendar, ...parsed.calendar },
        notifications: { ...DEFAULT_SETTINGS.notifications, ...parsed.notifications },
      };
      // 로그인 항목 등록은 여기서 하지 않는다. 이 생성자는 app ready 전에 돌아
      // AppUserModelId가 아직 기본값이라 잘못된 이름으로 등록된다. index.ts whenReady에서 한 번 등록.
      return merged;
    } catch (err) {
      logger.error('Settings', 'Failed to load settings, using default settings.', err);
      return DEFAULT_SETTINGS;
    }
  }

  private saveSettings(settings: AppSettings): void {
    try {
      writeFileSync(this.filePath, JSON.stringify(settings, null, 2), 'utf-8');
      this.settings = settings;
      logger.info('Settings', 'Settings saved successfully.');
    } catch (err) {
      logger.error('Settings', 'Failed to save settings.', err);
    }
  }

  // --- API ---
  public get(): AppSettings {
    return this.settings;
  }

  public getOf(key: string): AppSettings[keyof AppSettings] {
    // validate key
    if (!(key in this.settings)) {
      throw new Error(`[Settings] Key "${key}" does not exist in settings.`);
    }

    const typedKey: keyof AppSettings = key as keyof AppSettings;
    return this.settings[typedKey];
  }

  public update(partialSettings: Partial<AppSettings>): AppSettings {
    const updatedSettings = {
      ...this.settings,
      ...partialSettings,
      general: {
        ...this.settings.general,
        ...partialSettings.general,
      },
      appearance: {
        ...DEFAULT_SETTINGS.appearance,
        ...this.settings.appearance,
        ...partialSettings.appearance,
      },
      calendar: {
        ...this.settings.calendar,
        ...partialSettings.calendar,
      },
      notifications: {
        ...this.settings.notifications,
        ...partialSettings.notifications,
      },
    };

    this.saveSettings(updatedSettings);
    applyPersisted(updatedSettings);
    return updatedSettings;
  }
}

export const settingService = new SettingService();
