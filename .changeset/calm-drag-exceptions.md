---
'@croffledev/croffle-types': minor
---

Add recurring-schedule exceptions: `ScheduleException`, optional `Schedule.exceptions` and `SchedulesApi.moveOccurrence(id, occurrenceStart, newStart)`. Extensions that expand `recurrenceRule` themselves should skip each exception's `occurrenceStart` and use its `startDate`/`endDate` unless `cancelled`.
