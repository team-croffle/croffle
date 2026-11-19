import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import type { NotificationItem } from '../src/common/api/notifications.ts';
import { FIRED_LIMIT, NotificationLog } from '../src/main/notification/notification-list.ts';

const MINUTE = 60_000;
const NOW = Date.UTC(2026, 10, 19, 1, 0, 0);

function item(n: number, remindAt = NOW): NotificationItem {
  return {
    id: `s${n}|${remindAt}|10`,
    scheduleId: `s${n}`,
    title: `일정 ${n}`,
    occurrenceStart: remindAt + 10 * MINUTE,
    remindAt,
    reminderMinutes: 10,
    isAllDay: false,
  };
}

const none = () => false;

describe('notification list', () => {
  test('fired items are newest first and capped at 50', () => {
    const log = new NotificationLog();
    for (let n = 1; n <= FIRED_LIMIT + 5; n++) {
      log.add(item(n), NOW + n * MINUTE);
    }
    const first = log.list([], NOW, none);
    assert.equal(FIRED_LIMIT, 50);
    assert.equal(first.fired.length, 50);
    assert.equal(first.fired[0]!.scheduleId, `s${FIRED_LIMIT + 5}`);
    assert.equal(first.fired.at(-1)!.scheduleId, 's6');
    assert.equal(first.unread, 50);
    // Reading the list (opening the popover) does not mark anything read.
    assert.equal(log.list([], NOW, none).unread, 50);
    assert.ok(log.list([], NOW, none).fired.every((f) => !f.read));
  });

  test('mark all read clears the unread count', () => {
    const log = new NotificationLog();
    log.add(item(1), NOW);
    log.add(item(2), NOW + MINUTE);
    assert.equal(log.markAllRead(), true);
    const list = log.list([], NOW, none);
    assert.equal(list.unread, 0);
    assert.ok(list.fired.every((f) => f.read));
    assert.equal(log.markAllRead(), false);
    log.add(item(3), NOW + 2 * MINUTE);
    assert.equal(log.list([], NOW, none).unread, 1);
  });

  test('upcoming keeps candidates within 24 hours sorted by time', () => {
    const log = new NotificationLog();
    const later = item(1, NOW + 3 * 60 * MINUTE);
    const soon = item(2, NOW + 5 * MINUTE);
    const tooFar = item(3, NOW + 24 * 60 * MINUTE + MINUTE);
    const past = item(4, NOW - MINUTE);
    const alreadyFired = item(5, NOW + 10 * MINUTE);
    const list = log.list(
      [later, soon, tooFar, past, alreadyFired],
      NOW,
      (c) => c.id === alreadyFired.id,
    );
    assert.deepEqual(
      list.upcoming.map((u) => u.scheduleId),
      ['s2', 's1'],
    );
  });
});
