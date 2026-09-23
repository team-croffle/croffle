import { randomUUID } from 'node:crypto';

import { and, eq, isNotNull, notInArray } from 'drizzle-orm';

import { databaseManager } from '../database';
import { scheduleReminders, scheduleTags } from '../database/schema';
import { alive, revive, tombstone, touch } from '../sync/lww';

/**
 * Make the live link set equal `tags`. Links that drop out are tombstoned (never deleted);
 * a link that comes back is revived through the (scheduleId, tagId) unique index.
 */
export async function syncScheduleTags(scheduleId: string, tags: { id: string }[] | undefined) {
  if (tags === undefined) {
    return;
  }

  const db = databaseManager.getDb();
  const now = new Date();
  const tagIds = [...new Set(tags.map((tag) => tag.id))];

  await db
    .update(scheduleTags)
    .set(tombstone(scheduleTags, now))
    .where(
      and(
        eq(scheduleTags.scheduleId, scheduleId),
        alive(scheduleTags),
        tagIds.length > 0 ? notInArray(scheduleTags.tagId, tagIds) : undefined,
      ),
    );

  if (tagIds.length === 0) {
    return;
  }

  await db
    .insert(scheduleTags)
    .values(
      tagIds.map((tagId) => ({
        id: randomUUID(),
        scheduleId,
        tagId,
        version: 1,
        ...touch(now),
      })),
    )
    .onConflictDoUpdate({
      target: [scheduleTags.scheduleId, scheduleTags.tagId],
      set: revive(scheduleTags, now),
      setWhere: isNotNull(scheduleTags.deletedAt),
    });
}

/** Same tombstone / revive pattern as tags, keyed by the deterministic reminder id. */
export async function syncScheduleReminders(scheduleId: string, reminders: number[] | undefined) {
  if (reminders === undefined) {
    return;
  }

  const db = databaseManager.getDb();
  const now = new Date();

  await db
    .update(scheduleReminders)
    .set(tombstone(scheduleReminders, now))
    .where(
      and(
        eq(scheduleReminders.scheduleId, scheduleId),
        alive(scheduleReminders),
        reminders.length > 0 ? notInArray(scheduleReminders.minutes, reminders) : undefined,
      ),
    );

  if (reminders.length === 0) {
    return;
  }

  await db
    .insert(scheduleReminders)
    .values(
      reminders.map((minutes) => ({
        id: `${scheduleId}-${minutes}`,
        scheduleId,
        minutes,
        version: 1,
        ...touch(now),
      })),
    )
    .onConflictDoUpdate({
      target: scheduleReminders.id,
      set: revive(scheduleReminders, now),
      setWhere: isNotNull(scheduleReminders.deletedAt),
    });
}

/** Tombstone every live link of a tag (used when the tag itself is removed). */
export async function tombstoneLinksOfTag(tagId: string, now: Date = new Date()): Promise<void> {
  const db = databaseManager.getDb();
  await db
    .update(scheduleTags)
    .set(tombstone(scheduleTags, now))
    .where(and(eq(scheduleTags.tagId, tagId), alive(scheduleTags)));
}
