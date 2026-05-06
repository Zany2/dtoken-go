-- DToken PostgreSQL storage schema
CREATE TABLE IF NOT EXISTS "dtoken_storage" (
    "key" TEXT PRIMARY KEY,
    "value" BYTEA NOT NULL,
    "expires_at" TIMESTAMPTZ NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Speed up expired data cleanup and TTL queries
CREATE INDEX IF NOT EXISTS "dtoken_storage_expires_at_idx"
    ON "dtoken_storage" ("expires_at");

-- Optional cleanup SQL
-- DELETE FROM "dtoken_storage" WHERE "expires_at" <= NOW();
