/**
 * Storing settings, without Electron (testable): merge a patch per section, then write; the
 * stored settings and their revision change only when the write succeeded.
 */
export type SettingsState<T> = { settings: T; revision: number };

type Sections = {
  general: object;
  appearance?: object;
  calendar: object;
  notifications: object;
};

/** `patch` laid over `current`, section by section (main's partial update). */
export function mergeSettingsPatch<T extends Sections>(
  current: T,
  patch: Partial<T>,
  defaultAppearance: NonNullable<T['appearance']>,
): T {
  return {
    ...current,
    ...patch,
    general: { ...current.general, ...patch.general },
    appearance: { ...defaultAppearance, ...current.appearance, ...patch.appearance },
    calendar: { ...current.calendar, ...patch.calendar },
    notifications: { ...current.notifications, ...patch.notifications },
  };
}

/** Writes `next`; a failed write throws and leaves `state` as it was. */
export function commitSettings<T>(
  state: SettingsState<T>,
  next: T,
  write: (settings: T) => void,
): SettingsState<T> {
  write(next);
  return { settings: next, revision: state.revision + 1 };
}
