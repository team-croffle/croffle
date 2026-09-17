import {
  AppCloseBehavior,
  AppSettingLanguage,
  AppSettingTheme,
  CalendarTimeFormat,
  CalendarView,
  CalendarWeekStartDay,
} from '@croffledev/common';
import type { AppSettings } from '@croffledev/croffle-types';

function isValidEnum<T extends object>(value: unknown, enumObj: T): value is T[keyof T] {
  const validValues = Object.values(enumObj);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return validValues.includes(value as any);
}

export const validateSettings = (settings: Partial<AppSettings>): void => {
  if (settings.general) {
    const { language, theme, closeBehavior } = settings.general;

    if (language && !isValidEnum(language, AppSettingLanguage)) {
      throw new Error(
        `Invalid language setting: ${language}. Allowed values are: ${Object.values(AppSettingLanguage).join(', ')}`,
      );
    }

    if (theme && !isValidEnum(theme, AppSettingTheme)) {
      throw new Error(
        `Invalid theme setting: ${theme}. Allowed values are: ${Object.values(AppSettingTheme).join(', ')}`,
      );
    }

    if (closeBehavior && !isValidEnum(closeBehavior, AppCloseBehavior)) {
      throw new Error(
        `Invalid closeBehavior setting: ${closeBehavior}. Allowed values are: ${Object.values(AppCloseBehavior).join(', ')}`,
      );
    }
  }

  if (settings.appearance) {
    const { accentHue } = settings.appearance;
    if (accentHue !== undefined) {
      if (
        typeof accentHue !== 'number' ||
        Number.isNaN(accentHue) ||
        accentHue < 0 ||
        accentHue > 360
      ) {
        throw new Error(
          `Invalid accentHue setting: ${String(accentHue)}. Expected a number between 0 and 360.`,
        );
      }
    }
  }

  if (settings.calendar) {
    const { defaultView, weekStartDay, timeFormat } = settings.calendar || {};
    if (defaultView && !isValidEnum(defaultView, CalendarView)) {
      throw new Error(
        `Invalid defaultView setting: ${defaultView}. Allowed values are: ${Object.values(CalendarView).join(', ')}`,
      );
    }

    if (weekStartDay && !isValidEnum(weekStartDay, CalendarWeekStartDay)) {
      throw new Error(
        `Invalid weekStartDay setting: ${weekStartDay}. Allowed values are: ${Object.values(CalendarWeekStartDay).join(', ')}`,
      );
    }

    if (timeFormat && !isValidEnum(timeFormat, CalendarTimeFormat)) {
      throw new Error(
        `Invalid timeFormat setting: ${timeFormat}. Allowed values are: ${Object.values(CalendarTimeFormat).join(', ')}`,
      );
    }
  }
};
