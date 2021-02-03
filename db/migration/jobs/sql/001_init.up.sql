CREATE TABLE IF NOT EXISTS multi_jobs (
    jid TEXT PRIMARY KEY NOT NULL,
    started_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    details TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE jobs (
    jid TEXT PRIMARY KEY NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    finished_at DATETIME,
    created_by TEXT NOT NULL,
    client_id TEXT NOT NULL,
    multi_job_id TEXT,
    details TEXT NOT NULL,
    FOREIGN KEY (multi_job_id) REFERENCES multi_jobs(jid)
) WITHOUT ROWID;

CREATE INDEX idx_jobs_client_id_time
    ON jobs (client_id, finished_at DESC);

CREATE INDEX idx_jobs_multi_id
    ON jobs (multi_job_id);
