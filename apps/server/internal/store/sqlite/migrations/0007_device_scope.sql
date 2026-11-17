-- Team scopes (server 0.2.0).
--
-- The pull cursor is one serverSeq per device. When an account joins a team,
-- its devices' cursors are already past the team's older commits, so a plain
-- pull would never deliver them. A row here says "this device has received
-- this team's rows from the beginning": through a snapshot, or a pull that
-- started at 0. A device whose account is in a team without such a row gets
-- 410 cursor_too_old on its next pull and takes a snapshot, which desktop
-- 1.3.x already does for that answer.
--
-- Every existing device is taken as covered for the teams its account is in
-- now, so upgrading to 0.2.0 does not send every device back to a snapshot.
CREATE TABLE device_scope (
    deviceId  TEXT    NOT NULL REFERENCES device (id) ON DELETE CASCADE,
    teamId    TEXT    NOT NULL REFERENCES team (id) ON DELETE CASCADE,
    coveredAt INTEGER NOT NULL,
    PRIMARY KEY (deviceId, teamId)
) STRICT, WITHOUT ROWID;

CREATE INDEX device_scope_teamId_idx ON device_scope (teamId);

INSERT INTO device_scope (deviceId, teamId, coveredAt)
SELECT d.id, m.teamId, CAST(strftime('%s', 'now') AS INTEGER) * 1000
FROM device d JOIN team_member m ON m.accountId = d.accountId;
