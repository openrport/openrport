CREATE TABLE client_sessions (
    id TEXT PRIMARY KEY NOT NULL,
	client_id TEXT NOT NULL,
	disconnected DATETIME,
	details TEXT NOT NULL
) WITHOUT ROWID;

CREATE INDEX idx_disconnected_client
    ON client_sessions (disconnected DESC, client_id);

CREATE INDEX idx_disconnected_time_client
	ON client_sessions (DATETIME(disconnected) DESC, client_id);
