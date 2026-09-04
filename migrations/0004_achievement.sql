-- +goose Up
-- Achievement context: badge_configs with default seed data.

CREATE TABLE badge_configs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    key text UNIQUE NOT NULL,
    threshold int NOT NULL,
    name jsonb NOT NULL DEFAULT '{"id":"","en":""}',
    description jsonb NOT NULL DEFAULT '{"id":"","en":""}',
    icon_key text NOT NULL DEFAULT '',
    design text NOT NULL DEFAULT '',
    sort_order int NOT NULL DEFAULT 0,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Seed default badges
INSERT INTO badge_configs (key, threshold, name, description, sort_order) VALUES
('mountain_1',   1,   '{"id":"Pendaki Gunung",  "en":"Mountain Hiker"}',     '{"id":"Telah mendaki 1 gunung",   "en":"Hiked 1 mountain"}',   1),
('mountain_5',   5,   '{"id":"Penjelajah",       "en":"Explorer"}',           '{"id":"Telah mendaki 5 gunung",   "en":"Hiked 5 mountains"}',  2),
('mountain_10',  10,  '{"id":"Petualang",        "en":"Adventurer"}',         '{"id":"Telah mendaki 10 gunung",  "en":"Hiked 10 mountains"}', 3),
('mountain_25',  25,  '{"id":"Juara Gunung",     "en":"Mountain Champion"}',  '{"id":"Telah mendaki 25 gunung",  "en":"Hiked 25 mountains"}', 4),
('mountain_50',  50,  '{"id":"Legenda",           "en":"Legend"}',             '{"id":"Telah mendaki 50 gunung",  "en":"Hiked 50 mountains"}', 5),
('mountain_100', 100, '{"id":"Dewa Gunung",      "en":"Mountain Deity"}',     '{"id":"Telah mendaki 100 gunung", "en":"Hiked 100 mountains"}',6);

-- Updated_at trigger (reuses the existing set_updated_at function from foundation migration).
CREATE TRIGGER badge_configs_set_updated_at BEFORE UPDATE ON badge_configs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS badge_configs;
