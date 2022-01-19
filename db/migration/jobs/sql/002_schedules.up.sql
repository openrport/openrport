CREATE TABLE schedules (
    id TEXT PRIMARY KEY NOT NULL,
    created_at DATETIME NOT NULL,
    created_by TEXT NOT NULL,
    name TEXT NOT NULL,
    schedule TEXT NOT NULL,
    type TEXT NOT NULL,
    details TEXT NOT NULL
);
