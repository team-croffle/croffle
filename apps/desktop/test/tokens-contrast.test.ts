/**
 * Checks the colour tokens in src/renderer/src/index.css against the rules of the design system
 * "croffle" (apps/desktop/DESIGN.md): text pairs hold 4.5:1, control edges 3:1, the accent hue
 * setting keeps both for every hue, and the extension-visible `--croffle-*` names stay defined.
 */
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, test } from 'node:test';

import { contrast, oklchToSrgb, srgbToOklchL, type Rgb } from './support/color.ts';

const ROOT = join(import.meta.dirname, '..');
const CSS = readFileSync(join(ROOT, 'src/renderer/src/index.css'), 'utf8');

type Vars = Map<string, string>;

/** One line, single spaces, no space just inside parentheses (the formatter wraps long values). */
const normalize = (v: string) =>
  v.replace(/\s+/g, ' ').replace(/\( /g, '(').replace(/ \)/g, ')').trim();

/** Declarations of the first top-level block whose selector is exactly `selector`. */
function block(selector: string): Vars {
  const re = new RegExp(`(^|\\n)${selector.replace('.', '\\.')}\\s*\\{([\\s\\S]*?)\\n\\}`);
  const m = re.exec(CSS);
  assert.ok(m, `block ${selector} not found`);
  const body = m[2]!.replace(/\/\*[\s\S]*?\*\//g, '');
  const vars: Vars = new Map();
  for (const d of body.split(';')) {
    const i = d.indexOf(':');
    if (i < 0) {
      continue;
    }
    const name = d.slice(0, i).trim();
    if (name.startsWith('--')) {
      vars.set(name, normalize(d.slice(i + 1)));
    }
  }
  return vars;
}

const THEME_INLINE = (() => {
  const m = /@theme inline\s*\{([\s\S]*?)\n\}/.exec(CSS);
  assert.ok(m, '@theme inline not found');
  return m[1]!;
})();

const LIGHT = block(':root');
const DARK_OWN = block('.dark');
/** `.dark` overrides `:root`; anything it does not set comes from `:root`. */
const DARK: Vars = new Map([...LIGHT, ...DARK_OWN]);

/** Resolves a token to sRGB, following `var()` and evaluating `calc(var(--croffle-accent-hue) + n)`. */
function resolve(vars: Vars, name: string, hue: number, depth = 0): Rgb {
  assert.ok(depth < 10, `var loop at ${name}`);
  const raw = vars.get(name);
  assert.ok(raw !== undefined, `${name} is not defined`);
  const v = raw.trim();
  const ref = /^var\((--[\w-]+)\)$/.exec(v);
  if (ref) {
    return resolve(vars, ref[1]!, hue, depth + 1);
  }
  const hex = /^#([0-9a-f]{6})$/i.exec(v);
  if (hex) {
    return [0, 2, 4].map((i) => Number.parseInt(hex[1]!.slice(i, i + 2), 16) / 255) as Rgb;
  }
  const ok = /^oklch\(\s*(calc\(.+?\)\)\)|[\d.]+)\s+([\d.]+)\s+(.+?)\s*(\/\s*[\d.]+)?\)$/.exec(v);
  if (ok) {
    return oklchToSrgb(lightnessOf(ok[1]!, hue), Number(ok[2]), hueOf(ok[3]!, hue));
  }
  throw new Error(`cannot resolve ${name}: ${v}`);
}

/** The accent dip: `calc(L - k * max(0, cos((var(--croffle-accent-hue) - c) * 1deg)))`. */
const DIP =
  /^calc\(([\d.]+) - ([\d.]+) \* max\(0, cos\(\(var\(--croffle-accent-hue\) - ([\d.]+)\) \* 1deg\)\)\)$/;

function lightnessOf(expr: string, hue: number): number {
  const m = DIP.exec(expr);
  if (!m) {
    return Number(expr);
  }
  const [base, dip, center] = [Number(m[1]), Number(m[2]), Number(m[3])];
  return base - dip * Math.max(0, Math.cos(((hue - center) * Math.PI) / 180));
}

function hueOf(expr: string, hue: number): number {
  if (expr === 'var(--croffle-accent-hue)') {
    return hue;
  }
  const m = /^calc\(var\(--croffle-accent-hue\)\s*([+-])\s*([\d.]+)\)$/.exec(expr);
  if (m) {
    return hue + (m[1] === '+' ? 1 : -1) * Number(m[2]);
  }
  return Number(expr);
}

const DEFAULT_HUE = Number(LIGHT.get('--croffle-accent-hue'));

/** oklch lightness of a Croffle Dark token. */
const darkL = (n: string) => srgbToOklchL(resolve(DARK, n, DEFAULT_HUE));
const THEMES: [string, Vars][] = [
  ['light', LIGHT],
  ['dark', DARK],
];

function ratio(vars: Vars, fg: string, bg: string, hue = DEFAULT_HUE): number {
  return contrast(resolve(vars, fg, hue), resolve(vars, bg, hue));
}

