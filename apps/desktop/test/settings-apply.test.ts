import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import {
  applySetting,
  mergeEdits,
  parseNumberInput,
  SettingsDraft,
} from '../src/common/settings-apply.ts';

type Prefs = { weekNumbers: boolean; days: number };
const STORED: Prefs = { weekNumbers: false, days: 7 };

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

    const saved = await applySetting(STORED, { ...STORED, weekNumbers: true }, async (v) => v);
    assert.deepEqual(saved, { ok: true, value: { ...STORED, weekNumbers: true } });
  });

  test('screen edits do not overwrite a change saved elsewhere', () => {
    const base = { general: { theme: 'light', language: 'ko' }, calendar: { weekNumbers: false } };
    // The title bar saved dark meanwhile; the screen changed week numbers only.
    const latest = { ...base, general: { ...base.general, theme: 'dark' } };
    const edited = { ...base, calendar: { weekNumbers: true } };
    assert.deepEqual(mergeEdits(latest, base, edited), {
      general: { theme: 'dark', language: 'ko' },
      calendar: { weekNumbers: true },
    });
    // No edits: the latest stored values come through unchanged.
    assert.deepEqual(mergeEdits(latest, base, base), latest);
  });

  test('an edit made while saving is kept, even one back to the old value', () => {
    const draft = new SettingsDraft({ weekNumbers: false, theme: 'light' });
    draft.screen = { ...draft.screen, weekNumbers: true };
    const saving = draft.beginSave();
    assert.deepEqual(saving, { weekNumbers: true, theme: 'light' });
    // While saving, the user turns it back off; then the save's own SETTINGS_UPDATE arrives.
    draft.screen = { ...draft.screen, weekNumbers: false };
    draft.rebase({ weekNumbers: true, theme: 'light' });
    assert.equal(draft.screen.weekNumbers, false);
    draft.endSave(saving!, { weekNumbers: true, theme: 'light' });
    assert.equal(draft.screen.weekNumbers, false);
    // The next save stores the latest edit.
    assert.deepEqual(draft.beginSave(), { weekNumbers: false, theme: 'light' });
  });

  test('a failed save puts back only the fields it tried to save', () => {
    const draft = new SettingsDraft({ weekNumbers: false, theme: 'light' });
    draft.screen = { ...draft.screen, weekNumbers: true };
    const saving = draft.beginSave()!;
    draft.screen = { ...draft.screen, theme: 'dark' };
    draft.endSave(saving, draft.stored);
    assert.deepEqual(draft.screen, { weekNumbers: false, theme: 'dark' });
    assert.deepEqual(draft.beginSave(), { weekNumbers: false, theme: 'dark' });
  });
});
