/** The ko and en catalogs (src/common/i18n/locales) stay in step, and ko has no English left. */
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, test } from 'node:test';

type Tree = { [key: string]: string | Tree };

const LOCALES = join(import.meta.dirname, '../src/common/i18n/locales');
const load = (lang: string) =>
  JSON.parse(readFileSync(join(LOCALES, `${lang}.json`), 'utf8')) as Tree;

function flatten(tree: Tree, prefix = ''): Map<string, string> {
  const out = new Map<string, string>();
  for (const [key, value] of Object.entries(tree)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === 'string') {
      out.set(path, value);
    } else {
      for (const [k, v] of flatten(value, path)) {
        out.set(k, v);
      }
    }
  }
  return out;
}

const KO = flatten(load('ko'));
const EN = flatten(load('en'));

/** ko values that are rightly not Korean: a language's own name and an example URL. */
const KO_NON_HANGUL_ALLOWED = ['language.en', 'settings.sync.serverUrlPlaceholder'];

describe('locale catalogs', () => {
  test('ko and en have the same keys', () => {
    const onlyKo = [...KO.keys()].filter((k) => !EN.has(k));
    const onlyEn = [...EN.keys()].filter((k) => !KO.has(k));
    assert.deepEqual({ onlyKo, onlyEn }, { onlyKo: [], onlyEn: [] });
  });

  test('ko has no untranslated English values', () => {
    const english = [...KO]
      .filter(([, value]) => !/[가-힣]/.test(value.replace(/\{[^}]*\}/g, '')))
      .map(([key]) => key)
      .filter((key) => !KO_NON_HANGUL_ALLOWED.includes(key));
    assert.deepEqual(english, []);
  });

  test('help copy in ko uses polite haeyo style', () => {
    const formal = [...KO]
      .filter(([key]) => /^help\.[^.]+\.(description|content)$/.test(key))
      .filter(([, value]) => /습니다|입니다|십시오/.test(value))
      .map(([key]) => key);
    assert.deepEqual(formal, []);
  });
});