/** Text tokens and the grounds they are placed on. */
const TEXT_PAIRS: [string, string][] = [
  ...['--foreground', '--muted-foreground', '--accent-foreground'].flatMap((fg) =>
    ['--background', '--card', '--popover', '--sidebar', '--muted'].map(
      (bg) => [fg, bg] as [string, string],
    ),
  ),
  ['--primary-foreground', '--primary'],
  ['--destructive', '--background'],
  ['--destructive', '--card'],
  ['--destructive-foreground', '--destructive'],
  // Status colours used as lettering (warning notices, status words).
  ...['--success', '--warning', '--info'].flatMap((fg) =>
    ['--background', '--card', '--sidebar'].map((bg) => [fg, bg] as [string, string]),
  ),
  ['--success-foreground', '--success'],
  ['--warning-foreground', '--warning'],
  ['--info-foreground', '--info'],
];

/** The 16 `--croffle-*` names of desktop 1.3.1, which extensions may read. */
const CROFFLE_131 = [
  'bg',
  'sidebar',
  'sidebar-content',
  'sidebar-content-disabled',
  'sidebar-selected',
  'primary',
  'hover',
  'text',
  'border',
  'red',
  'blue',
  'green',
  'today-bg',
  'day-select-bg',
  'muted',
  'disabled',
].map((n) => `--croffle-${n}`);

/** Non-accent `--croffle-*` names and the token each one aliases (design system table). */
const CROFFLE_ALIASES: Record<string, string> = {
  '--croffle-bg': '--background',
  '--croffle-sidebar': '--sidebar',
  '--croffle-sidebar-content': '--card',
  '--croffle-sidebar-content-disabled': '--muted',
  '--croffle-sidebar-selected': '--accent',
  '--croffle-primary': '--primary',
  '--croffle-primary-strong': '--accent-foreground',
  '--croffle-hover': '--accent',
  '--croffle-text': '--muted-foreground',
  '--croffle-border': '--border',
  '--croffle-red': '--destructive',
  '--croffle-blue': '--info',
  '--croffle-green': '--success',
  '--croffle-muted': '--muted-foreground',
  '--croffle-disabled': '--muted',
};

/** Tokens allowed to follow the accent hue, directly or through an alias. */
const ACCENT_FAMILY = new Set([
  '--primary',
  '--ring',
  '--accent-foreground',
  '--croffle-today-bg',
  '--croffle-primary',
  '--croffle-primary-strong',
  '--sidebar-primary',
  '--sidebar-ring',
  '--sidebar-accent-foreground',
  '--chart-1',
]);

function followsHue(vars: Vars, name: string, depth = 0): boolean {
  const v = vars.get(name) ?? '';
  if (v.includes('--croffle-accent-hue')) {
    return true;
  }
  const ref = /^var\((--[\w-]+)\)$/.exec(v.trim());
  return ref !== null && depth < 10 && followsHue(vars, ref[1]!, depth + 1);
}

