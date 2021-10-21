-- ----------------------------
-- Table structure for scripts
-- ----------------------------
create table scripts
(
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT not null,
    created_at DATE not null,
    created_by TEXT not null,
    interpreter TEXT,
    is_sudo INTEGER(1) default 0 not null,
    cwd TEXT,
    script TEXT not null
);

CREATE INDEX "name"
    ON `scripts` (
    "name" ASC
    );

CREATE UNIQUE INDEX "unique_name"
    ON `scripts` (
    "name" ASC
);
