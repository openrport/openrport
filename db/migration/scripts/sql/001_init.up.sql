-- ----------------------------
-- Table structure for scripts
-- ----------------------------
create table scripts
(
    id INTEGER not null primary key autoincrement,
    name TEXT,
    created_at DATE not null,
    created_by TEXT not null,
    interpreter TEXT not null,
    is_sudo INTEGER(1) default 0 not null,
    cwd TEXT not null,
    script TEXT not null
);

UPDATE `sqlite_sequence`
SET `seq` = 1
    WHERE `name` = 'scripts';

CREATE INDEX "name"
    ON `scripts` (
    "name" ASC
    );
