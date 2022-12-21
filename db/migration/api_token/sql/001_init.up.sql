CREATE TABLE api_token (
    username TEXT NOT NULL CHECK (username != ''),
    prefix TEXT NOT NULL CHECK (prefix != ''),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    expires_at DATETIME,
    scope TEXT,
    token TEXT NOT NULL,
    PRIMARY KEY (username, prefix)
) WITHOUT ROWID; -- username + prefix must have a unique key

-- EDTODO: check if you still need to declare an index 
-- CREATE UNIQUE INDEX idx_api_token_username_prefix
--     ON jobs (username, prefix);