package application

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// RegisterParams is the email/password signup input.
type RegisterParams struct {
	Email       string
	Password    string
	DisplayName string
}

// Register creates an account, sends the verification email and does NOT log
// the user in (email must be verified first — spec Decisions → Auth).
func (s *Service) Register(ctx context.Context, p RegisterParams) (*domain.User, error) {
	if err := domain.ValidateEmail(p.Email); err != nil {
		return nil, err
	}
	if err := domain.ValidatePassword(p.Password); err != nil {
		return nil, err
	}
	if err := domain.ValidateDisplayName(p.DisplayName); err != nil {
		return nil, err
	}

	hash, err := s.d.Passwords.Hash(p.Password)
	if err != nil {
		return nil, kerr.WrapInternal("could not secure password", err)
	}

	now := s.d.Now()
	u := &domain.User{
		ID:           ids.New(),
		Email:        strings.ToLower(strings.TrimSpace(p.Email)),
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(p.DisplayName),
		Role:         domain.RoleMember,
		Status:       domain.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.d.Users.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			return nil, kerr.Conflict("an account with this email already exists").
				WithField("email", "already_registered")
		}
		return nil, kerr.WrapInternal("could not create account", err)
	}

	// A verification-email send failure is logged by the mailer adapter; it must
	// not silently strand the account, but also must not roll back registration.
	// The user can resend from the login screen. Registration succeeds regardless.
	if err := s.issueVerificationEmail(ctx, u); err != nil {
		// surface as non-fatal: account is created, verification pending.
		return u, nil
	}
	return u, nil
}

// issueVerificationEmail mints a single-use token and asks the mailer to send
// the verify link.
func (s *Service) issueVerificationEmail(ctx context.Context, u *domain.User) error {
	token, hash, err := domain.IssueSessionToken()
	if err != nil {
		return err
	}
	if err := s.d.Tokens.SaveEmailVerification(ctx, u.ID, hash, s.d.Now().Add(24*time.Hour)); err != nil {
		return err
	}
	link := joinURL(s.d.AppURL, "/auth/verify-email?token="+url.QueryEscape(token))
	return s.d.Mailer.SendVerification(ctx, u.Email, u.DisplayName, link)
}

func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + path
}

// LoginParams is the email/password login input.
type LoginParams struct {
	Email    string
	Password string
}

// Login authenticates and opens a session.
func (s *Service) Login(ctx context.Context, p LoginParams) (*Session, error) {
	email := strings.ToLower(strings.TrimSpace(p.Email))
	if email == "" || p.Password == "" {
		return nil, kerr.Validation("email and password are required").
			WithField("email", "required").WithField("password", "required")
	}

	u, err := s.d.Users.FindByEmail(ctx, email)
	if err != nil {
		return nil, kerr.Unauthorized("email or password is incorrect")
	}
	if u.PasswordHash == "" {
		return nil, kerr.Unauthorized("this account uses Google sign-in")
	}
	ok, err := s.d.Passwords.Verify(u.PasswordHash, p.Password)
	if err != nil || !ok {
		return nil, kerr.Unauthorized("email or password is incorrect")
	}
	if !u.Status.IsActive() {
		return nil, kerr.Forbidden("account is suspended")
	}

	return s.openSession(ctx, u)
}

// openSession persists a session row and returns the raw token for the cookie.
func (s *Service) openSession(ctx context.Context, u *domain.User) (*Session, error) {
	now := s.d.Now()
	token, hash, err := domain.IssueSessionToken()
	if err != nil {
		return nil, kerr.WrapInternal("could not start session", err)
	}
	csrf, err := domain.IssueCSRFToken()
	if err != nil {
		return nil, kerr.WrapInternal("could not start session", err)
	}
	sess := &domain.Session{
		ID:        ids.New(),
		UserID:    u.ID,
		TokenHash: hash,
		CSRFToken: csrf,
		CreatedAt: now,
		ExpiresAt: now.Add(s.d.SessionTTL),
	}
	if err := s.d.Sessions.Create(ctx, sess); err != nil {
		return nil, kerr.WrapInternal("could not persist session", err)
	}
	return &Session{Session: sess, Token: token}, nil
}

// Logout revokes the session token.
func (s *Service) Logout(ctx context.Context, tokenHash string) error {
	if tokenHash == "" {
		return nil
	}
	_ = s.d.Sessions.RevokeByHash(ctx, tokenHash)
	return nil
}

// VerifyEmail consumes a verification token.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return kerr.Validation("token is required").WithField("token", "required")
	}
	userID, err := s.d.Tokens.ConsumeEmailVerification(ctx, domain.HashToken(token))
	if err != nil {
		return kerr.Validation("token is invalid or expired").WithField("token", "invalid")
	}
	u, err := s.d.Users.FindByID(ctx, userID)
	if err != nil {
		return kerr.WrapNotFound("account not found", err)
	}
	now := s.d.Now()
	u.EmailVerifiedAt = &now
	if err := s.d.Users.Update(ctx, u); err != nil {
		return kerr.WrapInternal("could not verify email", err)
	}
	return nil
}

// ForgotPassword emails a reset link when the account exists. It always
// returns success from the caller's perspective (never reveals whether an
// email is registered — enumeration protection).
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return kerr.Validation("email is required").WithField("email", "required")
	}
	u, err := s.d.Users.FindByEmail(ctx, email)
	if err != nil {
		return nil // deliberate: no account-enumeration oracle
	}
	token, hash, err := domain.IssueSessionToken()
	if err != nil {
		return kerr.WrapInternal("could not create reset token", err)
	}
	if err := s.d.Tokens.SavePasswordReset(ctx, u.ID, hash, s.d.Now().Add(1*time.Hour)); err != nil {
		return kerr.WrapInternal("could not save reset token", err)
	}
	link := joinURL(s.d.AppURL, "/auth/reset-password?token="+url.QueryEscape(token))
	if err := s.d.Mailer.SendPasswordReset(ctx, u.Email, u.DisplayName, link); err != nil {
		return kerr.WrapInternal("could not send reset email", err)
	}
	return nil
}

// ResetPassword consumes a reset token and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := domain.ValidatePassword(newPassword); err != nil {
		return err
	}
	if token == "" {
		return kerr.Validation("token is required").WithField("token", "required")
	}
	userID, err := s.d.Tokens.ConsumePasswordReset(ctx, domain.HashToken(token))
	if err != nil {
		return kerr.Validation("token is invalid or expired").WithField("token", "invalid")
	}
	u, err := s.d.Users.FindByID(ctx, userID)
	if err != nil {
		return kerr.WrapNotFound("account not found", err)
	}
	hash, err := s.d.Passwords.Hash(newPassword)
	if err != nil {
		return kerr.WrapInternal("could not secure password", err)
	}
	u.PasswordHash = hash
	if err := s.d.Users.Update(ctx, u); err != nil {
		return kerr.WrapInternal("could not reset password", err)
	}
	// Force logout on other devices.
	_ = s.d.Sessions.RevokeAllForUser(ctx, u.ID)
	return nil
}
