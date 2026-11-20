/**
 * Values of the design system "croffle" that the accent-hue rule is checked against
 * (DESIGN.md › Accent hue). Light: the system's hex at the default hue 70. Dark: the system's
 * slate values, whose L and C are kept while the hue follows the accent like the light partner.
 */

/** Neutral tokens of Croffle Light that follow the accent hue, with the system hex at hue 70. */
export const SYSTEM_LIGHT_NEUTRALS: Record<string, string> = {
  '--background': '#faf7f1',
  '--foreground': '#221d13',
  '--secondary': '#f3eee2',
  '--secondary-foreground': '#3a3327',
  '--muted': '#f3eee2',
  '--muted-foreground': '#5c5546',
  '--accent': '#ece5d5',
  '--border': '#e4dcc9',
  '--input': '#958971', // one step darker than the system's #968a72 to keep 3:1 for every hue (A-25)
  '--sidebar': '#f6f2e9',
};

/** Neutral tokens of Croffle Dark that follow the accent hue, with the slate value they keep L and C of. */
export const SYSTEM_DARK_NEUTRALS: Record<string, string> = {
  '--background': '#15171b',
  '--foreground': '#ece7dd',
  '--card': '#1c1f25',
  '--popover': '#1c1f25',
  '--secondary': '#23262d',
  '--secondary-foreground': '#ece7dd',
  '--muted': '#23262d',
  '--muted-foreground': '#a9a294',
  '--accent': '#292d35',
  '--border': '#2b2f37',
  '--input': '#6a707c',
  '--sidebar': '#181b20',
  '--sidebar-accent': '#23262d',
  '--sidebar-border': '#23262d',
};

/** Dark tokens without a light partner of their own take the offset of this light token. */
export const DARK_PARTNER: Record<string, string> = {
  '--card': '--background',
  '--popover': '--background',
  '--sidebar-accent': '--muted',
  '--sidebar-border': '--muted',
};

/** Tokens that never follow the accent hue, with their values. */
export const FIXED_LIGHT: Record<string, string> = {
  '--card': '#ffffff',
  '--popover': '#ffffff',
  '--primary-foreground': '#fffaf0',
  '--destructive': '#bf3d40',
  '--destructive-foreground': '#fffaf0',
  '--success': '#2a7c52',
  '--success-foreground': '#fffaf0',
  '--warning': '#af5519', // the system's #b05519 one step deeper for every hue (A-24)
  '--warning-foreground': '#fffaf0',
  '--info': '#3c6ea5',
  '--info-foreground': '#fffaf0',
  '--croffle-day-select-bg': 'rgba(60, 110, 165, 0.14)',
};

export const FIXED_DARK: Record<string, string> = {
  '--primary-foreground': '#241703',
  '--destructive': '#e3676a',
  '--destructive-foreground': '#1a0d0d',
  '--success': '#57b17e',
  '--success-foreground': '#0f1f16',
  '--warning': '#e2874a',
  '--warning-foreground': '#1f1105',
  '--info': '#6fa8dc',
  '--info-foreground': '#0d1620',
  '--croffle-day-select-bg': 'rgba(111, 168, 220, 0.18)',
};
