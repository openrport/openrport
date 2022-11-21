DROP TABLE IF EXISTS api_sessions;

CREATE TABLE api_sessions (
    token TEXT PRIMARY KEY NOT NULL,
    expires_at DATETIME NOT NULL
) WITHOUT ROWID;

CREATE INDEX idx_expires_at_time
	ON api_sessions (DATETIME(expires_at) DESC);
