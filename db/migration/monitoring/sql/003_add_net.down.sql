-- ----------------------------
-- drop net columns
-- ----------------------------
ALTER TABLE "measurements" DROP COLUMN "net_lan_in";
ALTER TABLE "measurements" DROP COLUMN "net_lan_out";
ALTER TABLE "measurements" DROP COLUMN "net_wan_in";
ALTER TABLE "measurements" DROP COLUMN "net_wan_out";
