-- ----------------------------
-- Indexes for table measurements
-- ----------------------------
CREATE INDEX "measurements_timestamp" ON `measurements` (
    "timestamp" ASC
);

CREATE INDEX "measurements_client_id" ON `measurements` (
    "client_id" ASC
);
