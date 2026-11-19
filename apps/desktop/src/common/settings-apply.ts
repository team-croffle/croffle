/**
 * Settings apply as soon as they change (no Save/Cancel). One change goes through here:
 * an invalid value is not saved; a failed save leaves the stored value on screen.
 */
export type SettingApplyResult<T> =
  | { ok: true; value: T }
  /** Not saved because the value is invalid; `value` is what is stored. */
  | { ok: false; reason: 'invalid'; message: string; value: T }
  /** The save failed; `value` is what is stored, to put back on screen. */
  | { ok: false; reason: 'failed'; error: unknown; value: T };

export async function applySetting<T>(
  stored: T,
  next: T,
  save: (value: T) => Promise<T>,
  validate: (value: T) => string | null = () => null,
): Promise<SettingApplyResult<T>> {
  const message = validate(next);
  if (message !== null) {
    return { ok: false, reason: 'invalid', message, value: stored };
  }
  try {
    return { ok: true, value: await save(next) };
  } catch (error) {
    return { ok: false, reason: 'failed', error, value: stored };
  }
}

/** A number typed into a settings field: a finite number, or null when it is not one. */
export function parseNumberInput(raw: string): number | null {
  if (raw.trim() === '') {
    return null;
  }
  const value = Number(raw);
  return Number.isFinite(value) ? value : null;
}
