-- +goose Up
-- Journey context: hike_log_entries + journey_posts.

-- Enums
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE hike_status AS ENUM ('verified', 'disputed', 'removed');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE post_visibility AS ENUM ('published', 'draft');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE moderation_status AS ENUM ('visible', 'under_review', 'hidden');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- Hike log entries (evidence of a summit/ridge completion)
CREATE TABLE IF NOT EXISTS hike_log_entries (
    id                  uuid PRIMARY KEY,
    user_id             uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mountain_id         uuid NOT NULL REFERENCES mountains(id) ON DELETE CASCADE,
    route_id            uuid REFERENCES mountain_routes(id) ON DELETE SET NULL,
    climb_date          date NOT NULL,
    evidence_photo_keys text[] NOT NULL DEFAULT '{}',
    status              hike_status NOT NULL DEFAULT 'verified',
    published_post_id   uuid,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS hike_log_entries_user_idx ON hike_log_entries(user_id, climb_date DESC);
CREATE INDEX IF NOT EXISTS hike_log_entries_mountain_idx ON hike_log_entries(mountain_id);

-- Journey posts (user-authored narratives)
CREATE TABLE IF NOT EXISTS journey_posts (
    id                uuid PRIMARY KEY,
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mountain_id       uuid NOT NULL REFERENCES mountains(id) ON DELETE CASCADE,
    route_id          uuid REFERENCES mountain_routes(id) ON DELETE SET NULL,
    title             text NOT NULL DEFAULT '',
    summary           jsonb NOT NULL DEFAULT '{}'::jsonb,
    narrative         jsonb NOT NULL DEFAULT '{}'::jsonb,
    photo_keys        text[] NOT NULL DEFAULT '{}',
    visibility        post_visibility NOT NULL DEFAULT 'draft',
    moderation_status moderation_status NOT NULL DEFAULT 'visible',
    published_at      timestamptz,
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS journey_posts_user_idx ON journey_posts(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS journey_posts_feed_idx ON journey_posts(visibility, moderation_status, published_at DESC);
CREATE INDEX IF NOT EXISTS journey_posts_mountain_idx ON journey_posts(mountain_id);

-- updated_at triggers (reuses set_updated_at function from 0001)
-- We create per-table trigger functions since the 0001 function is column-specific to users.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_hike_log_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS hike_log_entries_set_updated_at ON hike_log_entries;
CREATE TRIGGER hike_log_entries_set_updated_at
    BEFORE UPDATE ON hike_log_entries
    FOR EACH ROW EXECUTE FUNCTION set_hike_log_updated_at();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_journey_post_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS journey_posts_set_updated_at ON journey_posts;
CREATE TRIGGER journey_posts_set_updated_at
    BEFORE UPDATE ON journey_posts
    FOR EACH ROW EXECUTE FUNCTION set_journey_post_updated_at();

-- +goose Down
DROP TABLE IF EXISTS journey_posts;
DROP TABLE IF EXISTS hike_log_entries;

DROP FUNCTION IF EXISTS set_journey_post_updated_at();
DROP FUNCTION IF EXISTS set_hike_log_updated_at();

DROP TYPE IF EXISTS moderation_status;
DROP TYPE IF EXISTS post_visibility;
DROP TYPE IF EXISTS hike_status;
