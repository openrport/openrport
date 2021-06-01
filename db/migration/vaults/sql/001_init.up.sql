-- ----------------------------
-- Table structure for vault
-- ----------------------------
CREATE TABLE "values"
(
    "id"             INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "client_id"      text    NOT NULL DEFAULT 0,
    "required_group" TEXT,
    "created_at"     DATE    NOT NULL,
    "created_by"     TEXT    NOT NULL,
    "updated_at"     DATE    NOT NULL,
    "updated_by"     TEXT,
    "key"            TEXT    NOT NULL,
    "value"          TEXT    NOT NULL,
    "type"           TEXT    NOT NULL
);
-- ----------------------------
-- Auto increment value for vault
-- ----------------------------
UPDATE `sqlite_sequence`
SET `seq` = 1
    WHERE `name` = 'values';
-- ----------------------------
-- Indexes structure for table vault
-- ----------------------------
CREATE INDEX "client_id"
    ON `values` (
    "client_id" ASC
    );
CREATE INDEX "key"
    ON `values` (
    "key" ASC
    );
CREATE UNIQUE INDEX "unique_client_id_key"
    ON `values` (
    "client_id" ASC,
    "key" ASC
    );

CREATE TABLE "status"
(
    "id"        INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "db_status" text    NOT NULL,
    "enc_check" TEXT,
    "dec_check" TEXT
);

-- ----------------------------
-- Auto increment value for status
-- ----------------------------
UPDATE `sqlite_sequence`
SET `seq` = 1
WHERE `name` = 'status';
