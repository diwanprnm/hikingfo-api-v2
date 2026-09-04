// Package application holds the identity bounded context's use-cases. Pure Go:
// it depends only on identity/domain and the shared kernel, so it is unit-
// testable without a database or network (plan.md → DDD layers).
package application

import (
	"context"
	"time"

	"hikingfo/backend/internal/identity/domain"
)

// Dependencies the identity use-cases need. The composition root satisfies
// them with infrastructure adapters (argon2id, SMTP, OIDC, pgx repos).
type Dependencies struct {
	Users      domain.UserRepository
	Contacts   domain.ContactRepository
	Sessions   domain.SessionRepository
	Tokens     domain.TokenStore
	Passwords  domain.PasswordHasher
	Mailer     Mailer
	Now        func() time.Time
	SessionTTL time.Duration
	// AppURL is the external base URL used to build verify/reset links.
	AppURL string
}

// Mailer is the outbound email port (verification + reset links).
type Mailer interface {
	// SendVerification delivers the verify-email message to address.
	SendVerification(ctx context.Context, to, displayName, verifyURL string) error
	// SendPasswordReset delivers the reset link to address.
	SendPasswordReset(ctx context.Context, to, displayName, resetURL string) error
}

// Service exposes the identity use-cases to the interfaces layer.
type Service struct{ d Dependencies }

// New wires a Service.
func New(d Dependencies) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.SessionTTL <= 0 {
		d.SessionTTL = 7 * 24 * time.Hour
	}
	return &Service{d: d}
}

// Session is the result of a successful login / OIDC exchange.
type Session struct {
	Session *domain.Session
	// Token is the raw opaque token to place in the httpOnly cookie. It is not
	// persisted anywhere; only its hash is stored server-side.
	Token string
}
