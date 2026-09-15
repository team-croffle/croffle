---
'@croffledev/croffle-types': minor
---

Add `reminders` to `Schedule` and deprecate `reminderMinutes`

A schedule can now carry several reminder offsets instead of a single one.

- `reminders: number[] | null` — offsets in minutes before the start, ascending. `null` follows the app-wide default; `[]` means no reminder.
- `reminderMinutes` still reports the first offset so extensions written against 1.2.0 keep working. It is deprecated and will be removed in 1.3; read `reminders` instead.
