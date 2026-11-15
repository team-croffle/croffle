-- One-time codes (TOTP) and recovery codes (server 0.1.1).
--
-- A row with enabledAt NULL is an enrolment in progress: the secret has been
-- shown but no code has confirmed it yet, so sign-in does not ask for one.
-- lastStep is the highest 30-second step a code was accepted for; a code for
-- that step or an earlier one is refused, so an observed code cannot be
-- replayed.
--
-- The secret is stored as is. Encrypting it needs a key the operator manages,
-- which is planned for a later version; until then the database file is the
-- trust boundary, as it already is for the operator commands.
CREATE TABLE account_otp (
    accountId TEXT    NOT NULL PRIMARY KEY REFERENCES account (id) ON DELETE CASCADE,
    secret    TEXT    NOT NULL,
    enabledAt INTEGER,
    lastStep  INTEGER NOT NULL DEFAULT 0,
    createdAt INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

-- Recovery codes are random (80 bits), so a plain SHA-256 is enough, as for
-- tokens. Each works once.
CREATE TABLE otp_recovery_code (
    codeHash  BLOB    NOT NULL PRIMARY KEY,
    accountId TEXT    NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    createdAt INTEGER NOT NULL,
    usedAt    INTEGER
) STRICT, WITHOUT ROWID;

CREATE INDEX otp_recovery_code_accountId_idx ON otp_recovery_code (accountId);
