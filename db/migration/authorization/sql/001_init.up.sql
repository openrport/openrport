CREATE TABLE api_token (
    id TEXT PRIMARY KEY NOT NULL,
    username TEXT NOT NULL,
    prefix TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME,
    scope TEXT,
    token TEXT NOT NULL,
) WITHOUT ROWID; -- username + prefix must have a unique key

CREATE UNIQUE INDEX idx_api_token_username_prefix
    ON jobs (username, prefix);