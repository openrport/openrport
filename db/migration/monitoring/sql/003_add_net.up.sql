-- ----------------------------
-- add net columns
-- ----------------------------
ALTER TABLE "measurements" ADD COLUMN "net_lan_in" INTEGER;
ALTER TABLE "measurements" ADD COLUMN "net_lan_out" INTEGER;
ALTER TABLE "measurements" ADD COLUMN "net_wan_in" INTEGER;
ALTER TABLE "measurements" ADD COLUMN "net_wan_out" INTEGER;
