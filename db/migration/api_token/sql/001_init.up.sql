CREATE TABLE api_token (
    username TEXT NOT NULL CHECK (username != ''),
    prefix TEXT NOT NULL CHECK (prefix != ''),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    expires_at DATETIME,
    scope TEXT,
    token TEXT NOT NULL,
    PRIMARY KEY (username, prefix)
) WITHOUT ROWID;