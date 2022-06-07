ALTER TABLE "scripts" ADD COLUMN "updated_by" TEXT NOT NULL DEFAULT '';
ALTER TABLE "scripts" ADD COLUMN "updated_at" DATE NOT NULL DEFAULT now;
UPDATE scripts SET updated_at = created_at, updated_by = created_by;

ALTER TABLE "commands" ADD COLUMN "tags" TEXT NOT NULL DEFAULT '[]';
ALTER TABLE "scripts" ADD COLUMN "tags" TEXT NOT NULL DEFAULT '[]';
