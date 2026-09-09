-- +goose Up
-- 002: mountain cover photo + append-only admin edit log.
-- data-model.md (002) §Migration; research.md R1/R5.

ALTER TABLE mountains ADD COLUMN IF NOT EXISTS photo_key text;

CREATE TABLE IF NOT EXISTS admin_edit_log (
    id          uuid PRIMARY KEY,
    admin_id    uuid   NOT NULL,
    entity_type text   NOT NULL CHECK (entity_type IN ('mountain','route','basecamp')),
    entity_id   uuid   NOT NULL,
    mountain_id uuid   NOT NULL,
    action      text   NOT NULL CHECK (action IN ('create','update','delete','photo_set','photo_remove')),
    reason      text   NOT NULL DEFAULT '',
    detail      jsonb  NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- mountain_id has NO FK on purpose: the audit trail outlives the entity
-- (research.md R7 — deletion must never fail or cascade away provenance).
CREATE INDEX IF NOT EXISTS admin_edit_log_mountain_idx ON admin_edit_log(mountain_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS admin_edit_log;
ALTER TABLE mountains DROP COLUMN IF EXISTS photo_key;
