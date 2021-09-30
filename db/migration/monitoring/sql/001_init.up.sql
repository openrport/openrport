-- ----------------------------
-- Table structure for measurements
-- ----------------------------
CREATE TABLE IF NOT EXISTS "measurements"
(
    "client_id"             TEXT  PRIMARY KEY  NOT NULL,
    "timestamp"             INTEGER PRIMARY KEY NULL ,
    "cpu_usage_percent"     REAL    NOT NULL,
    "memory_usage_percent"  REAL    NOT NULL,
    "io_usage_percent"      REAL    NOT NULL,
    "processes"             TEXT,
    "mountpoints"           TEXT
) WITHOUT ROWID ;
