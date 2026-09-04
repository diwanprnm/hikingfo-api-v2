-- +goose Up
CREATE TYPE partner_notice_status AS ENUM ('open', 'matched', 'expired', 'withdrawn');
CREATE TYPE partner_request_status AS ENUM ('pending', 'accepted', 'declined', 'expired', 'withdrawn');

CREATE TABLE partner_notices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id),
    mountain_id uuid NOT NULL REFERENCES mountains(id),
    trip_start date NOT NULL,
    trip_end date NOT NULL,
    note text,
    status partner_notice_status NOT NULL DEFAULT 'open',
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_partner_notices_mountain ON partner_notices(mountain_id);
CREATE INDEX idx_notices_active ON partner_notices(status, expires_at) WHERE status = 'open';

CREATE TABLE partner_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    from_user_id uuid NOT NULL REFERENCES users(id),
    to_user_id uuid NOT NULL REFERENCES users(id),
    notice_id uuid REFERENCES partner_notices(id),
    mountain_id uuid NOT NULL REFERENCES mountains(id),
    trip_start date NOT NULL,
    trip_end date NOT NULL,
    message text,
    status partner_request_status NOT NULL DEFAULT 'pending',
    matched_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX idx_requests_from ON partner_requests(from_user_id);
CREATE INDEX idx_requests_to ON partner_requests(to_user_id);
CREATE INDEX idx_requests_active ON partner_requests(status, expires_at) WHERE status = 'pending';

CREATE TRIGGER set_updated_at BEFORE UPDATE ON partner_notices FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS partner_requests;
DROP TABLE IF EXISTS partner_notices;
DROP TYPE IF EXISTS partner_request_status;
DROP TYPE IF EXISTS partner_notice_status;
