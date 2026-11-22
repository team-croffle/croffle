---
'@croffledev/croffle-types': minor
---

`Schedule` gains two optional fields for To-Dos (desktop 1.3.3): `kind` (`'event' | 'todo'`, missing means an event) and `completedAt` (when a To-Do was done; `null` or missing means not done). A To-Do's `endDate` is its deadline and its `startDate` equals it; a To-Do does not repeat.
