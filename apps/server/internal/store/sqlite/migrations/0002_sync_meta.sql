-- Bookkeeping that is neither a row nor a commit.
--
-- purgedThroughSeq is the highest commit sequence the retention sweep has
-- deleted. Pull compares a client cursor against it to answer the one question
-- it cannot otherwise answer: "are the commits this client still needs gone?"
--
-- Deriving that from MIN(serverSeq) alone does not work once the log is empty,
-- and reading SQLite's internal sqlite_sequence would tie the answer to one
-- engine. An explicit counter is both exact and portable.
CREATE TABLE sync_meta (
    key   TEXT    NOT NULL PRIMARY KEY,
    value INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

INSERT INTO sync_meta (key, value) VALUES ('purgedThroughSeq', 0);
