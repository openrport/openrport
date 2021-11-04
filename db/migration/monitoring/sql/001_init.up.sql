-- ----------------------------
-- Table structure for measurements
-- ----------------------------
CREATE TABLE IF NOT EXISTS "measurements"
(
    "client_id"             TEXT        NOT NULL,
    "timestamp"             DATETIME    NOT NULL ,
    "cpu_usage_percent"     REAL        NOT NULL,
    "memory_usage_percent"  REAL        NOT NULL,
    "io_usage_percent"      REAL        NOT NULL,
    "processes"             TEXT,
    "mountpoints"           TEXT,
    PRIMARY KEY (client_id, timestamp)
);
