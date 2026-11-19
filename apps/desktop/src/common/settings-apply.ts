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

/**
 * App settings while they apply immediately. Each control sends only the field it changed
 * (main merges a patch into what it has stored); the screen shows the latest stored settings
 * with the patches that have not finished laid on top.
 * - Stored settings come in one way only, `receive` (main's SETTINGS_UPDATE, sent before the
 *   save's reply, or a read): a reply never brings back an older stored value.
 * - A change saved elsewhere only replaces `stored`; it is never taken for an edit.
 * - A failed patch is dropped, so its field shows the latest stored value again.
 * - An edit made while an earlier patch is saving is its own patch and stays on screen.
 * - A patch may be held back (the colour picker) and revised until it is sent; it shows meanwhile.
 */
export class PendingPatches<T extends object> {
  private pending: { id: number; section: keyof T; field: string; value: unknown }[] = [];
  private nextId = 1;
  public stored: T;

  public constructor(stored: T) {
    this.stored = stored;
  }

  /** Records an edit; returns its id and the patch to send. */
  public edit<S extends keyof T>(
    section: S,
    field: keyof T[S] & string,
    value: T[S][keyof T[S]],
  ): { id: number; patch: Partial<T> } {
    const id = this.nextId++;
    this.pending.push({ id, section, field, value });
    return { id, patch: this.patchOf(id) };
  }

  /** Changes the value of a patch not sent yet; returns the patch as it is now. */
  public revise(id: number, value: unknown): Partial<T> {
    const entry = this.pending.find((p) => p.id === id);
    if (entry) {
      entry.value = value;
    }
    return this.patchOf(id);
  }

  public has(id: number): boolean {
    return this.pending.some((p) => p.id === id);
  }

  /** Whether a patch of this field is still unfinished. */
  public pendingFor(section: keyof T, field: string): boolean {
    return this.pending.some((p) => p.section === section && p.field === field);
  }

  /** The patch finished (saved or failed); its field now shows what is stored. */
  public settle(id: number): void {
    this.pending = this.pending.filter((p) => p.id !== id);
  }

  /** The latest stored settings arrived. */
  public receive(latest: T): void {
    this.stored = latest;
  }

  /** What the screen shows: the latest stored settings with unfinished patches on top. */
  public view(): T {
    const out = JSON.parse(JSON.stringify(this.stored)) as T;
    for (const p of this.pending) {
      (out[p.section] as Record<string, unknown>)[p.field] = p.value;
    }
    return out;
  }

  private patchOf(id: number): Partial<T> {
    const entry = this.pending.find((p) => p.id === id);
    if (!entry) {
      return {};
    }
    return { [entry.section]: { [entry.field]: entry.value } } as unknown as Partial<T>;
  }
}

/** A theme or colour shown before it is stored (the settings screen's unsent edits). */
export type LookPreview = { theme?: string; accentHue?: number };

/**
 * The look the app shows: the stored theme and accent, unless the settings screen previews
 * another one. Applied in one place (the app settings store) for every stored-settings event,
 * so the order in which listeners run cannot drop a preview.
 */
export function lookOf<Theme extends string>(
  stored: { theme: Theme; accentHue: number },
  preview: LookPreview,
): { theme: Theme; accentHue: number } {
  return {
    theme: (preview.theme as Theme | undefined) ?? stored.theme,
    accentHue: preview.accentHue ?? stored.accentHue,
  };
}
