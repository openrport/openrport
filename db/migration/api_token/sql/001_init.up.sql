CREATE TABLE api_token (
    username TEXT NOT NULL CHECK (username != ''),
    prefix TEXT NOT NULL CHECK (prefix != ''),
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    expires_at DATETIME,
    scope TEXT,
    token TEXT NOT NULL,
    PRIMARY KEY (username, prefix)
) WITHOUT ROWID;

CREATE UNIQUE INDEX api_token_unique_name
    ON api_token (username, name);