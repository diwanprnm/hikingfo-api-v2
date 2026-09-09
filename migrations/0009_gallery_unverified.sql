-- +goose Up
-- 004: curated mountain photo gallery + hike status 'unverified'.
-- data-model.md (004) §Migration; research.md R3/R7.

CREATE TABLE IF NOT EXISTS mountain_photos (
    id          uuid PRIMARY KEY,
    mountain_id uuid   NOT NULL REFERENCES mountains(id) ON DELETE CASCADE,
    photo_key   text   NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS mountain_photos_mountain_idx ON mountain_photos(mountain_id, created_at);

-- hike status gains 'unverified' (FR-005, derived at record time).
-- Column is the hike_status enum (0003), not a CHECK constraint.
ALTER TYPE hike_status ADD VALUE IF NOT EXISTS 'unverified';

-- +goose Down
-- Postgres cannot remove an enum value; 'unverified' survives rollback.
DROP TABLE IF EXISTS mountain_photos;