describe('design system tokens', () => {
  test('text pairs hold 4.5:1 in both themes', () => {
    for (const [theme, vars] of THEMES) {
      for (const [fg, bg] of TEXT_PAIRS) {
        const r = ratio(vars, fg, bg);
        assert.ok(r >= 4.5, `${theme}: ${fg} on ${bg} is ${r.toFixed(2)}:1`);
      }
    }
  });

  test('input holds 3:1 against background card sidebar in both themes', () => {
    for (const [theme, vars] of THEMES) {
      for (const bg of ['--background', '--card', '--sidebar']) {
        const r = ratio(vars, '--input', bg);
        assert.ok(r >= 3, `${theme}: --input on ${bg} is ${r.toFixed(2)}:1`);
      }
    }
  });

  test('accent hue sweep keeps accent pairs for every hue 0..359', () => {
    for (const [theme, vars] of THEMES) {
      for (let hue = 0; hue < 360; hue++) {
        const p = ratio(vars, '--primary-foreground', '--primary', hue);
        assert.ok(p >= 4.5, `${theme} hue ${hue}: primary-foreground on primary ${p.toFixed(2)}:1`);
        for (const bg of ['--background', '--card']) {
          const s = ratio(vars, '--croffle-primary-strong', bg, hue);
          assert.ok(s >= 4.5, `${theme} hue ${hue}: primary-strong on ${bg} ${s.toFixed(2)}:1`);
        }
      }
    }
  });

  test('non-accent tokens do not use the accent hue', () => {
    for (const [theme, vars] of THEMES) {
      const tinted = [...vars.keys()].filter(
        (n) => n !== '--croffle-accent-hue' && followsHue(vars, n),
      );
      const stray = tinted.filter((n) => !ACCENT_FAMILY.has(n));
      assert.deepEqual(stray, [], `${theme}: ${stray.join(', ')} follow the accent hue`);
      for (const n of ['--primary', '--ring', '--accent-foreground', '--croffle-today-bg']) {
        assert.ok(tinted.includes(n), `${theme}: ${n} does not follow the accent hue`);
      }
    }
  });

  test('every croffle variable of 1.3.1 is still defined', () => {
    const names = [...CROFFLE_131, '--croffle-accent-hue', '--croffle-primary-strong'];
    for (const n of names) {
      assert.ok(LIGHT.has(n), `:root misses ${n}`);
      if (n !== '--croffle-accent-hue') {
        assert.ok(DARK_OWN.has(n), `.dark misses ${n}`);
      }
    }
  });

  test('croffle variables alias semantic tokens and default accent hue is 70', () => {
    for (const [theme, vars] of THEMES) {
      for (const [alias, token] of Object.entries(CROFFLE_ALIASES)) {
        assert.equal(vars.get(alias), `var(${token})`, `${theme}: ${alias}`);
      }
    }
    assert.equal(DEFAULT_HUE, 70);
    const appearance = readFileSync(join(ROOT, 'src/renderer/src/utils/appearance.ts'), 'utf8');
    assert.match(appearance, /export const DEFAULT_ACCENT_HUE = 70;/);
    const settings = readFileSync(join(ROOT, 'src/main/setting/setting-service.ts'), 'utf8');
    assert.match(settings, /accentHue: 70,/);
    // --primary in Croffle Light: the system's lightness at the default hue, the same formula and
    // chroma as the settings preview (appearance.ts).
    const primary = /^oklch\((calc\(.+?\)\)\)) ([\d.]+) var\(--croffle-accent-hue\)\)$/.exec(
      LIGHT.get('--primary') ?? '',
    );
    assert.ok(primary, '--primary is not oklch(calc(…) C var(--croffle-accent-hue))');
    const dip = DIP.exec(primary[1]!);
    assert.ok(dip, '--primary lightness is not the accent dip formula');
    assert.equal(
      lightnessOf(primary[1]!, DEFAULT_HUE),
      0.5469,
      'default hue is not the system value',
    );
    assert.match(appearance, new RegExp(`ACCENT_LIGHTNESS = ${dip[1]};`));
    assert.match(appearance, new RegExp(`ACCENT_DIP = ${dip[2]};`));
    assert.match(appearance, new RegExp(`ACCENT_DIP_CENTER = ${dip[3]};`));
    assert.match(appearance, new RegExp(`PREVIEW_CHROMA = ${primary[2]};`));
  });

  test('dark surfaces rise: background < card = popover', () => {
    assert.ok(darkL('--background') < darkL('--card'), 'background is not darker than card');
    assert.equal(darkL('--card'), darkL('--popover'));
  });

  test('theme inline exposes status colors radius steps and 4 shadows', () => {
    assert.equal(LIGHT.get('--radius'), '0.75rem');
    const radius: Record<string, string> = {
      sm: 'calc(var(--radius) - 4px)',
      md: 'calc(var(--radius) - 2px)',
      lg: 'var(--radius)',
      xl: 'calc(var(--radius) + 4px)',
      '2xl': 'calc(var(--radius) + 8px)',
      '3xl': 'calc(var(--radius) + 12px)',
    };
    for (const [step, value] of Object.entries(radius)) {
      assert.ok(THEME_INLINE.includes(`--radius-${step}: ${value};`), `--radius-${step}`);
    }
    for (const c of ['success', 'warning', 'info']) {
      for (const n of [c, `${c}-foreground`]) {
        assert.ok(THEME_INLINE.includes(`--color-${n}: var(--${n});`), `--color-${n}`);
      }
    }
    assert.ok(
      THEME_INLINE.includes('--color-croffle-primary-strong: var(--croffle-primary-strong);'),
    );
    for (const s of ['xs', 'sm', 'surface', 'lg']) {
      assert.ok(THEME_INLINE.includes(`--shadow-${s}: var(--elevation-${s});`), `--shadow-${s}`);
      assert.ok(
        LIGHT.has(`--elevation-${s}`) && DARK_OWN.has(`--elevation-${s}`),
        `elevation ${s}`,
      );
    }
  });

  test('pretendard 400 500 600 700 and fredoka 600 700 font faces are declared', () => {
    const faces = [...CSS.matchAll(/@font-face\s*\{([\s\S]*?)\}/g)].map((m) => m[1]!);
    const declared = (family: string, weight: number) =>
      faces.find((f) => f.includes(`'${family}'`) && f.includes(`font-weight: ${weight};`));
    for (const [family, weights] of [
      ['Pretendard', [400, 500, 600, 700]],
      ['Fredoka', [600, 700]],
    ] as const) {
      for (const w of weights) {
        const face = declared(family, w);
        assert.ok(face, `${family} ${w} is not declared`);
        const woff2 = /url\('\.\/(assets\/fonts\/woff2\/[\w-]+\.woff2)'\)/.exec(face);
        assert.ok(woff2, `${family} ${w} has no woff2 source`);
        assert.ok(existsSync(join(ROOT, 'src/renderer/src', woff2[1]!)), `${woff2[1]} is missing`);
      }
    }
  });
});
