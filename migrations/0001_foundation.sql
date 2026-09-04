-- +goose Up
-- Foundation schema (task T014): shared enums + identity, notification,
-- moderation tables. Per data-model.md, context ownership is tagged per table.

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS citext;

-- ---------- enums ----------
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE user_status AS ENUM ('active', 'suspended', 'banned');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE notification_type AS ENUM (
        'badge_earned',
        'partner_request_received',
        'partner_request_accepted',
        'moderation_outcome'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE report_target_type AS ENUM (
        'journey_post',
        'user_profile',
        'hike_evidence',
        'mountain_field'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE report_reason AS ENUM (
        'spam',
        'false_info',
        'safety_critical',
        'offensive',
        'other'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE report_status AS ENUM ('open', 'under_review', 'resolved', 'dismissed');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- +goose StatementEnd

-- ---------- identity: users ----------
CREATE TABLE IF NOT EXISTS users (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE,
    password_hash     text,                     -- null when Google-only account
    google_sub        text UNIQUE,              -- OIDC subject for Google sign-in
    email_verified_at timestamptz,
    display_name      text NOT NULL,
    avatar_key        text,                     -- blob-storage key (MinIO avatars)
    bio               jsonb,                    -- optional-public, JSONB{id,en}
    home_region       text,                     -- province/region; optional-public
    role              text NOT NULL DEFAULT 'member', -- 'member' | 'admin' (single admin)
    status            user_status NOT NULL DEFAULT 'active',
    last_seen_at      timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- Private contact channels (email, phone, whatsapp, instagram). Stored on a
-- 1:1 row and returned ONLY to a mutually-matched counterpart (data-model →
-- users → Private). email duplicates users.email for reveal ease.
CREATE TABLE IF NOT EXISTS user_contacts (
    user_id   uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phone     text,
    whatsapp  text,
    instagram text,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- ---------- identity: sessions (opaque, server-side) ----------
CREATE TABLE IF NOT EXISTS sessions (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,            -- sha256 of the opaque cookie token
    csrf_token text NOT NULL,                   -- per-session CSRF token (mutation header)
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions(user_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions(expires_at);

-- ---------- identity: single-use email tokens ----------
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz
);

-- ---------- notification: in-app centre ----------
CREATE TABLE IF NOT EXISTS notifications (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       notification_type NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}'::jsonb,
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_user_created_idx
    ON notifications(user_id, created_at DESC);

-- ---------- moderation: polymorphic reports queue ----------
CREATE TABLE IF NOT EXISTS reports (
    id          uuid PRIMARY KEY,
    reporter_id uuid REFERENCES users(id) ON DELETE SET NULL, -- any visitor → nullable
    target_type report_target_type NOT NULL,
    target_id   uuid NOT NULL,
    field_ref   text,                        -- when target_type = 'mountain_field'
    reason      report_reason NOT NULL,
    detail      text,                        -- ID-only (v1)
    status      report_status NOT NULL DEFAULT 'open',
    resolution  text,                        -- admin note on resolve
    created_at  timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz
);

CREATE INDEX IF NOT EXISTS reports_status_created_idx ON reports(status, created_at);
CREATE INDEX IF NOT EXISTS reports_target_idx ON reports(target_type, target_id);

-- ---------- moderation: directed blocks ----------
CREATE TABLE IF NOT EXISTS blocks (
    blocker_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id)
);

-- Trigger to keep updated_at current on users. It only bumps updated_at when a
-- business column actually changes: TouchLastSeen writes last_seen_at alone,
-- and must not recycle updated_at (identity repo contract).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    IF NEW.password_hash IS DISTINCT FROM OLD.password_hash
       OR NEW.google_sub IS DISTINCT FROM OLD.google_sub
       OR NEW.email_verified_at IS DISTINCT FROM OLD.email_verified_at
       OR NEW.display_name IS DISTINCT FROM OLD.display_name
       OR NEW.avatar_key IS DISTINCT FROM OLD.avatar_key
       OR NEW.bio IS DISTINCT FROM OLD.bio
       OR NEW.home_region IS DISTINCT FROM OLD.home_region
       OR NEW.role IS DISTINCT FROM OLD.role
       OR NEW.status IS DISTINCT FROM OLD.status THEN
        NEW.updated_at := now();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_contacts;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();

DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS report_reason;
DROP TYPE IF EXISTS report_target_type;
DROP TYPE IF EXISTS notification_type;
DROP TYPE IF EXISTS user_status;
