CREATE TABLE dummy
(
    id              TEXT PRIMARY KEY NOT NULL,
    client_auth_id  TEXT             NOT NULL,
    disconnected_at DATETIME,
    details         TEXT             NOT NULL
) WITHOUT ROWID;