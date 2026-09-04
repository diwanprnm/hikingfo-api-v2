package application

import (
	"context"
	"errors"
	"strings"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// GoogleIdentity is the verified claim set from a Google ID token.
type GoogleIdentity struct {
	Sub           string // OIDC subject, stable per Google account
	Email         string
	EmailVerified bool
	DisplayName   string
	PictureURL    string
}

// GoogleVerifier validates a Google ID token / auth code and returns the
// verified identity. The infrastructure package implements it with go-oidc
// (research.md §2); the application depends only on this port.
type GoogleVerifier interface {
	Verify(ctx context.Context, idTokenOrCode string) (*GoogleIdentity, error)
}

// GoogleParams is the exchange input from POST /auth/google.
type GoogleParams struct {
	Credential string // the ID token (or authorization code) from Google
}

// GoogleLogin verifies the Google credential and returns a session, creating
// the account on first sign-in (email verified by Google).
func (s *Service) GoogleLogin(ctx context.Context, p GoogleParams, verifier GoogleVerifier) (*Session, *domain.User, error) {
	if p.Credential == "" {
		return nil, nil, kerr.Validation("credential is required").WithField("credential", "required")
	}
	gi, err := verifier.Verify(ctx, p.Credential)
	if err != nil {
		return nil, nil, kerr.Unauthorized("Google sign-in could not be verified")
	}
	if !gi.EmailVerified || gi.Email == "" {
		return nil, nil, kerr.Unauthorized("Google account email is not verified")
	}

	email := strings.ToLower(strings.TrimSpace(gi.Email))

	u, err := s.d.Users.FindByGoogleSub(ctx, gi.Sub)
	if err == nil {
		if !u.Status.IsActive() {
			return nil, nil, kerr.Forbidden("account is suspended")
		}
		// Merge any changed profile details from Google.
		if gi.DisplayName != "" && u.DisplayName == "" {
			u.DisplayName = gi.DisplayName
		}
		if u.GoogleSub == "" {
			u.GoogleSub = gi.Sub // link email account to Google on subsequent sign-in
		}
		_ = s.d.Users.Update(ctx, u)
		sess, err := s.openSession(ctx, u)
		return sess, u, err
	}

	// No account for this Google subject yet: find by email to link, else create.
	existing, findErr := s.d.Users.FindByEmail(ctx, email)
	if findErr == nil && existing.GoogleSub == "" {
		// Link the Google subject to the existing email/password account.
		existing.GoogleSub = gi.Sub
		if existing.DisplayName == "" {
			existing.DisplayName = gi.DisplayName
		}
		if err := s.d.Users.Update(ctx, existing); err != nil {
			return nil, nil, kerr.WrapInternal("could not link Google account", err)
		}
		sess, err := s.openSession(ctx, existing)
		return sess, existing, err
	}
	if findErr == nil {
		// An account with this email exists but is already linked to another
		// Google subject — treat as a conflict to avoid account confusion.
		return nil, nil, kerr.Conflict("this email is already linked to a different Google account")
	}

	now := s.d.Now()
	nu := &domain.User{
		ID:              ids.New(),
		Email:           email,
		GoogleSub:       gi.Sub,
		EmailVerifiedAt: &now, // Google already verified it
		DisplayName:     gi.DisplayName,
		AvatarKey:       "", // avatar fetched lazily from picture URL by infra
		Role:            domain.RoleMember,
		Status:          domain.StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if nu.DisplayName == "" {
		nu.DisplayName = strings.Split(email, "@")[0]
	}
	if err := s.d.Users.Create(ctx, nu); err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return nil, nil, kerr.Conflict("an account with this email already exists")
		}
		return nil, nil, kerr.WrapInternal("could not create account", err)
	}
	sess, err := s.openSession(ctx, nu)
	return sess, nu, err
}
