DROP TABLE IF EXISTS api_tokens;
DROP INDEX IF EXISTS api_tokens_unique_name;

CREATE TABLE IF NOT EXISTS api_tokens (
    username TEXT NOT NULL CHECK (username != ''),
    prefix TEXT NOT NULL CHECK (prefix != ''),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    expires_at DATETIME,
    scope TEXT,
    token TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (username, prefix)
) WITHOUT ROWID;
CREATE UNIQUE INDEX IF NOT EXISTS api_tokens_unique_name
    ON api_tokens (username, name);
