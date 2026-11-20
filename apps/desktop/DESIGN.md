# Croffle desktop design system

Rules for the renderer UI (`src/renderer/src`). Follow them for every new or changed screen. When a
screen needs something these rules do not cover, extend this file in the same change instead of
adding a one-off class.

The values come from the shared design system **"croffle"** (themes **Croffle Light** and
**Croffle Dark**), which the desktop app, the Server WebUI, the mobile app and pipit-hub use:
<https://claude.ai/artifact/9RJ3iMqwtduoMasBnMe4ky> (read `project/README.md`, then
`project/tokens.json`). This file is how the desktop app applies it.

The short version: **use the parts in `components/ui` as they are, and style with tokens.** Almost
every inconsistency this file was written to remove came from a screen overriding a part's height,
radius, color or focus ring with its own classes.

## Tokens

Colors come from CSS variables in `src/renderer/src/index.css`. Screens use the Tailwind names of
those variables and never a palette color (`neutral-200`, `emerald-500`), a hex value
or an `rgba()`.

| Use                                           | Class                                                         |
| --------------------------------------------- | ------------------------------------------------------------- |
| Accent fill (the main action, active)         | `bg-primary` with `text-primary-foreground` on it             |
| Accent as lettering (title, today, link)      | `text-croffle-primary-strong`                                 |
| Page and window background                    | `bg-background` (app shell `bg-croffle-bg`)                   |
| Raised surface                                | `bg-card`; menus and popovers `bg-popover`                    |
| Sidebars                                      | `bg-sidebar` (`bg-croffle-sidebar`)                           |
| Quiet fill, hover                             | `bg-muted`; hover `bg-accent` (`bg-croffle-hover`)            |
| Text                                          | `text-foreground`, secondary `text-muted-foreground`          |
| Decorative lines                              | `border-border` (the default of `border`)                     |
| Control edges (inputs, selects, switch track) | `border-input` / `bg-input` — at least 3:1                    |
| Focus ring                                    | `ring-ring` (applied by the parts)                            |
| Danger fill                                   | `bg-destructive` with `text-destructive-foreground` on it     |
| Error text                                    | `text-destructive`                                            |
| Status                                        | `success`, `warning`, `info`, `destructive` (+ `-foreground`) |

- **Contrast.** Every text pair holds 4.5:1 in both themes and control edges 3:1;
  `test/tokens-contrast.test.ts` checks it from `index.css` for every accent hue. Labels on `primary` are
  `primary-foreground`, never white. Status colors used as lettering go on `background`, `card` or
  `sidebar` (not on `muted`, where `warning` falls under 4.5:1).
- **Error text** is `text-destructive`. `destructive-foreground` is only for text placed over a
  `bg-destructive` fill (shadcn convention).
- **Status** is a dot and a word, never color alone, and never the accent. Priority: 낮음
  `success`, 보통 `warning`, 높음 `destructive` (`border-… bg-…/14` with a `bg-…` dot).
- **Warnings** (insecure option, beta notice, clock skew) use `warning`: `text-warning`, or a
  notice box `border-warning bg-warning/10 text-foreground`.
- **Dark surfaces rise in lightness:** `background` < `card` = `popover`.
- Light and dark come from the tokens. A screen should not need a `dark:` color class.
- The only allowed palette color is the window close button's red hover in the title bar.

### `--croffle-*` names

Extensions read the `--croffle-*` variables, so **their names stay**. Their values are aliases of
the tokens above (`--croffle-bg` → `--background`, `--croffle-text` → `--muted-foreground`,
`--croffle-red` → `--destructive`, …); a value changes when the design system changes. Never rename
or remove one; add a new one only with an alias. New screen code uses the semantic names.

### Accent color setting

Settings › General › Accent color changes only `--croffle-accent-hue` (default 70; a picked color
gives only its hue). Three groups of tokens:

- **Accent family** — `primary`, `ring`, `accent-foreground` (= `croffle-primary-strong`) and
  `croffle-today-bg` follow the hue at their own lightness and chroma. Dark adds a small hue offset
  so the default gives the system's dark values. Light `primary` and `croffle-today-bg` keep the
  system's lightness 0.5469 except near hue 185, where it dips by up to 0.0095
  (`0.5469 - 0.0095 * max(0, cos(hue - 185°))`, the same formula as the settings preview's
  `accentLightness`) so `primary-foreground` keeps 4.5:1 for every hue.
- **Neutrals** follow the hue only: `oklch(L C calc(var(--croffle-accent-hue) + d))`, with the
  system's L and C and a fixed offset `d` per token. Light: `background`, `foreground`,
  `secondary(-foreground)`, `muted(-foreground)`, `accent`, `border`, `input`, `sidebar`, with
  `d` = the token's system hue − 70, so the default gives the system's hex. Dark: the same tokens
  plus `card`, `popover`, `sidebar-accent`, `sidebar-border`, at the system's dark L and C with
  the light partner's `d` (`card`/`popover` → `background`, `sidebar-accent`/`sidebar-border` →
  `muted`), so both themes lean toward the accent the same way. Their chroma is low (0.009–0.04),
  so the tint is slight and contrast barely moves.
