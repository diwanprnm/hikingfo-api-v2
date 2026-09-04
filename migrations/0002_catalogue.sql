-- +goose Up
-- Catalogue context: mountains, routes, basecamps, weather cache, revisions.

-- Enums
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE mountain_region AS ENUM (
        'Jawa', 'Sumatra', 'Bali & Nusa Tenggara', 'Kalimantan',
        'Sulawesi', 'Maluku', 'Papua'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE mountain_status AS ENUM ('draft', 'published', 'hidden');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- Mountains
CREATE TABLE IF NOT EXISTS mountains (
    id              uuid PRIMARY KEY,
    slug            text NOT NULL UNIQUE,
    name            jsonb NOT NULL DEFAULT '{}'::jsonb,
    aliases         text[] NOT NULL DEFAULT '{}',
    region          mountain_region NOT NULL,
    province        text NOT NULL DEFAULT '',
    location        jsonb NOT NULL DEFAULT '{}'::jsonb,
    latitude        numeric NOT NULL DEFAULT 0,
    longitude       numeric NOT NULL DEFAULT 0,
    peak_name       jsonb NOT NULL DEFAULT '{}'::jsonb,
    peak_height_m   int NOT NULL DEFAULT 0,
    difficulty      smallint NOT NULL DEFAULT 1 CHECK (difficulty BETWEEN 1 AND 5),
    status          mountain_status NOT NULL DEFAULT 'draft',
    data_meta       jsonb NOT NULL DEFAULT '{}'::jsonb,
    search_text     text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS mountains_slug_idx ON mountains(slug);
CREATE INDEX IF NOT EXISTS mountains_region_idx ON mountains(region);
CREATE INDEX IF NOT EXISTS mountains_status_idx ON mountains(status);

-- Trigram search index over search_text (name_id + name_en + aliases)
CREATE INDEX IF NOT EXISTS mountains_search_idx ON mountains USING gin(search_text gin_trgm_ops);

-- Mountain routes
CREATE TABLE IF NOT EXISTS mountain_routes (
    id                uuid PRIMARY KEY,
    mountain_id       uuid NOT NULL REFERENCES mountains(id) ON DELETE CASCADE,
    name              jsonb NOT NULL DEFAULT '{}'::jsonb,
    distance_km       numeric NOT NULL DEFAULT 0,
    duration_hours    numeric NOT NULL DEFAULT 0,
    elevation_gain_m  int NOT NULL DEFAULT 0,
    entry_requirements jsonb NOT NULL DEFAULT '{}'::jsonb,
    sort_order        int NOT NULL DEFAULT 0,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS mountain_routes_mountain_idx ON mountain_routes(mountain_id, sort_order);

-- Mountain basecamps
CREATE TABLE IF NOT EXISTS mountain_basecamps (
    id              uuid PRIMARY KEY,
    mountain_id     uuid NOT NULL REFERENCES mountains(id) ON DELETE CASCADE,
    name            jsonb NOT NULL DEFAULT '{}'::jsonb,
    facilities      jsonb NOT NULL DEFAULT '{}'::jsonb,
    cost_estimate   jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_permit_point boolean NOT NULL DEFAULT false,
    latitude        numeric,
    longitude       numeric,
    sort_order      int NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS mountain_basecamps_mountain_idx ON mountain_basecamps(mountain_id, sort_order);

-- Mountain weather cache
CREATE TABLE IF NOT EXISTS mountain_weather (
    mountain_id   uuid PRIMARY KEY REFERENCES mountains(id) ON DELETE CASCADE,
    payload       jsonb NOT NULL DEFAULT '{}'::jsonb,
    captured_at   timestamptz NOT NULL DEFAULT now(),
    is_live       boolean NOT NULL DEFAULT true,
    expires_at    timestamptz NOT NULL DEFAULT now()
);

-- Mountain revision history (full-row snapshots for rollback)
CREATE TABLE IF NOT EXISTS mountain_revisions (
    id          uuid PRIMARY KEY,
    mountain_id uuid NOT NULL REFERENCES mountains(id) ON DELETE CASCADE,
    snapshot    jsonb NOT NULL,
    revised_by  uuid,
    reason      text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS mountain_revisions_mountain_idx ON mountain_revisions(mountain_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS mountain_revisions;
DROP TABLE IF EXISTS mountain_weather;
DROP TABLE IF EXISTS mountain_basecamps;
DROP TABLE IF EXISTS mountain_routes;
DROP TABLE IF EXISTS mountains;
DROP TYPE IF EXISTS mountain_status;
DROP TYPE IF EXISTS mountain_region;
