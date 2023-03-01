DROP INDEX api_tokens_unique_name;
ALTER TABLE api_tokens drop COLUMN name;
ALTER TABLE api_tokens RENAME TO api_token;