- **Fixed** — `destructive`, `success`, `warning`, `info` and their `-foreground`,
  `primary-foreground`, the shadows, `croffle-day-select-bg`, and colors without chroma (light
  `card`, `popover`).

`test/tokens-contrast.test.ts` checks the groups, the system values at hue 70, and every text pair
and control edge for every hue 0–359. To reuse the rule elsewhere, change the base hue, each
token's L, C and `d`, and the fixed list.

## Controls

| Control                                                   | Height | How                                    |
| --------------------------------------------------------- | ------ | -------------------------------------- |
| Button, Input, Select trigger, time field                 | 36px   | the part's default size — add no `h-*` |
| Dense toggle (weekday chips)                              | 32px   | `size="sm"`                            |
| Icon button on its own in content                         | 32px   | `size="icon-sm"`                       |
| Icon button next to 36px controls, or in the sidebar rail | 36px   | `size="icon"`                          |
| Icon button in the title bar                              | 28px   | `size="icon-xs"`                       |

- Use `Button`, `Input`, `Textarea`, `Select`, `Switch`, `Badge` from `components/ui`. Do not write
  a bare `<button>` or `<input>`. Exceptions: color swatches, the options of the priority picker,
  the small remove button inside a `Badge`, the three window buttons, `<input type="color">`, the
  rows of a list inside a popover (notifications) and the version tile.
- Keep the focus ring of a part. A composite control (the time field, the tags input) shows the
  same ring on its wrapper with `focus-within:`.
- These rules are about how a screen uses a part. What a generated part draws inside itself (the
  day buttons of the date picker, the items of a select list or a menu) keeps the part's own sizes.
- Do not override a part's height, horizontal padding, font weight, radius, border color or focus
  ring from a screen. If a size is missing, add a variant to the part.
