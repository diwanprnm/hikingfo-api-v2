package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// ErrNotFound is returned by repo methods when a row does not exist.
// Sentinel errors live in the domain so application services can branch on
// them without importing an infrastructure package.
type ErrNotFound struct{ Kind string }

func (e ErrNotFound) Error() string { return "identity: " + e.Kind + " not found" }

// ErrDuplicate is returned by Create when a unique constraint (email, google_sub)
// is violated.
var ErrDuplicate = errDuplicate{}

type errDuplicate struct{}

func (errDuplicate) Error() string { return "identity: duplicate row" }

// UserRepository is the persistence port for users.
type UserRepository interface {
	// Create persists a new user (fails on duplicate email/google_sub).
	Create(ctx context.Context, u *User) error
	// FindByID loads a user by id.
	FindByID(ctx context.Context, id ids.ID) (*User, error)
	// FindByEmail loads a user by verified-normalised email.
	FindByEmail(ctx context.Context, email string) (*User, error)
	// FindByGoogleSub loads a user linked to a Google OIDC subject.
	FindByGoogleSub(ctx context.Context, sub string) (*User, error)
	// Update persists mutable user fields.
	Update(ctx context.Context, u *User) error
	// SetStatus suspends/bans/reactivates a user (admin + moderation).
	SetStatus(ctx context.Context, id ids.ID, status Status) error
	// TouchLastSeen records activity without bumping updated_at.
	TouchLastSeen(ctx context.Context, id ids.ID, at time.Time) error
}

// ContactRepository is the persistence port for the private 1:1 contact row.
type ContactRepository interface {
	// Get loads the private channels for a user (empty when none set).
	Get(ctx context.Context, userID ids.ID) (ContactChannel, error)
	// Upsert sets/replaces the private channels for a user.
	Upsert(ctx context.Context, userID ids.ID, c ContactChannel) error
}

// SessionRepository is the persistence port for opaque sessions.
type SessionRepository interface {
	// Create persists a session row.
	Create(ctx context.Context, s *Session) error
	// FindByTokenHash loads a session by its persisted token hash.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	// Revoke marks a session revoked (logout).
	Revoke(ctx context.Context, id ids.ID) error
	// RevokeByHash revokes a session by its token hash (stateless logout).
	RevokeByHash(ctx context.Context, tokenHash string) error
	// RevokeAllForUser logs out every session of a user (suspend/ban).
	RevokeAllForUser(ctx context.Context, userID ids.ID) error
}

// TokenStore is the persistence port for single-use email tokens.
type TokenStore interface {
	// SaveEmailVerification stores a verification token row.
	SaveEmailVerification(ctx context.Context, userID ids.ID, tokenHash string, expiresAt time.Time) error
	// SavePasswordReset stores a reset token row.
	SavePasswordReset(ctx context.Context, userID ids.ID, tokenHash string, expiresAt time.Time) error
	// ConsumeEmailVerification marks a verification token used and returns the user.
	ConsumeEmailVerification(ctx context.Context, tokenHash string) (userID ids.ID, err error)
	// ConsumePasswordReset marks a reset token used and returns the user.
	ConsumePasswordReset(ctx context.Context, tokenHash string) (userID ids.ID, err error)
}

// PasswordHasher hashes and verifies passwords (argon2id; research.md §2).
// The domain declares the port; the infrastructure package provides the
// concrete argon2id implementation.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) (bool, error)
}
