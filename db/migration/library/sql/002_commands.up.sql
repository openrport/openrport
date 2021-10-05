-- ----------------------------
-- Table structure for commands
-- ----------------------------
create table commands
(
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT not null,
    created_at DATE not null,
    created_by TEXT not null,
    updated_at DATE not null,
    updated_by TEXT not null,
    cmd TEXT not null
);

CREATE INDEX "commands__name"
    ON `commands` (
    "name" ASC
    );

CREATE UNIQUE INDEX "commands__unique_name"
    ON `commands` (
    "name" ASC
);
