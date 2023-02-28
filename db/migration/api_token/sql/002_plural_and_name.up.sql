ALTER TABLE api_token RENAME TO api_tokens;
ALTER TABLE api_tokens add name TEXT NOT NULL;
CREATE UNIQUE INDEX api_tokens_unique_name
    ON api_tokens (username, name);