package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// pgx notes (kept explicit so the adapters never depend on pgx codecs for our
// domain types):
//   - uuid columns are selected as `::text` and bound as `string(...)::uuid`;
//     ids.ID (a string alias) is never handed to pgx directly.
//   - enum columns are selected as `::text` and bound as `string(...)::myenum`.
//   - citext equality binds `$n::citext`.
//   - nullable text/time columns scan into **string / **time.Time (pgx's
//     NULL-aware pattern); a plain *T destination would error on SQL NULL.

// userColumns lists every users row in fixed order.
const userColumns = `id::text, email, password_hash, google_sub, email_verified_at,
	display_name, avatar_key, bio::text, home_region, role, status::text,
	last_seen_at, created_at, updated_at`

// UserRepository is the pgx adapter for domain.UserRepository.
type UserRepository struct{ pool *pgxpool.Pool }

// NewUserRepository wires the adapter over a shared pool.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Compile-time check: the adapter satisfies the domain port.
var _ domain.UserRepository = (*UserRepository)(nil)

// Create persists a new user (unique email/google_sub → ErrDuplicate).
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	bio, err := bioParam(u.Bio)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, google_sub, email_verified_at,
		                  display_name, avatar_key, bio, home_region, role, status,
		                  created_at, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10,
		        $11::user_status, $12, $13)`,
		string(u.ID),
		string(u.Email),
		nilOr(u.PasswordHash),
		nilOr(u.GoogleSub),
		u.EmailVerifiedAt,
		u.DisplayName,
		nilOr(u.AvatarKey),
		bio,
		nilOr(u.HomeRegion),
		string(u.Role),
		string(u.Status),
		u.CreatedAt,
		u.UpdatedAt,
	)
	return classify(err, "user")
}

// FindByID loads a user by id (ErrNotFound when absent).
func (r *UserRepository) FindByID(ctx context.Context, id ids.ID) (*domain.User, error) {
	return r.find(ctx, `WHERE id = $1::uuid`, string(id))
}

// FindByEmail loads a user by its citext email (case-insensitive).
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.find(ctx, `WHERE email = $1::citext`, email)
}

// FindByGoogleSub loads the account linked to a Google OIDC subject.
func (r *UserRepository) FindByGoogleSub(ctx context.Context, sub string) (*domain.User, error) {
	return r.find(ctx, `WHERE google_sub = $1`, sub)
}

func (r *UserRepository) find(ctx context.Context, where string, arg any) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users `+where, arg)
	u, err := scanUser(row)
	if err != nil {
		return nil, classify(err, "user")
	}
	return u, nil
}

// Update persists the mutable user fields. updated_at is maintained by the
// set_updated_at trigger (0001_foundation), which ignores last_seen_at.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	bio, err := bioParam(u.Bio)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, google_sub = $3, email_verified_at = $4,
		                 display_name = $5, avatar_key = $6, bio = $7::jsonb,
		                 home_region = $8, role = $9, status = $10::user_status
		WHERE id = $1::uuid`,
		string(u.ID),
		nilOr(u.PasswordHash),
		nilOr(u.GoogleSub),
		u.EmailVerifiedAt,
		u.DisplayName,
		nilOr(u.AvatarKey),
		bio,
		nilOr(u.HomeRegion),
		string(u.Role),
		string(u.Status),
	)
	return classify(err, "user")
}

// SetStatus suspends/bans/reactivates a user (admin + moderation).
func (r *UserRepository) SetStatus(ctx context.Context, id ids.ID, status domain.Status) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET status = $2::user_status WHERE id = $1::uuid`,
		string(id), string(status))
	return classify(err, "user")
}

// TouchLastSeen records activity. The trigger ignores last_seen_at, so
// updated_at is not bumped (domain.UserRepository contract).
func (r *UserRepository) TouchLastSeen(ctx context.Context, id ids.ID, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET last_seen_at = $2 WHERE id = $1::uuid`, string(id), at)
	return classify(err, "user")
}

// scanUser maps a fixed-order users row onto the domain model. Raw string/time
// locals are used for scan targets; the result is copied onto ids.ID and the
// domain aliases after the scan, so pgx never touches a defined type.
func scanUser(row pgx.Row) (*domain.User, error) {
	var (
		id             string
		email          string
		passwordHash   *string
		googleSub      *string
		avatarKey      *string
		bioJSON        *string
		homeRegion     *string
		displayName    string
		role, status   string
		emailVerified  *time.Time
		lastSeen       *time.Time
		created, up    time.Time
	)
	err := row.Scan(
		&id, &email, &passwordHash, &googleSub, &emailVerified,
		&displayName, &avatarKey, &bioJSON, &homeRegion, &role, &status,
		&lastSeen, &created, &up,
	)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		ID:              ids.ID(id),
		Email:           email,
		PasswordHash:    deref(passwordHash),
		GoogleSub:       deref(googleSub),
		EmailVerifiedAt: emailVerified,
		DisplayName:     displayName,
		AvatarKey:       deref(avatarKey),
		HomeRegion:      deref(homeRegion),
		Role:            domain.Role(role),
		Status:          domain.Status(status),
		LastSeenAt:      lastSeen,
		CreatedAt:       created,
		UpdatedAt:       up,
	}
	if bioJSON != nil {
		var b langtext.Text
		if err := json.Unmarshal([]byte(*bioJSON), &b); err != nil {
			return nil, fmt.Errorf("infra/identity: decode bio: %w", err)
		}
		u.Bio = b
	}
	return u, nil
}

// ContactRepository is the pgx adapter for domain.ContactRepository.
type ContactRepository struct{ pool *pgxpool.Pool }

// NewContactRepository wires the adapter over a shared pool.
func NewContactRepository(pool *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{pool: pool}
}

var _ domain.ContactRepository = (*ContactRepository)(nil)

// Get loads the private channels; a missing row reads as empty (no error).
func (r *ContactRepository) Get(ctx context.Context, userID ids.ID) (domain.ContactChannel, error) {
	var phone, whatsapp, instagram *string
	err := r.pool.QueryRow(ctx,
		`SELECT phone, whatsapp, instagram FROM user_contacts WHERE user_id = $1::uuid`,
		string(userID),
	).Scan(&phone, &whatsapp, &instagram)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ContactChannel{}, nil
		}
		return domain.ContactChannel{}, err
	}
	return domain.ContactChannel{
		Phone:     deref(phone),
		WhatsApp:  deref(whatsapp),
		Instagram: deref(instagram),
	}, nil
}

// Upsert sets/replaces the private channels for a user.
func (r *ContactRepository) Upsert(ctx context.Context, userID ids.ID, c domain.ContactChannel) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_contacts (user_id, phone, whatsapp, instagram)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			phone = EXCLUDED.phone, whatsapp = EXCLUDED.whatsapp,
			instagram = EXCLUDED.instagram, updated_at = now()`,
		string(userID), nilOr(c.Phone), nilOr(c.WhatsApp), nilOr(c.Instagram),
	)
	return classify(err, "contact")
}

// classify maps driver errors onto domain sentinels so application services can
// branch without importing pgx.
func classify(err error, kind string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound{Kind: kind}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" { // unique_violation
			return domain.ErrDuplicate
		}
	}
	return err
}

// bioParam renders a langtext.Text as a JSON string for a ::jsonb cast, or nil
// when the field is empty.
func bioParam(t langtext.Text) (any, error) {
	if !t.Has() {
		return nil, nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("infra/identity: encode bio: %w", err)
	}
	return string(b), nil
}

// nilOr renders an empty string column as SQL NULL.
func nilOr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// deref unwraps a nullable column pointer, empty when absent.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
