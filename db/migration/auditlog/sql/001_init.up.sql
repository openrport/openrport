-- ----------------------------
-- Table structure for auditlog
-- ----------------------------
CREATE TABLE "auditlog"
(
    "timestamp"       DATE NOT NULL,
	"application"     TEXT NOT NULL,
	"action"          TEXT NOT NULL,
	"username"        TEXT NULL,
	"remote_ip"       TEXT NULL,
	"affected_id"     TEXT NULL,
	"client_id"       TEXT NULL,
	"client_hostname" TEXT NULL,
	"request"         TEXT NULL,
	"response"        TEXT NULL
);

CREATE INDEX "auditlog_timestamp_application_action_affedted_id" ON `auditlog` (
    "timestamp" ASC,
    "application" ASC,
    "action" ASC,
    "affected_id" ASC
);

CREATE INDEX "auditlog_client_id" ON `auditlog` (
    "client_id" ASC
);

CREATE INDEX "auditlog_client_hostname" ON `auditlog` (
    "client_hostname" ASC
);

CREATE INDEX "auditlog_username" ON `auditlog` (
    "client_username" ASC
);

CREATE INDEX "auditlog_remote_ip" ON `auditlog` (
    "client_remote_ip" ASC
);
