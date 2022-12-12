CREATE TABLE api_token (
    username TEXT NOT NULL,
    prefix TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME,
    scope TEXT,
    token TEXT NOT NULL,
    PRIMARY KEY (username, prefix)
) WITHOUT ROWID; -- username + prefix must have a unique key

-- EDTODO: check if you still need to declare an index 
-- CREATE UNIQUE INDEX idx_api_token_username_prefix
--     ON jobs (username, prefix);