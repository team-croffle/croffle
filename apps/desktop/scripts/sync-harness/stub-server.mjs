#!/usr/bin/env node
// Fault-injecting stand-in for croffle-server 0.1, for checking the desktop sync client by hand
// (spec 001, W-12). It speaks the same wire protocol but keeps everything in memory, and can be
// told to misbehave. Not a server to run for real; the real target is the 0.1.0-rc.1 image.
//
// Usage: node scripts/sync-harness/stub-server.mjs   (see README.md for the switches)

import { randomBytes } from 'node:crypto';
import { readFileSync } from 'node:fs';
import http from 'node:http';
import https from 'node:https';

const env = process.env;
const PORT = Number(env.PORT ?? 8787);
const PAGE = Number(env.PAGE ?? 2);
const PROTOCOL_MIN = Number(env.PROTOCOL_MIN ?? 1);
const PROTOCOL_MAX = Number(env.PROTOCOL_MAX ?? 1);
const CLOCK_SKEW_MS = Number(env.CLOCK_SKEW_MS ?? 0);
const FORCE_OTP = env.FORCE_OTP === '1';
const OTP_CODE = env.OTP_CODE ?? '123456';
const CHILD_FIRST = env.CHILD_FIRST === '1';
const INVALID_ROW_ID = env.INVALID_ROW_ID ?? '';
let fail503 = Number(env.FAIL_503_TIMES ?? 0);

const USERNAME = env.STUB_USER ?? 'alice';
const PASSWORD = env.STUB_PASSWORD ?? 'secret';
const ACCOUNT_ID = 'acct-1';
const TEAM_ID = 'team-default';
const TABLES = ['tag', 'schedule', 'schedule_tags', 'schedule_reminder', 'schedule_exception'];
const CHILDREN = new Set(['schedule_tags', 'schedule_reminder', 'schedule_exception']);

const tokens = new Set(env.STUB_TOKEN ? [env.STUB_TOKEN] : []);
/** commit log: { serverSeq, table, rowId, op, row, version, updatedAtMs, lastWriterId } */
const log = [];
/** current rows by `${table}/${rowId}` */
const rows = new Map();
const appliedOps = new Set();
let purgedThroughSeq = 0;

const key = (table, rowId) => `${table}/${rowId}`;
const maxSeq = () => (log.length ? log.at(-1).serverSeq : 0);

function send(res, status, body) {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
}
const error = (res, status, code, message = code) => send(res, status, { code, message });

function readBody(req) {
  return new Promise((resolve) => {
    let data = '';
    req.on('data', (chunk) => (data += chunk));
    req.on('end', () => {
      try {
        resolve(data ? JSON.parse(data) : {});
      } catch {
        resolve(undefined);
      }
    });
  });
}

function authorised(req) {
  const header = req.headers.authorization ?? '';
  return header.startsWith('Bearer ') && tokens.has(header.slice(7));
}

/** The row as the server hands it back: payload plus the promoted fields. */
function wireRow(stored) {
  return {
    ...stored.row,
    version: stored.version,
    updatedAtMs: stored.updatedAtMs,
    lastWriterId: stored.lastWriterId ?? null,
    ownerId: ACCOUNT_ID,
    teamId: null,
    deletedAt: stored.row.deletedAt ?? null,
  };
}

function append(commit) {
  const serverSeq = maxSeq() + 1;
  const stored = {
    table: commit.table,
    rowId: commit.rowId,
    op: commit.op,
    row: commit.row ?? {},
    version: commit.version,
    updatedAtMs: commit.updatedAtMs,
    lastWriterId: commit.lastWriterId,
    serverSeq,
  };
  rows.set(key(commit.table, commit.rowId), stored);
  log.push({ ...stored, row: wireRow(stored) });
  return serverSeq;
}

function invalid(commit) {
  return (
    !commit.opId ||
    !commit.rowId ||
    !TABLES.includes(commit.table) ||
    !['upsert', 'delete'].includes(commit.op) ||
    !(commit.version > commit.baseVersion) ||
    !(commit.updatedAtMs > 0) ||
    (commit.op === 'upsert' && !commit.row) ||
    (INVALID_ROW_ID && commit.rowId === INVALID_ROW_ID)
  );
}

function childFirst(list) {
  if (!CHILD_FIRST) {
    return list;
  }
  return [...list].sort((a, b) => Number(CHILDREN.has(b.table)) - Number(CHILDREN.has(a.table)));
}

