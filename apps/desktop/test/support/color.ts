/** Colour maths for the token tests (oklch → sRGB, WCAG luminance and contrast). */
export type Rgb = [number, number, number];

export const srgbToLinear = (c: number) =>
  c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
export const linearToSrgb = (c: number) => {
  const x = Math.min(1, Math.max(0, c));
  return x <= 0.0031308 ? x * 12.92 : 1.055 * x ** (1 / 2.4) - 0.055;
};

export function oklchToSrgb(l: number, c: number, h: number): Rgb {
  const a = c * Math.cos((h * Math.PI) / 180);
  const b = c * Math.sin((h * Math.PI) / 180);
  const l3 = (l + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m3 = (l - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s3 = (l - 0.0894841775 * a - 1.291485548 * b) ** 3;
  // Out-of-gamut channels are clipped, as the browser shows them.
  return [
    linearToSrgb(4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3),
    linearToSrgb(-1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3),
    linearToSrgb(-0.0041960863 * l3 - 0.7034186147 * m3 + 1.707614701 * s3),
  ];
}

/** oklch [L, C, H°] of an sRGB colour. */
export function srgbToOklch([r, g, b]: Rgb): Rgb {
  const [lr, lg, lb] = [r, g, b].map(srgbToLinear) as Rgb;
  const l = Math.cbrt(0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb);
  const m = Math.cbrt(0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb);
  const s = Math.cbrt(0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb);
  const a = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
  const bb = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
  const h = (Math.atan2(bb, a) * 180) / Math.PI;
  return [
    0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    Math.hypot(a, bb),
    (h + 360) % 360,
  ];
}

export const srgbToOklchL = (rgb: Rgb): number => srgbToOklch(rgb)[0];

export const hexToRgb = (hex: string): Rgb =>
  [1, 3, 5].map((i) => Number.parseInt(hex.slice(i, i + 2), 16) / 255) as Rgb;

/** sRGB channels as 8-bit integers (what a hex value can hold). */
export const to8bit = (rgb: Rgb) => rgb.map((c) => Math.round(Math.min(1, Math.max(0, c)) * 255));

export const luminance = (rgb: Rgb) => {
  const [r, g, b] = rgb.map(srgbToLinear) as Rgb;
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};

export const contrast = (a: Rgb, b: Rgb) => {
  const [hi, lo] = [luminance(a), luminance(b)].toSorted((x, y) => y - x) as [number, number];
  return (hi + 0.05) / (lo + 0.05);
};

/** The accent dip: `calc(L - k * max(0, cos((var(--croffle-accent-hue) - c) * 1deg)))`. */
export const DIP =
  /^calc\(([\d.]+) - ([\d.]+) \* max\(0, cos\(\(var\(--croffle-accent-hue\) - ([\d.]+)\) \* 1deg\)\)\)$/;

export function lightnessOf(expr: string, hue: number): number {
  const m = DIP.exec(expr);
  if (!m) {
    return Number(expr);
  }
  const [base, dip, center] = [Number(m[1]), Number(m[2]), Number(m[3])];
  return base - dip * Math.max(0, Math.cos(((hue - center) * Math.PI) / 180));
}
