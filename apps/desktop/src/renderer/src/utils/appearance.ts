/** Default Croffle accent hue (warm brown), matches index.css --croffle-accent-hue. */
export const DEFAULT_ACCENT_HUE = 69.8;

export type AppearanceOverrides = {
  accentHue?: number;
};

const ACCENT_HUE_VAR = '--croffle-accent-hue';

/** Clamp hue to the CSS hue range. */
export const normalizeAccentHue = (hue: number): number => {
  if (!Number.isFinite(hue)) {
    return DEFAULT_ACCENT_HUE;
  }
  const wrapped = ((hue % 360) + 360) % 360;
  return wrapped;
};

/**
 * Apply appearance overrides to the document.
 * Only sets provided keys so future tokens (e.g. bgHue) can extend the same API.
 */
export const applyAppearance = (overrides: AppearanceOverrides): void => {
  if (typeof document === 'undefined') {
    return;
  }
  if (overrides.accentHue !== undefined) {
    const hue = normalizeAccentHue(overrides.accentHue);
    document.documentElement.style.setProperty(ACCENT_HUE_VAR, String(hue));
  }
};

/** Convert #RRGGBB (or #RGB) to OKLCH hue degrees (matches CSS oklch hue). */
export const hexToHue = (hex: string): number => {
  const cleaned = hex.replace('#', '').trim();
  let r = 0;
  let g = 0;
  let b = 0;
  if (cleaned.length === 3) {
    r = Number.parseInt(cleaned[0]! + cleaned[0]!, 16);
    g = Number.parseInt(cleaned[1]! + cleaned[1]!, 16);
    b = Number.parseInt(cleaned[2]! + cleaned[2]!, 16);
  } else if (cleaned.length === 6) {
    r = Number.parseInt(cleaned.slice(0, 2), 16);
    g = Number.parseInt(cleaned.slice(2, 4), 16);
    b = Number.parseInt(cleaned.slice(4, 6), 16);
  } else {
    return DEFAULT_ACCENT_HUE;
  }

  return srgbToOklchHue(r / 255, g / 255, b / 255);
};

/**
 * Build a preview hex from hue using primary-like L/C in sRGB approximation.
 * Good enough for color-input binding and preset comparison.
 */
export const hueToPreviewHex = (hue: number): string => {
  const h = normalizeAccentHue(hue);
  // Match approximate lightness/chroma of --croffle-primary in light mode
  const l = 0.785;
  const c = 0.104;
  const [r, g, b] = oklchToSrgb(l, c, h);
  return rgbToHex(r, g, b);
};

const srgbToLinear = (c: number): number => {
  if (c <= 0.04045) {
    return c / 12.92;
  }
  return ((c + 0.055) / 1.055) ** 2.4;
};

const srgbToOklchHue = (r: number, g: number, b: number): number => {
  const lr = srgbToLinear(r);
  const lg = srgbToLinear(g);
  const lb = srgbToLinear(b);

  const l_ = Math.cbrt(0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb);
  const m_ = Math.cbrt(0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb);
  const s_ = Math.cbrt(0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb);

  const a = 1.9779984951 * l_ - 2.428592205 * m_ + 0.4505937099 * s_;
  const bOk = 0.0259040371 * l_ + 0.7827717662 * m_ - 0.808675766 * s_;

  if (Math.abs(a) < 1e-8 && Math.abs(bOk) < 1e-8) {
    return 0;
  }

  let hue = (Math.atan2(bOk, a) * 180) / Math.PI;
  if (hue < 0) {
    hue += 360;
  }
  return normalizeAccentHue(hue);
};

const oklchToSrgb = (l: number, c: number, h: number): [number, number, number] => {
  const hr = (h * Math.PI) / 180;
  const a = c * Math.cos(hr);
  const b = c * Math.sin(hr);

  const l_ = l + 0.3963377774 * a + 0.2158037573 * b;
  const m_ = l - 0.1055613458 * a - 0.0638541728 * b;
  const s_ = l - 0.0894841775 * a - 1.291485548 * b;

  const l3 = l_ ** 3;
  const m3 = m_ ** 3;
  const s3 = s_ ** 3;

  let r = +4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3;
  let g = -1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3;
  let bl = -0.0041960863 * l3 - 0.7034186147 * m3 + 1.707614701 * s3;

  r = linearToSrgb(r);
  g = linearToSrgb(g);
  bl = linearToSrgb(bl);

  return [r, g, bl];
};

const linearToSrgb = (c: number): number => {
  const clipped = Math.min(1, Math.max(0, c));
  if (clipped <= 0.0031308) {
    return clipped * 12.92;
  }
  return 1.055 * clipped ** (1 / 2.4) - 0.055;
};

const rgbToHex = (r: number, g: number, b: number): string => {
  const toByte = (v: number) =>
    Math.round(v * 255)
      .toString(16)
      .padStart(2, '0');
  return `#${toByte(r)}${toByte(g)}${toByte(b)}`.toUpperCase();
};

/** Preview hex for the default Croffle primary (oklch L/C + DEFAULT_ACCENT_HUE). */
export const DEFAULT_ACCENT_PRESET_HEX = hueToPreviewHex(DEFAULT_ACCENT_HUE);

/** True when two hues are close enough to treat as the same preset selection. */
export const huesMatch = (a: number, b: number, epsilon = 2): boolean => {
  const diff = Math.abs(normalizeAccentHue(a) - normalizeAccentHue(b));
  return Math.min(diff, 360 - diff) <= epsilon;
};
