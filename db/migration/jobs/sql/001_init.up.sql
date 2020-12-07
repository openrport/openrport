CREATE TABLE jobs (
    jid TEXT PRIMARY KEY NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    finished_at DATETIME,
    created_by TEXT NOT NULL,
    sid TEXT NOT NULL,
    details TEXT
) WITHOUT ROWID;

CREATE INDEX idx_jobs_sid_time
    ON jobs (sid, finished_at DESC);
