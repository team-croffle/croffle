---
'@croffledev/croffle-types': major
---

Remove the deprecated `Schedule.reminderMinutes` mirror. Read and write `Schedule.reminders` (`number[] | null`) instead: `null` follows the app-wide default, `[]` means no reminder, and `[m]` replaces the old single value `m`.
