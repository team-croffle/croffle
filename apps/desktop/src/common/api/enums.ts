import type {
  AppCloseBehavior,
  AppEventType,
  AppSettingLanguage,
  AppSettingTheme,
  CalendarTimeFormat,
  CalendarView,
  CalendarWeekStartDay,
  ClipboardDataType,
} from '../enums';

/**
 * The enum objects the preload exposes as `croffle.enums`. Keys mirror the public
 * `EnumsApi` (asserted in assert-public-types.ts); nothing else from common is exposed.
 */
export interface HostEnums {
  AppSettingLanguage: typeof AppSettingLanguage;
  AppSettingTheme: typeof AppSettingTheme;
  CalendarView: typeof CalendarView;
  CalendarWeekStartDay: typeof CalendarWeekStartDay;
  CalendarTimeFormat: typeof CalendarTimeFormat;
  AppCloseBehavior: typeof AppCloseBehavior;
  ClipboardDataType: typeof ClipboardDataType;
  AppEventType: typeof AppEventType;
}
