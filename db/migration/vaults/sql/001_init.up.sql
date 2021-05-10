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
    "updated_at"     DATE,
    "updated_by"     TEXT,
    "key"            TEXT    NOT NULL,
    "value"          TEXT    NOT NULL,
    "type"           TEXT    NOT NULL
);
-- ----------------------------
-- Auto increment value for vault
-- ----------------------------
UPDATE "main"."sqlite_sequence"
SET seq = 1
WHERE name = 'values';
-- ----------------------------
-- Indexes structure for table vault
-- ----------------------------
CREATE INDEX "main"."client_id"
    ON "values" (
                 "client_id" ASC
        );
CREATE INDEX "main"."key"
    ON "values" (
                 "key" ASC
        );
CREATE UNIQUE INDEX "main"."unique_client_id_key"
    ON "values" (
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
UPDATE "main"."sqlite_sequence" SET seq = 1 WHERE name = 'status';
