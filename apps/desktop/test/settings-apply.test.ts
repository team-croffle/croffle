import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import {
  applySetting,
  lookOf,
  parseNumberInput,
  PendingPatches,
} from '../src/common/settings-apply.ts';

type Prefs = { weekNumbers: boolean; days: number };
const STORED: Prefs = { weekNumbers: false, days: 7 };

type App = { general: { theme: string; language: string }; calendar: { weekNumbers: boolean } };
const APP: App = { general: { theme: 'light', language: 'ko' }, calendar: { weekNumbers: false } };
const withTheme = (theme: string): App => ({ ...APP, general: { ...APP.general, theme } });

describe('settings apply immediately', () => {
  test('invalid input is rejected without calling save', async () => {
    let saves = 0;
    const result = await applySetting(
      STORED,
      { ...STORED, days: Number.NaN },
      async (v) => {
        saves += 1;
        return v;
      },
      (v) => (Number.isFinite(v.days) ? null : 'not a number'),
    );
    assert.equal(saves, 0);
    assert.deepEqual(result, {
      ok: false,
      reason: 'invalid',
      message: 'not a number',
      value: STORED,
    });
    assert.equal(parseNumberInput('12abc'), null);
    assert.equal(parseNumberInput(''), null);
    assert.equal(parseNumberInput('Infinity'), null);
    assert.equal(parseNumberInput(' 30 '), 30);
  });

  test('settings save failure restores the stored value', async () => {
    const failure = new Error('disk full');
    const result = await applySetting(STORED, { ...STORED, weekNumbers: true }, async () => {
      throw failure;
    });
    assert.equal(result.ok, false);
    assert.deepEqual(result.value, STORED);
    assert.equal(result.ok === false && result.reason === 'failed' && result.error, failure);

    // In the settings screen a failed patch is dropped: the field shows the stored value again.
    const patches = new PendingPatches(APP, 1);
    const { id, patch } = patches.edit('calendar', 'weekNumbers', true);
    assert.deepEqual(patch, { calendar: { weekNumbers: true } });
    assert.equal(patches.view().calendar.weekNumbers, true);
    patches.settle(id);
    assert.deepEqual(patches.view(), APP);
  });

  test('screen edits do not overwrite a change saved elsewhere', () => {
    const patches = new PendingPatches(APP, 1);
    const { patch } = patches.edit('calendar', 'weekNumbers', true);
    // Only the changed field is sent; main merges it into what it stored (dark, from the title bar).
    assert.deepEqual(patch, { calendar: { weekNumbers: true } });
    patches.receive(withTheme('dark'), 2);
    assert.deepEqual(patches.view(), {
      general: { theme: 'dark', language: 'ko' },
      calendar: { weekNumbers: true },
    });
  });

  test('an edit made while saving is kept, even one back to the old value', () => {
    const patches = new PendingPatches(APP, 1);
    const first = patches.edit('calendar', 'weekNumbers', true);
    const second = patches.edit('calendar', 'weekNumbers', false);
    patches.confirm(first.id, 2);
    patches.receive({ ...APP, calendar: { weekNumbers: true } }, 2);
    assert.equal(patches.view().calendar.weekNumbers, false);
    patches.confirm(second.id, 3);
    patches.receive(APP, 3);
    assert.equal(patches.view().calendar.weekNumbers, false);
    assert.deepEqual(patches.view(), APP);
  });

  test('changes saved elsewhere during a save are never taken as edits', () => {
    const patches = new PendingPatches(APP, 1);
    const { id } = patches.edit('calendar', 'weekNumbers', true);
    patches.receive(withTheme('dark'), 2);
    patches.receive(withTheme('light'), 3);
    patches.confirm(id, 4);
    patches.receive({ ...withTheme('light'), calendar: { weekNumbers: true } }, 4);
    assert.deepEqual(patches.view(), { ...APP, calendar: { weekNumbers: true } });
    const next = patches.edit('general', 'language', 'en');
    assert.deepEqual(next.patch, { general: { language: 'en' } });
  });

  test('a failed save keeps the latest stored values received meanwhile', () => {
    const patches = new PendingPatches(APP, 1);
    const { id } = patches.edit('calendar', 'weekNumbers', true);
    // Another window stored weekNumbers = true while this patch was saving; then this one fails.
    patches.receive({ ...APP, calendar: { weekNumbers: true } }, 2);
    patches.settle(id);
    assert.equal(patches.view().calendar.weekNumbers, true);
    assert.deepEqual(patches.stored, { ...APP, calendar: { weekNumbers: true } });
  });

  test('a late reply never brings back an older stored value', () => {
    const patches = new PendingPatches(APP, 1);
    const { id } = patches.edit('calendar', 'weekNumbers', true);
    patches.receive({ ...withTheme('dark'), calendar: { weekNumbers: true } }, 3);
    // An older broadcast and the reply for revision 2 arrive last.
    assert.equal(patches.receive({ ...APP, calendar: { weekNumbers: true } }, 2), false);
    patches.confirm(id, 2);
    assert.deepEqual(patches.view(), { ...withTheme('dark'), calendar: { weekNumbers: true } });
  });

  test('a patch stays until stored settings of its revision arrive, in either order', () => {
    // Reply first: the patch stays on screen until the broadcast of its revision.
    const a = new PendingPatches(APP, 1);
    const first = a.edit('calendar', 'weekNumbers', true);
    a.confirm(first.id, 2);
    a.receive(withTheme('dark'), 1);
    assert.equal(a.view().calendar.weekNumbers, true);
    a.receive({ ...APP, calendar: { weekNumbers: true } }, 2);
    assert.equal(a.pendingValue('calendar', 'weekNumbers'), null);
    // Another field's broadcast first does not end it either.
    const b = new PendingPatches(APP, 1);
    const colour = b.edit('calendar', 'weekNumbers', true);
    b.receive(withTheme('dark'), 2);
    assert.equal(b.view().calendar.weekNumbers, true);
    b.confirm(colour.id, 3);
    b.receive({ ...withTheme('dark'), calendar: { weekNumbers: true } }, 3);
    assert.equal(b.pendingValue('calendar', 'weekNumbers'), null);
  });

  test('a held-back colour shows on screen until it is sent', () => {
    type Look = { appearance: { accentHue: number }; calendar: { weekNumbers: boolean } };
    const patches = new PendingPatches<Look>(
      { appearance: { accentHue: 70 }, calendar: { weekNumbers: false } },
      1,
    );
    const colour = patches.edit('appearance', 'accentHue', 120);
    patches.revise(colour.id, 200);
    const week = patches.edit('calendar', 'weekNumbers', true);
    patches.confirm(week.id, 2);
    patches.receive({ appearance: { accentHue: 70 }, calendar: { weekNumbers: true } }, 2);
    assert.equal(patches.view().appearance.accentHue, 200);
    assert.deepEqual(patches.revise(colour.id, 200), { appearance: { accentHue: 200 } });
  });

  test('the preview is the newest unfinished value; a later failure falls back to an earlier one', () => {
    type Look = { appearance: { accentHue: number } };
    const patches = new PendingPatches<Look>({ appearance: { accentHue: 70 } }, 1);
    const a = patches.edit('appearance', 'accentHue', 100);
    const b = patches.edit('appearance', 'accentHue', 200);
    assert.deepEqual(patches.pendingValue('appearance', 'accentHue'), { value: 200 });
    patches.settle(b.id);
    assert.deepEqual(patches.pendingValue('appearance', 'accentHue'), { value: 100 });
    assert.equal(patches.view().appearance.accentHue, 100);
    patches.confirm(a.id, 2);
    patches.receive({ appearance: { accentHue: 100 } }, 2);
    assert.equal(patches.pendingValue('appearance', 'accentHue'), null);
  });

  test('a previewed colour or theme survives a settings event', () => {
    const stored = { theme: 'light', accentHue: 70 };
    assert.deepEqual(lookOf(stored, {}), stored);
    // The screen previews hue 200; another field's SETTINGS_UPDATE re-applies the look.
    assert.deepEqual(lookOf(stored, { accentHue: 200 }), { theme: 'light', accentHue: 200 });
    assert.deepEqual(lookOf({ theme: 'dark', accentHue: 70 }, { accentHue: 200 }), {
      theme: 'dark',
      accentHue: 200,
    });
  });
});
