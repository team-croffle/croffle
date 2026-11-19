import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import {
  nextUpdateStatus,
  toastsLatest,
  UPDATE_IDLE,
  type UpdateCheckEvent,
  type UpdateCheckStatus,
} from '../src/common/update-status.ts';

const run = (events: UpdateCheckEvent[], from: UpdateCheckStatus = UPDATE_IDLE) => {
  const seen: UpdateCheckStatus[] = [];
  let status = from;
  for (const event of events) {
    status = nextUpdateStatus(status, event);
    seen.push(status);
  }
  return seen;
};

describe('update check status', () => {
  test('manual check goes checking then latest', () => {
    const seen = run([{ type: 'check', result: { status: 'started' } }, { type: 'not-available' }]);
    assert.deepEqual(seen, [{ kind: 'checking' }, { kind: 'latest' }]);
    assert.equal(toastsLatest(true), false);
    assert.equal(toastsLatest(false), true);
  });

  test('available opens the update dialog state', () => {
    const seen = run([
      { type: 'check', result: { status: 'started' } },
      { type: 'available', version: '1.3.3' },
    ]);
    assert.deepEqual(seen.at(-1), { kind: 'available', version: '1.3.3' });
  });

  test('error keeps the message', () => {
    const seen = run([
      { type: 'check', result: { status: 'started' } },
      { type: 'error', message: 'net::ERR_INTERNET_DISCONNECTED' },
    ]);
    assert.deepEqual(seen.at(-1), { kind: 'error', message: 'net::ERR_INTERNET_DISCONNECTED' });
  });

  test('unsupported install never stays checking', () => {
    for (const reason of ['development', 'not-updatable'] as const) {
      const seen = run([{ type: 'check', result: { status: 'unsupported', reason } }]);
      assert.deepEqual(seen, [{ kind: 'unsupported', reason }]);
      assert.ok(seen.every((s) => s.kind !== 'checking'));
    }
  });
});
