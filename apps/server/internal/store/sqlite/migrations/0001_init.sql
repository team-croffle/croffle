-- Initial schema for the Croffle sync server.
--
-- Timestamps are unix milliseconds throughout, matching the wire format
-- (updatedAtMs) so no conversion happens on the hot path.
--
-- Tables are STRICT: a type mismatch is a bug in the server, and finding it on
-- write is cheaper than finding it in a client that cannot decode a row.

-- An account is a login. 0.1 creates them through the CLI.
CREATE TABLE account (
    id           TEXT    NOT NULL PRIMARY KEY,
    username     TEXT    NOT NULL UNIQUE,
    passwordHash TEXT    NOT NULL,
    createdAt    INTEGER NOT NULL,
    disabledAt   INTEGER
) STRICT;

-- One installation of the desktop app. `id` is the client's own clientId, the
-- same value that appears in a row's lastWriterId as "device:<clientId>".
-- tokenHash is sha256(token); the token itself is never stored.
CREATE TABLE device (
    id         TEXT    NOT NULL PRIMARY KEY,
    accountId  TEXT    NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    tokenHash  BLOB    UNIQUE,
    createdAt  INTEGER NOT NULL,
    lastSeenAt INTEGER,
    revokedAt  INTEGER
) STRICT;

CREATE INDEX device_accountId_idx ON device (accountId);

CREATE TABLE team (
    id        TEXT    NOT NULL PRIMARY KEY,
    name      TEXT    NOT NULL,
    createdAt INTEGER NOT NULL
) STRICT;

CREATE TABLE team_member (
    teamId    TEXT    NOT NULL REFERENCES team (id) ON DELETE CASCADE,
    accountId TEXT    NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    role      TEXT    NOT NULL DEFAULT 'member',
    createdAt INTEGER NOT NULL,
    PRIMARY KEY (teamId, accountId)
) STRICT;

CREATE INDEX team_member_accountId_idx ON team_member (accountId);

-- Current state of a synchronised row: the source for snapshots and for
-- conflict detection. Hybrid on purpose — the columns the server reasons about
-- are columns, the calendar fields stay in the opaque payload.
--
-- The client's row id is stored as `rowKey`, not `rowId`: SQLite identifiers are
-- case-insensitive, so a column named rowId shadows the built-in rowid and
-- `ORDER BY rowid` would silently sort by that text column instead of the
-- primary key. In the commit log that mistake reorders the pull feed, which
-- loses changes. The wire field stays `rowId`; only the column is renamed.
--
-- WITHOUT ROWID because the primary key is the whole identity of the row.
CREATE TABLE sync_row (
    tbl          TEXT    NOT NULL,
    rowKey       TEXT    NOT NULL,
    ownerId      TEXT    NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    teamId       TEXT             REFERENCES team (id) ON DELETE SET NULL,
    version      INTEGER NOT NULL,
    updatedAtMs  INTEGER NOT NULL,
    lastWriterId TEXT,
    deletedAt    INTEGER,
    payload      TEXT    NOT NULL,
    serverSeq    INTEGER NOT NULL,
    PRIMARY KEY (tbl, rowKey)
) STRICT, WITHOUT ROWID;

CREATE INDEX sync_row_ownerId_idx ON sync_row (ownerId);
CREATE INDEX sync_row_teamId_idx ON sync_row (teamId);

-- The commit log is the pull feed. serverSeq is AUTOINCREMENT, not a bare
-- INTEGER PRIMARY KEY, so a deleted row's number is never handed out again:
-- cursors must stay monotonic for the lifetime of the database.
CREATE TABLE commit_log (
    serverSeq    INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    tbl          TEXT    NOT NULL,
    rowKey       TEXT    NOT NULL,
    op           TEXT    NOT NULL,
    ownerId      TEXT    NOT NULL,
    teamId       TEXT,
    version      INTEGER NOT NULL,
    updatedAtMs  INTEGER NOT NULL,
    lastWriterId TEXT,
    payload      TEXT,
    deviceId     TEXT    NOT NULL,
    createdAt    INTEGER NOT NULL
) STRICT;

-- Pull filters by visibility and orders by sequence, hence the composite
-- indexes; createdAt carries the retention sweep.
CREATE INDEX commit_log_ownerId_seq_idx ON commit_log (ownerId, serverSeq);
CREATE INDEX commit_log_teamId_seq_idx ON commit_log (teamId, serverSeq);
CREATE INDEX commit_log_createdAt_idx ON commit_log (createdAt);

-- Idempotency keys. A push whose response was lost is retried with the same
-- opIds; those commits are reported as accepted without being applied again.
CREATE TABLE applied_op (
    opId      TEXT    NOT NULL PRIMARY KEY,
    deviceId  TEXT    NOT NULL,
    serverSeq INTEGER NOT NULL,
    createdAt INTEGER NOT NULL
) STRICT;

CREATE INDEX applied_op_createdAt_idx ON applied_op (createdAt);

-- Server 0.1 has exactly one team (multi-team is 0.3), seeded here so sharing
-- has a target from the first boot.
INSERT INTO team (id, name, createdAt)
VALUES ('team-default', 'Default team', CAST(strftime('%s', 'now') AS INTEGER) * 1000);
