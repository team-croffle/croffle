import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import { commitSettings, mergeSettingsPatch } from '../src/main/setting/settings-commit.ts';

type S = {
  general: { theme: string; language: string };
  appearance: { accentHue: number };
  calendar: { weekNumbers: boolean };
  notifications: { enabled: boolean };
};
const CURRENT: S = {
  general: { theme: 'light', language: 'ko' },
  appearance: { accentHue: 70 },
  calendar: { weekNumbers: false },
  notifications: { enabled: true },
};

describe('main stores settings', () => {
  test('a one-field patch is merged into its section', () => {
    const next = mergeSettingsPatch(CURRENT, { general: { theme: 'dark' } } as Partial<S>, {
      accentHue: 70,
    });
    assert.deepEqual(next, { ...CURRENT, general: { theme: 'dark', language: 'ko' } });
  });

  test('a failed write keeps the stored settings and revision', () => {
    const state = { settings: CURRENT, revision: 4 };
    const next = { ...CURRENT, appearance: { accentHue: 200 } };
    assert.throws(
      () =>
        commitSettings(state, next, () => {
          throw Object.assign(new Error('no space left on device'), { code: 'ENOSPC' });
        }),
      /no space left/,
    );
    assert.deepEqual(state, { settings: CURRENT, revision: 4 });
    const written: S[] = [];
    assert.deepEqual(
      commitSettings(state, next, (s) => written.push(s)),
      { settings: next, revision: 5 },
    );
    assert.equal(written.length, 1);
  });
});
