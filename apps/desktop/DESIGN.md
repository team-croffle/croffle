# Croffle desktop design system

Rules for the renderer UI (`src/renderer/src`). Follow them for every new or changed screen. When a
screen needs something these rules do not cover, extend this file in the same change instead of
adding a one-off class.

The short version: **use the parts in `components/ui` as they are, and style with tokens.** Almost
every inconsistency this file was written to remove came from a screen overriding a part's height,
radius, color or focus ring with its own classes.

## Tokens

Colors come from CSS variables in `src/renderer/src/index.css`. Screens use the Tailwind names of
those variables and never a palette color (`neutral-200`, `red-500`) or a hex value.

| Use                        | Class                                                     |
| -------------------------- | --------------------------------------------------------- |
| Accent (primary actions)   | `bg-primary` / `text-primary` / `text-primary-foreground` |
| Page and dialog background | `bg-background`, app shell `bg-croffle-bg`                |
| Raised surface             | `bg-card`                                                 |
| Subtle fill, hover         | `bg-muted`, `bg-accent`                                   |
| Text                       | `text-foreground`, secondary `text-muted-foreground`      |
| Borders                    | `border-border` (the default of `border`), `border-input` |
| Focus ring                 | `ring-ring` (applied by the parts)                        |
| Danger                     | `bg-destructive`, `text-destructive`                      |

- `--primary`, `--ring`, `--border` and `--input` are wired to the Croffle tokens
  (`--croffle-primary`, `--croffle-border`), which follow the accent color the user picks in
  Settings. Anything painted with `primary` changes with that setting; a hex value does not.
- `--croffle-*` variables are visible to extensions. Do not rename them or change their values.
- Light and dark come from the tokens. A screen should not need a `dark:` color class.
- Error text is `text-destructive-foreground` (readable in dark); `text-destructive` is for
  destructive actions.
- Allowed exceptions: the window close button's red hover in the title bar, and warning notices,
  which use Tailwind's `amber-*` with a `dark:` pair until there is a warning token.

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
  the small remove button inside a `Badge`, the three window buttons, and `<input type="color">`.
- Keep the focus ring of a part. A composite control (the time field, the tags input) shows the
  same ring on its wrapper with `focus-within:`.
- Do not override a part's height, horizontal padding, font weight, radius, border color or focus
  ring from a screen. If a size is missing, add a variant to the part.
- A trigger that shows a value and opens a picker (the date fields) is
  `Button variant="outline"` with `w-full justify-between font-normal`, so it reads as a field.
- A toggle is `Button variant="outline"` with `aria-pressed`; the part styles the pressed state.
- Sidebar rail buttons are `SidebarMenuButton size="rail"`.
- Clickable badge-like things (tag suggestions) look like `Badge`, not like a small button.
- Selectors that are not single-line controls (the priority picker, color swatches) are outside
  the height rule.

### Buttons

| Role                                   | Variant                             |
| -------------------------------------- | ----------------------------------- |
| The main action of a screen or dialog  | default (accent) — no color classes |
| Secondary, cancel, close               | `outline`                           |
| Confirming something irreversible      | `destructive`                       |
| A delete action inside a row or footer | `ghost` with `text-destructive`     |
| Low-emphasis action                    | `ghost`                             |

- Width comes from the label and the size's padding. Never add `px-*` to make a button wider.
- Dialog footer buttons have `min-w-20` (80px), sit right-aligned with `gap-2`, and the main action
  is the rightmost one.
- An icon inside a button is 16px and is spaced by the gap of the button's size. Do not add `mr-*`
  or size classes to it.

### Inputs

- In a stacked form (label above the field) the field is `w-full`.
- In a settings row (label and hint on the left, control on the right) the control is `w-60`
  (240px) and the row is `flex items-center justify-between gap-4`.
- Number inputs are `w-24`.
- A fixed-width control in a row also gets `shrink-0`, so a long label wraps instead of squeezing it.
- Labels are `Label` with `text-sm font-medium`; hints are `text-xs text-muted-foreground`.

## Radius

`--radius` is 10px. Use three steps only.

| Step    | Value | Class          | For                                                      |
| ------- | ----- | -------------- | -------------------------------------------------------- |
| Control | 8px   | `rounded-md`   | buttons, inputs, selects, badges, list and nav items     |
| Surface | 14px  | `rounded-xl`   | cards, dialogs, panels                                   |
| Round   | —     | `rounded-full` | avatars, swatches, status dots, the collapsed add button |

Menus and popovers (select lists, context menus, the date picker) keep the part's 8px.

Exceptions: the calendar card is the app's main surface and uses 24px (`--radius-3xl`); calendar
events use `--radius-lg`. In `<style>` blocks use the variables (`var(--radius-md)`), not pixel
values.

## Dialogs and cards

| Kind                               | Width                      |
| ---------------------------------- | -------------------------- |
| Confirmation, notice, help, update | `sm:max-w-md`              |
| Form (schedule)                    | `sm:max-w-lg`              |
| Settings                           | its own full-window layout |

- A dialog keeps the part's defaults: `rounded-xl`, `shadow-lg`, title `text-lg font-semibold`,
  description `text-sm text-muted-foreground`.
- A footer that is visually separated from a scrolling body is
  `border-t bg-muted/20 px-6 py-4`.
- A card is `rounded-xl border bg-card` (add `shadow-sm` when it sits on the page background), or
  the `Card` part. An inset option group inside a form is `rounded-xl border bg-muted/30`.

## Type

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

## Checking a change

There is no lint rule for these. Before finishing UI work, search the screens you touched for the
things this file forbids:

```sh
cd apps/desktop/src/renderer/src
grep -rnE '(bg|text|border|ring)-(neutral|gray|zinc|slate)-[0-9]+|\[#[0-9A-Fa-f]{3,8}\]' --include='*.vue' --exclude-dir=ui .
grep -rnE '<(button|input)\b' --include='*.vue' --exclude-dir=ui .
```

and look at the screen in both light and dark.
