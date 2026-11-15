-- Token expiry and access/refresh pairs (server 0.1.1).
--
-- device.tokenExpiresAt: NULL means the device token never expires. Every
-- token issued by 0.1.0 is NULL here and keeps working; an expiry only appears
-- on tokens issued after the operator sets CROFFLE_DEVICE_TOKEN_TTL.
ALTER TABLE device ADD COLUMN tokenExpiresAt INTEGER;

-- Short-lived access tokens and the refresh tokens that renew them.
--
-- Only hashes are stored, as for device tokens. A refresh token is used once:
-- usedAt is stamped when it is exchanged, and a second exchange of the same
-- token revokes its whole family, because only a stolen copy is ever replayed.
-- familyId groups every pair descended from one sign-in.
CREATE TABLE auth_token (
    tokenHash BLOB    NOT NULL PRIMARY KEY,
    deviceId  TEXT    NOT NULL REFERENCES device (id) ON DELETE CASCADE,
    kind      TEXT    NOT NULL CHECK (kind IN ('access', 'refresh')),
    familyId  TEXT    NOT NULL,
    createdAt INTEGER NOT NULL,
    expiresAt INTEGER NOT NULL,
    usedAt    INTEGER,
    revokedAt INTEGER
) STRICT, WITHOUT ROWID;

CREATE INDEX auth_token_deviceId_idx ON auth_token (deviceId);
CREATE INDEX auth_token_familyId_idx ON auth_token (familyId);
CREATE INDEX auth_token_expiresAt_idx ON auth_token (expiresAt);
