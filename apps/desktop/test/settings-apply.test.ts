import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import { applySetting, parseNumberInput, PendingPatches } from '../src/common/settings-apply.ts';

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
    const patches = new PendingPatches(APP);
    const { id, patch } = patches.edit('calendar', 'weekNumbers', true);
    assert.deepEqual(patch, { calendar: { weekNumbers: true } });
    assert.equal(patches.view().calendar.weekNumbers, true);
    patches.settle(id);
    assert.deepEqual(patches.view(), APP);
  });

  test('screen edits do not overwrite a change saved elsewhere', () => {
    const patches = new PendingPatches(APP);
    const { patch } = patches.edit('calendar', 'weekNumbers', true);
    // Only the changed field is sent; main merges it into what it stored (dark, from the title bar).
    assert.deepEqual(patch, { calendar: { weekNumbers: true } });
    patches.receive(withTheme('dark'));
    assert.deepEqual(patches.view(), {
      general: { theme: 'dark', language: 'ko' },
      calendar: { weekNumbers: true },
    });
  });

  test('an edit made while saving is kept, even one back to the old value', () => {
    const patches = new PendingPatches(APP);
    const first = patches.edit('calendar', 'weekNumbers', true);
    const second = patches.edit('calendar', 'weekNumbers', false);
    // The first save is stored and reported back before the second finishes.
    patches.receive({ ...APP, calendar: { weekNumbers: true } });
    patches.settle(first.id);
    assert.equal(patches.view().calendar.weekNumbers, false);
    patches.receive(APP);
    patches.settle(second.id);
    assert.equal(patches.view().calendar.weekNumbers, false);
  });

  test('changes saved elsewhere during a save are never taken as edits', () => {
    const patches = new PendingPatches(APP);
    const { id } = patches.edit('calendar', 'weekNumbers', true);
    patches.receive(withTheme('dark'));
    patches.receive(withTheme('light'));
    patches.receive({ ...withTheme('light'), calendar: { weekNumbers: true } });
    patches.settle(id);
    assert.deepEqual(patches.view(), { ...APP, calendar: { weekNumbers: true } });
    const next = patches.edit('general', 'language', 'en');
    assert.deepEqual(next.patch, { general: { language: 'en' } });
  });

  test('a failed save keeps the latest stored values received meanwhile', () => {
    const patches = new PendingPatches(APP);
    const { id } = patches.edit('calendar', 'weekNumbers', true);
    // Another window stored weekNumbers = true while this patch was saving; then this one fails.
    patches.receive({ ...APP, calendar: { weekNumbers: true } });
    patches.settle(id);
    assert.equal(patches.view().calendar.weekNumbers, true);
    assert.deepEqual(patches.stored, { ...APP, calendar: { weekNumbers: true } });
  });

  test('a late reply never brings back an older stored value', () => {
    const patches = new PendingPatches(APP);
    const { id } = patches.edit('calendar', 'weekNumbers', true);
    // Main stored this patch, then the title bar stored dark; the reply to the patch comes last.
    patches.receive({ ...APP, calendar: { weekNumbers: true } });
    patches.receive({ ...withTheme('dark'), calendar: { weekNumbers: true } });
    patches.settle(id);
    assert.deepEqual(patches.view(), { ...withTheme('dark'), calendar: { weekNumbers: true } });
  });

  test('a held-back colour shows on screen until it is sent', () => {
    type Look = { appearance: { accentHue: number }; calendar: { weekNumbers: boolean } };
    const patches = new PendingPatches<Look>({
      appearance: { accentHue: 70 },
      calendar: { weekNumbers: false },
    });
    const colour = patches.edit('appearance', 'accentHue', 120);
    patches.revise(colour.id, 200);
    // Another field changes and is stored before the colour is sent.
    const week = patches.edit('calendar', 'weekNumbers', true);
    patches.receive({ appearance: { accentHue: 70 }, calendar: { weekNumbers: true } });
    patches.settle(week.id);
    assert.equal(patches.view().appearance.accentHue, 200);
    assert.deepEqual(patches.revise(colour.id, 200), { appearance: { accentHue: 200 } });
  });
});
