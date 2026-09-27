import { randomUUID } from 'node:crypto';

import { and, asc, desc, eq } from 'drizzle-orm';

import { databaseManager } from '../database';
import { scheduleExceptions, type ScheduleWithTags } from '../database/schema';
import type { ScheduleExceptionInput } from '../mapper/schedule-mapper';
import { alive, bumpVersion, revive, tombstone, touch } from '../sync/lww';
import { findScheduleWithTags } from './schedule-query';

const toDate = (value: Date | string): Date => (value instanceof Date ? value : new Date(value));

/** Same wall-clock time, `days` calendar days later (DST-safe for local dates). */
export function addLocalDays(date: Date, days: number): Date {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

/** Create or update (and revive) the exception for one occurrence. */
async function upsertException(
  scheduleId: string,
  input: ScheduleExceptionInput,
  now: Date,
): Promise<void> {
  const db = databaseManager.getDb();
  const fields = {
    startDate: input.cancelled ? null : input.startDate,
    endDate: input.cancelled ? null : input.endDate,
    cancelled: input.cancelled,
  };
  await db
    .insert(scheduleExceptions)
    .values({
      id: randomUUID(),
      scheduleId,
      occurrenceStart: input.occurrenceStart,
      ...fields,
      version: 1,
      ...touch(now),
    })
    .onConflictDoUpdate({
      target: [scheduleExceptions.scheduleId, scheduleExceptions.occurrenceStart],
      set: { ...fields, ...revive(scheduleExceptions, now) },
    });
}

/** Tombstone every live exception of a schedule. */
export async function tombstoneExceptionsOf(
  scheduleId: string,
  now: Date = new Date(),
): Promise<void> {
  const db = databaseManager.getDb();
  await db
    .update(scheduleExceptions)
    .set(tombstone(scheduleExceptions, now))
    .where(and(eq(scheduleExceptions.scheduleId, scheduleId), alive(scheduleExceptions)));
}

/**
 * Make the live exception set equal `list` (import / full-object update path).
 * Entries missing from `list` are tombstoned; the rest are upserted.
 */
export async function replaceExceptions(
  scheduleId: string,
  list: ScheduleExceptionInput[],
): Promise<void> {
  const now = new Date();
  await tombstoneExceptionsOf(scheduleId, now);
  for (const entry of list) {
    await upsertException(scheduleId, entry, now);
  }
}

/**
 * The series start moved: keep every exception attached to the same occurrence by moving
 * its key (and its override) with the same calendar-day + time-of-day shift.
 * Rows are updated farthest-first so the (scheduleId, occurrenceStart) index never collides;
 * tombstones move too because they still hold keys.
 */
export async function shiftExceptionKeys(
  scheduleId: string,
  oldStart: Date,
  newStart: Date,
): Promise<void> {
  const db = databaseManager.getDb();
  const oldDay = new Date(oldStart.getFullYear(), oldStart.getMonth(), oldStart.getDate());
  const newDay = new Date(newStart.getFullYear(), newStart.getMonth(), newStart.getDate());
  const days = Math.round((newDay.getTime() - oldDay.getTime()) / 86_400_000);
  const timeMs = newStart.getTime() - newDay.getTime() - (oldStart.getTime() - oldDay.getTime());
  if (days === 0 && timeMs === 0) {
    return;
  }

  const shift = (d: Date | null) => (d ? new Date(addLocalDays(d, days).getTime() + timeMs) : d);
  const forward = days > 0 || (days === 0 && timeMs > 0);
  const rows = await db
    .select()
    .from(scheduleExceptions)
    .where(eq(scheduleExceptions.scheduleId, scheduleId))
    .orderBy(
      forward ? desc(scheduleExceptions.occurrenceStart) : asc(scheduleExceptions.occurrenceStart),
    );

  const now = new Date();
  for (const row of rows) {
    await db
      .update(scheduleExceptions)
      .set({
        occurrenceStart: shift(row.occurrenceStart)!,
        startDate: shift(row.startDate),
        endDate: shift(row.endDate),
        version: bumpVersion(scheduleExceptions),
        ...touch(now),
      })
      .where(eq(scheduleExceptions.id, row.id));
  }
}

/**
 * Move one occurrence of a recurring schedule to `newStart`, keeping the series duration.
 * Moving it back onto its original start removes the override.
 */
export async function moveOccurrence(
  id: string,
  occurrenceStartInput: Date | string,
  newStartInput: Date | string,
): Promise<ScheduleWithTags> {
  const schedule = await findScheduleWithTags(id);
  if (!schedule) {
    throw new Error('Schedule not found');
  }
  if (!schedule.recurrenceRule?.trim()) {
    throw new Error('Only recurring schedules have occurrences');
  }

  const occurrenceStart = toDate(occurrenceStartInput);
  const newStart = toDate(newStartInput);
  if (Number.isNaN(occurrenceStart.getTime()) || Number.isNaN(newStart.getTime())) {
    throw new Error('Invalid occurrence date');
  }

  const now = new Date();
  if (newStart.getTime() === occurrenceStart.getTime()) {
    const db = databaseManager.getDb();
    await db
      .update(scheduleExceptions)
      .set(tombstone(scheduleExceptions, now))
      .where(
        and(
          eq(scheduleExceptions.scheduleId, id),
          eq(scheduleExceptions.occurrenceStart, occurrenceStart),
          alive(scheduleExceptions),
        ),
      );
  } else {
    const duration = schedule.endDate.getTime() - schedule.startDate.getTime();
    await upsertException(
      id,
      {
        occurrenceStart,
        startDate: newStart,
        endDate: new Date(newStart.getTime() + duration),
        cancelled: false,
      },
      now,
    );
  }

  const updated = await findScheduleWithTags(id);
  if (!updated) {
    throw new Error('Schedule not found');
  }
  return updated;
}