const handlers = {
  'GET /health': (_req, res) =>
    send(res, 200, {
      server: 'stub',
      protocol: { min: PROTOCOL_MIN, max: PROTOCOL_MAX },
      db: 'ok',
      retention: { oldestCommitSeq: purgedThroughSeq + 1, purgedThroughSeq },
      time: Date.now() + CLOCK_SKEW_MS,
    }),

  'POST /auth/login': (_req, res, body) => {
    if (body.username !== USERNAME || body.password !== PASSWORD) {
      return error(res, 401, 'unauthorized', 'wrong username or password');
    }
    if (FORCE_OTP && body.otp !== OTP_CODE) {
      return error(res, 401, 'otp_required', 'a one-time code is required');
    }
    const token = randomBytes(24).toString('hex');
    tokens.add(token);
    return send(res, 200, { token, accountId: ACCOUNT_ID, teamId: TEAM_ID });
  },

  'POST /auth/whoami': (req, res) =>
    authorised(req)
      ? send(res, 200, { accountId: ACCOUNT_ID, teamId: TEAM_ID })
      : error(res, 401, 'unauthorized'),

  'POST /sync/push': (_req, res, body) => {
    const commits = body.commits ?? [];
    if (commits.some(invalid)) {
      return error(res, 400, 'invalid_commit');
    }
    const result = { accepted: [], conflicts: [], serverSeq: 0 };
    for (const commit of commits) {
      if (appliedOps.has(commit.opId)) {
        result.accepted.push(commit.opId);
        continue;
      }
      const current = rows.get(key(commit.table, commit.rowId));
      if (current && current.version !== commit.baseVersion) {
        result.conflicts.push({
          opId: commit.opId,
          table: commit.table,
          rowId: commit.rowId,
          serverRow: wireRow(current),
          serverVersion: current.version,
          serverUpdatedAtMs: current.updatedAtMs,
          serverLastWriterId: current.lastWriterId,
        });
        continue;
      }
      result.serverSeq = append(commit);
      appliedOps.add(commit.opId);
      result.accepted.push(commit.opId);
    }
    result.serverSeq ||= maxSeq();
    return send(res, 200, result);
  },

  'POST /sync/pull': (_req, res, body) => {
    const last = body.lastServerSeq ?? 0;
    if (last < purgedThroughSeq) {
      return error(res, 410, 'cursor_too_old');
    }
    const page = log.filter((c) => c.serverSeq > last).slice(0, Math.min(body.limit ?? PAGE, PAGE));
    const serverSeq = page.length ? page.at(-1).serverSeq : last;
    return send(res, 200, {
      changes: childFirst(page),
      serverSeq,
      hasMore: log.some((c) => c.serverSeq > serverSeq),
    });
  },

  'POST /sync/bootstrap': (_req, res, body) => {
    let position = { s: maxSeq(), i: 0 };
    if (body.cursor) {
      try {
        position = JSON.parse(Buffer.from(body.cursor, 'base64url').toString());
      } catch {
        return error(res, 400, 'invalid_cursor');
      }
    }
    const snapshot = childFirst(
      [...rows.values()]
        .filter((r) => r.serverSeq <= position.s)
        .sort((a, b) => key(a.table, a.rowId).localeCompare(key(b.table, b.rowId))),
    );
    const page = snapshot.slice(position.i, position.i + PAGE);
    const next = position.i + page.length;
    const hasMore = next < snapshot.length;
    return send(res, 200, {
      rows: page.map((r) => ({ ...r, row: wireRow(r) })),
      serverSeq: position.s,
      hasMore,
      ...(hasMore
        ? { cursor: Buffer.from(JSON.stringify({ s: position.s, i: next })).toString('base64url') }
        : {}),
    });
  },

  // --- test controls, not part of the protocol -----------------------------------------------

  /** Writes a row as another device would: `{ table, row }` (wire shape, row.id required). */
  'POST /__stub/write': (_req, res, body) => {
    const current = rows.get(key(body.table, body.row.id));
    const version = (current?.version ?? 0) + 1;
    const serverSeq = append({
      table: body.table,
      rowId: body.row.id,
      op: body.row.deletedAt ? 'delete' : 'upsert',
      row: body.row,
      version,
      updatedAtMs: body.row.updatedAtMs ?? Date.now(),
      lastWriterId: body.lastWriterId ?? 'device:other',
    });
    return send(res, 200, { serverSeq, version });
  },

  /** Pretends retention deleted the log up to `{ seq }`, so older cursors get 410. */
  'POST /__stub/purge': (_req, res, body) => {
    purgedThroughSeq = Number(body.seq ?? maxSeq());
    return send(res, 200, { purgedThroughSeq });
  },

  /** Sets how many of the next sync requests answer 503. */
  'POST /__stub/fail503': (_req, res, body) => {
    fail503 = Number(body.times ?? 0);
    return send(res, 200, { fail503 });
  },

  /** Everything the stub holds, for comparing with the desktop database. */
  'GET /__stub/state': (_req, res) =>
    send(res, 200, { rows: [...rows.values()].map(wireRow), log: log.length, purgedThroughSeq }),
};

async function handle(req, res) {
  const path = new URL(req.url, 'http://stub').pathname;
  const route = `${req.method} ${path}`;
  const handler = handlers[route];
  const body = req.method === 'POST' ? await readBody(req) : {};

  const respond = (status) => {
    // Header presence only, never values: this log is evidence for V-8 and must not hold tokens.
    console.log(
      JSON.stringify({
        route,
        status,
        version: req.headers['x-croffle-sync-version'] ?? null,
        client: req.headers['x-croffle-client'] ? 'present' : 'absent',
        authorization: req.headers.authorization ? 'present' : 'absent',
      }),
    );
  };
  const originalWriteHead = res.writeHead.bind(res);
  res.writeHead = (status, ...rest) => {
    respond(status);
    return originalWriteHead(status, ...rest);
  };

  if (!handler) {
    return error(res, 404, 'not_found');
  }
  if (body === undefined) {
    return error(res, 400, 'invalid_json');
  }
  if (path.startsWith('/sync/')) {
    if (!authorised(req)) {
      return error(res, 401, 'unauthorized');
    }
    if (fail503 > 0) {
      fail503--;
      return error(res, 503, 'unavailable');
    }
  }
  return handler(req, res, body);
}

const server =
  env.TLS_CERT && env.TLS_KEY
    ? https.createServer(
        { cert: readFileSync(env.TLS_CERT), key: readFileSync(env.TLS_KEY) },
        handle,
      )
    : http.createServer(handle);

server.listen(PORT, () => {
  const scheme = env.TLS_CERT ? 'https' : 'http';
  console.error(`stub sync server on ${scheme}://localhost:${PORT} (user ${USERNAME})`);
});
