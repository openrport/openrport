CREATE TABLE clients (
    id TEXT PRIMARY KEY NOT NULL,
	client_auth_id TEXT NOT NULL,
	disconnected DATETIME,
	details TEXT NOT NULL
) WITHOUT ROWID;

CREATE INDEX idx_disconnected_client
    ON clients (disconnected DESC, client_auth_id);

CREATE INDEX idx_disconnected_time_client
	ON clients (DATETIME(disconnected) DESC, client_auth_id);
