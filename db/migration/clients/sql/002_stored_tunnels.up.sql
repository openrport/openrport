CREATE TABLE stored_tunnels (
    id TEXT PRIMARY KEY NOT NULL,
    client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    created_at DATETIME,
    name TEXT,
    scheme TEXT,
    remote_ip TEXT,
    remote_port NUMBER,
    acl TEXT
);
