-- Dashboard sessions.
--
-- Deliberately not the same credential as a device token. A sync token can
-- push and pull; if a browser kept one in a cookie or in local storage, an XSS
-- in the dashboard would hand an attacker the ability to rewrite someone's
-- calendar. A session is read-only by construction: the dashboard API accepts
-- nothing else, and nothing else accepts a session.
--
-- Only the hash is stored, for the same reason device tokens are hashed: a copy
-- of the database must not be convertible into a working credential.
CREATE TABLE dash_session (
    tokenHash BLOB    NOT NULL PRIMARY KEY,
    accountId TEXT    NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    createdAt INTEGER NOT NULL,
    expiresAt INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

CREATE INDEX dash_session_accountId_idx ON dash_session (accountId);
CREATE INDEX dash_session_expiresAt_idx ON dash_session (expiresAt);