- A part in `components/ui` is changed in place only when tokens alone cannot meet these rules
  (1.3.2: the destructive variants of `button` and `badge`, the error text of the context menu
  and the calendars, the switch, the popover's radius and shadow).
- A trigger that shows a value and opens a picker (the date fields) is
  `Button variant="outline"` with `w-full justify-between font-normal`, so it reads as a field.
- A toggle is `Button variant="outline"` with `aria-pressed`; the part styles the pressed state.
- Sidebar rail buttons are `SidebarMenuButton size="rail"`.
- Clickable badge-like things (tag suggestions) look like `Badge`, not like a small button.
- Selectors that are not single-line controls (the priority picker, color swatches) are outside
  the height rule.
- A row of color swatches ends with a custom swatch (`<input type="color">`). While the value is
  not one of the presets, the custom swatch is filled with it and drawn as selected; otherwise it
  is a dashed circle with a palette icon.

### Buttons

| Role                                   | Variant                             |
| -------------------------------------- | ----------------------------------- |
| The main action of a screen or dialog  | default (accent) — no color classes |
| Secondary, cancel, close               | `outline`                           |
| Confirming something irreversible      | `destructive`                       |
| A delete action inside a row or footer | `ghost` with `text-destructive`     |
| Low-emphasis action                    | `ghost`                             |

- One accent fill per area; everything else there is `outline` or `ghost`.
- Width comes from the label and the size's padding. Never add `px-*` to make a button wider.
- Dialog footer buttons have `min-w-20` (80px), sit right-aligned with `gap-2`, and the main action
  is the rightmost one.
- A secondary action that leaves the flow (Delete in the schedule dialog, Back in the help dialog)
  may sit at the left end of the footer; the row still has `gap-2`.
- An icon inside a button is 16px and is spaced by the gap of the button's size. Do not add `mr-*`
  or size classes to it.

### Inputs

- In a stacked form (label above the field) the field is `w-full`.
- In a settings row (label and hint on the left, control on the right) the control is `w-60`
  (240px) and the row is `flex items-center justify-between gap-4`.
- Number inputs are `w-24`.
- A fixed-width control in a row also gets `shrink-0`, so a long label wraps instead of squeezing it.
- Labels are `Label` with `text-sm font-medium`; hints are `text-xs text-muted-foreground`.
- An invalid value shows its reason under the field in `text-xs text-destructive` with
  `role="alert"`, and the field gets `aria-invalid`.

## Radius

`--radius` is 12px. Use these steps.

| Step    | Value | Class          | For                                                      |
| ------- | ----- | -------------- | -------------------------------------------------------- |
| Small   | 8px   | `rounded-sm`   | small chips, week numbers                                |
| Control | 10px  | `rounded-md`   | buttons, inputs, selects, badges, list and nav items     |
| Large   | 12px  | `rounded-lg`   | calendar events                                          |
| Surface | 16px  | `rounded-xl`   | cards, dialogs, popovers, menus                          |
| 2xl     | 20px  | `rounded-2xl`  | today and selected calendar cells                        |
| 3xl     | 24px  | `rounded-3xl`  | the calendar card (the app's main surface)               |
| Round   | —     | `rounded-full` | avatars, swatches, status dots, the collapsed add button |

In `<style>` blocks use the variables (`var(--radius-md)`), not pixel values.

## Shadows

One set, tinted with the ink color in light and deep black in dark (`--elevation-*` in
`index.css`, used through the Tailwind classes):

| Class            | For                                                                              |
| ---------------- | -------------------------------------------------------------------------------- |
| `shadow-xs`      | inputs and outline buttons (the parts)                                           |
| `shadow-sm`      | cards on the page                                                                |
| `shadow-surface` | the one main surface of a screen (calendar card; `var(--shadow-surface)` in CSS) |
| `shadow-lg`      | dialogs, popovers, menus                                                         |

## Dialogs, popovers and cards

| Kind                               | Width                      |
| ---------------------------------- | -------------------------- |
| Confirmation, notice, help, update | `sm:max-w-md` (448px)      |
| Form (schedule)                    | `sm:max-w-lg` (512px)      |
| Popover (notifications, version)   | `w-[340px]`                |
| Settings                           | its own full-window layout |

- A dialog keeps the part's defaults: `rounded-xl`, `shadow-lg`, title `text-lg font-semibold`,
  description `text-sm text-muted-foreground`.
- A footer that is visually separated from a scrolling body is
  `border-t bg-muted/20 px-6 py-4`.
- A card is `rounded-xl border bg-card` (add `shadow-sm` when it sits on the page background), or
  the `Card` part. An inset option group inside a form is `rounded-xl border bg-muted/30`.
- A popover is `rounded-xl shadow-lg` (the part's default): a header with the title and at most one
  text action, then rows of a 16px icon, a sentence and a meta line. Empty: one calm sentence.

## Behaviour

- **Settings apply immediately.** There is no Save/Cancel footer. Switches and selects save when
  changed; a text or number field saves when it is left or Enter is pressed; an invalid value is
  not saved and says why under the field; a failed save shows an error toast and puts the stored
  value back.
- **Irreversible actions** (deleting an extension, changing the sync server, importing) ask with
  the app's confirm dialog (`uiStore.openConfirm`, `destructive` confirm button), never the
  browser's `confirm()`.
- **Notifications** open from the bell as a popover: reminders that went off during this run
  (newest first, unread in `font-semibold`, kept in memory only) and those due in the next 24
  hours. The bell shows the unread count; only "모두 읽음" marks them read. A row opens its
  schedule.
- **Updates** can be checked by hand from the version tile and Settings › General › Updates; both
  show the same state: checking, 최신 버전, 새 버전 (opens the update dialog), 실패, or "cannot
  check" for a development build or an install that cannot update. A check started by hand shows
  its result in place, without a toast. The version tile sits at the bottom of the right sidebar:
  expanded, a row (info icon, `CROFFLE v<version>`, an info dot when a new version is known);
  collapsed, a ghost icon button. Its bottom lines up with the left sidebar's rail buttons.
- **The theme button** saves on left-click (light ↔ dark); right-click offers 라이트 / 다크 / 시스템.
- **Nav subtitles** add information ("오늘 일정 3개"); they never repeat the title in another
  language.
- **Every panel that loads data** has a loading, an error and an empty state; an empty state is one
  calm sentence.
- **Help text** describes what the app actually does and changes in the same change as the
  behaviour.

## Type

Body text is Pretendard (400 · 500 · 600 · 700, bundled); the wordmark is Fredoka.

| Text                                   | Class                                  |
| -------------------------------------- | -------------------------------------- |
| Settings page title                    | `text-2xl font-bold`                   |
| Dialog title                           | `text-lg font-semibold` (part default) |
| Section heading                        | `text-base font-bold text-foreground`  |
| Heading inside a card or sidebar panel | `text-sm font-bold`                    |
| Label                                  | `text-sm font-medium`                  |
| Body                                   | `text-sm`                              |
| Hint, meta                             | `text-xs text-muted-foreground`        |
| Smallest (badges in lists)             | `text-2xs`                             |

Icons are sized with `size-*` (`size-4` = 16px by default, `size-5` for sidebar rail icons), not
`h-* w-*`.

## Copy

Korean is the primary language; every string is in both `ko.json` and `en.json` under the same key,
and `ko.json` has no English sentences (`test/i18n-catalogs.test.ts`). Help and empty states use
해요체; notices and errors use 합니다체. Say what happened, then what to do.

## Checking a change

`pnpm -r test` runs the token contrast and catalog checks. Before finishing UI work, also run
`apps/desktop/scripts/check-ui-tokens.sh`, which lists the palette colors, hex and `rgba()` values,
`text-destructive-foreground` uses and browser `confirm()` calls in the screens; only the title bar
close button's red hover and `text-destructive-foreground` over `bg-destructive` may remain. Then
look at the screen in both light and dark.
