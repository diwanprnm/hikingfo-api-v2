package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
)

// SessionRepository is the pgx adapter for domain.SessionRepository.
type SessionRepository struct{ pool *pgxpool.Pool }

// NewSessionRepository wires the adapter over a shared pool.
func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

var _ domain.SessionRepository = (*SessionRepository)(nil)

const sessionColumns = `id::text, user_id::text, token_hash, csrf_token,
	created_at, expires_at, revoked_at`

// Create persists a session row.
func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, csrf_token, created_at, expires_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)`,
		string(s.ID), string(s.UserID), s.TokenHash, s.CSRFToken, s.CreatedAt, s.ExpiresAt,
	)
	return err
}

// FindByTokenHash loads a session by its persisted token hash.
func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	var (
		id, userID, tokenHashOut, csrf string
		createdAt, expiresAt           time.Time
		revokedAt                      *time.Time
	)
	err := r.pool.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM sessions WHERE token_hash = $1`,
		tokenHash,
	).Scan(&id, &userID, &tokenHashOut, &csrf, &createdAt, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound{Kind: "session"}
		}
		return nil, err
	}
	return &domain.Session{
		ID:        ids.ID(id),
		UserID:    ids.ID(userID),
		TokenHash: tokenHashOut,
		CSRFToken: csrf,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		RevokedAt: revokedAt,
	}, nil
}

// Revoke marks a session revoked (logout).
func (r *SessionRepository) Revoke(ctx context.Context, id ids.ID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE id = $1::uuid`, string(id))
	return err
}

// RevokeByHash revokes a session by its token hash (stateless logout).
func (r *SessionRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE token_hash = $1`, tokenHash)
	return err
}

// RevokeAllForUser logs out every session of a user (suspend/ban).
func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID ids.ID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1::uuid`, string(userID))
	return err
}

// TokenStore is the pgx adapter for the single-use email tokens.
type TokenStore struct {
	pool       *pgxpool.Pool
	table      string
	kind       string
}

// NewEmailVerificationStore backs email_verification_tokens.
func NewEmailVerificationStore(pool *pgxpool.Pool) *TokenStore {
	return &TokenStore{pool: pool, table: "email_verification_tokens", kind: "email verification token"}
}

// NewPasswordResetStore backs password_reset_tokens.
func NewPasswordResetStore(pool *pgxpool.Pool) *TokenStore {
	return &TokenStore{pool: pool, table: "password_reset_tokens", kind: "password reset token"}
}

var _ domain.TokenStore = (*TokenStore)(nil)

// SaveEmailVerification stores a verification token row.
func (s *TokenStore) SaveEmailVerification(ctx context.Context, userID ids.ID, tokenHash string, expiresAt time.Time) error {
	return s.save(ctx, userID, tokenHash, expiresAt)
}

// SavePasswordReset stores a reset token row.
func (s *TokenStore) SavePasswordReset(ctx context.Context, userID ids.ID, tokenHash string, expiresAt time.Time) error {
	return s.save(ctx, userID, tokenHash, expiresAt)
}

func (s *TokenStore) save(ctx context.Context, userID ids.ID, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO `+s.table+` (id, user_id, token_hash, expires_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4)`,
		string(ids.New()), string(userID), tokenHash, expiresAt,
	)
	return err
}

// ConsumeEmailVerification marks a verification token used and returns the user.
func (s *TokenStore) ConsumeEmailVerification(ctx context.Context, tokenHash string) (ids.ID, error) {
	return s.consume(ctx, tokenHash)
}

// ConsumePasswordReset marks a reset token used and returns the user.
func (s *TokenStore) ConsumePasswordReset(ctx context.Context, tokenHash string) (ids.ID, error) {
	return s.consume(ctx, tokenHash)
}

func (s *TokenStore) consume(ctx context.Context, tokenHash string) (ids.ID, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`UPDATE `+s.table+`
		   SET used_at = now()
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
		 RETURNING user_id::text`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ids.Nil, domain.ErrNotFound{Kind: s.kind}
		}
		return ids.Nil, err
	}
	return ids.ID(userID), nil
}
