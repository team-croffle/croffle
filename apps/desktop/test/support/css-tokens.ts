/** Reads colour tokens out of index.css for the token tests: blocks, aliases and oklch values. */
import assert from 'node:assert/strict';

import { lightnessOf, oklchToSrgb, type Rgb } from './color.ts';

export type Vars = Map<string, string>;

/** One line, single spaces, no space just inside parentheses (the formatter wraps long values). */
export const normalize = (v: string) =>
  v.replace(/\s+/g, ' ').replace(/\( /g, '(').replace(/ \)/g, ')').trim();

/** Declarations of the first top-level block whose selector is exactly `selector`. */
export function block(css: string, selector: string): Vars {
  const re = new RegExp(`(^|\\n)${selector.replace('.', '\\.')}\\s*\\{([\\s\\S]*?)\\n\\}`);
  const m = re.exec(css);
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

/** Resolves a token to sRGB, following `var()` and evaluating `calc(var(--croffle-accent-hue) + n)`. */
export function resolve(vars: Vars, name: string, hue: number, depth = 0): Rgb {
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

export function hueOf(expr: string, hue: number): number {
  if (expr === 'var(--croffle-accent-hue)') {
    return hue;
  }
  const m = /^calc\(var\(--croffle-accent-hue\)\s*([+-])\s*([\d.]+)\)$/.exec(expr);
  if (m) {
    return hue + (m[1] === '+' ? 1 : -1) * Number(m[2]);
  }
  return Number(expr);
}

/** The token a name finally aliases (`var(--x)` chains), or the name itself. */
export function base(vars: Vars, name: string, depth = 0): string {
  const ref = /^var\((--[\w-]+)\)$/.exec((vars.get(name) ?? '').trim());
  return ref && depth < 10 ? base(vars, ref[1]!, depth + 1) : name;
}

export function followsHue(vars: Vars, name: string): boolean {
  return (vars.get(base(vars, name)) ?? '').includes('--croffle-accent-hue');
}

/** `oklch(L C calc(var(--croffle-accent-hue) + d))` → [L, C, d]. */
export function neutralParts(value: string | undefined): [number, number, string] | null {
  const m = /^oklch\(([\d.]+) ([\d.]+) calc\(var\(--croffle-accent-hue\) \+ ([\d.]+)\)\)$/.exec(
    value ?? '',
  );
  return m ? [Number(m[1]), Number(m[2]), m[3]!] : null;
}
