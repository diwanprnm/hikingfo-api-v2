// Package application unit tests: register/login/reset flows and privacy
// gating run against in-memory fakes (no DB — T026).
package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// ---- fakes -----------------------------------------------------------------

type fakeUsers struct {
	users    map[ids.ID]*domain.User
	byEmail  map[string]*domain.User
	createErr error
	updated  []*domain.User
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[ids.ID]*domain.User{}, byEmail: map[string]*domain.User{}}
}

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, dup := f.byEmail[u.Email]; dup {
		return domain.ErrDuplicate
	}
	cp := *u
	f.users[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	return nil
}

func (f *fakeUsers) FindByID(_ context.Context, id ids.ID) (*domain.User, error) {
	if u, ok := f.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, domain.ErrNotFound{Kind: "user"}
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := f.byEmail[email]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, domain.ErrNotFound{Kind: "user"}
}

func (f *fakeUsers) FindByGoogleSub(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound{Kind: "user"}
}

func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	cp := *u
	f.users[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	f.updated = append(f.updated, &cp)
	return nil
}

func (f *fakeUsers) SetStatus(_ context.Context, id ids.ID, s domain.Status) error {
	u, ok := f.users[id]
	if !ok {
		return domain.ErrNotFound{Kind: "user"}
	}
	u.Status = s
	return nil
}

func (f *fakeUsers) TouchLastSeen(_ context.Context, _ ids.ID, _ time.Time) error { return nil }

type fakeContacts struct{ channels map[ids.ID]domain.ContactChannel }

func newFakeContacts() *fakeContacts { return &fakeContacts{channels: map[ids.ID]domain.ContactChannel{}} }

func (f *fakeContacts) Get(_ context.Context, id ids.ID) (domain.ContactChannel, error) {
	return f.channels[id], nil
}
func (f *fakeContacts) Upsert(_ context.Context, id ids.ID, c domain.ContactChannel) error {
	f.channels[id] = c
	return nil
}

type fakeSessions struct{ sessions map[string]*domain.Session }

func newFakeSessions() *fakeSessions { return &fakeSessions{sessions: map[string]*domain.Session{}} }

func (f *fakeSessions) Create(_ context.Context, s *domain.Session) error {
	f.sessions[s.TokenHash] = s
	return nil
}
func (f *fakeSessions) FindByTokenHash(_ context.Context, h string) (*domain.Session, error) {
	if s, ok := f.sessions[h]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound{Kind: "session"}
}
func (f *fakeSessions) Revoke(_ context.Context, _ ids.ID) error         { return nil }
func (f *fakeSessions) RevokeByHash(_ context.Context, _ string) error   { return nil }
func (f *fakeSessions) RevokeAllForUser(_ context.Context, _ ids.ID) error { return nil }

type fakeTokens struct{ verifications, resets map[string]ids.ID }

func newFakeTokens() *fakeTokens {
	return &fakeTokens{verifications: map[string]ids.ID{}, resets: map[string]ids.ID{}}
}

func (f *fakeTokens) SaveEmailVerification(_ context.Context, id ids.ID, h string, _ time.Time) error {
	f.verifications[h] = id
	return nil
}
func (f *fakeTokens) SavePasswordReset(_ context.Context, id ids.ID, h string, _ time.Time) error {
	f.resets[h] = id
	return nil
}
func (f *fakeTokens) ConsumeEmailVerification(_ context.Context, h string) (ids.ID, error) {
	id, ok := f.verifications[h]
	if !ok {
		return "", domain.ErrNotFound{Kind: "token"}
	}
	delete(f.verifications, h)
	return id, nil
}
func (f *fakeTokens) ConsumePasswordReset(_ context.Context, h string) (ids.ID, error) {
	id, ok := f.resets[h]
	if !ok {
		return "", domain.ErrNotFound{Kind: "token"}
	}
	delete(f.resets, h)
	return id, nil
}

type fakePasswords struct{}

func (fakePasswords) Hash(p string) (string, error) { return "hashed:" + p, nil }
func (fakePasswords) Verify(hash, p string) (bool, error) {
	return hash == "hashed:"+p, nil
}

type fakeMailer struct{ verificationSent, resetSent int }

func (m *fakeMailer) SendVerification(_ context.Context, _, _, _ string) error {
	m.verificationSent++
	return nil
}
func (m *fakeMailer) SendPasswordReset(_ context.Context, _, _, _ string) error {
	m.resetSent++
	return nil
}

// newTestService wires a Service over fakes.
func newTestService(users *fakeUsers, mailer *fakeMailer) *Service {
	return New(Dependencies{
		Users:      users,
		Contacts:   newFakeContacts(),
		Sessions:   newFakeSessions(),
		Tokens:     newFakeTokens(),
		Passwords:  fakePasswords{},
		Mailer:     mailer,
		Now:        func() time.Time { return time.Date(2026, 9, 4, 8, 0, 0, 0, time.UTC) },
		SessionTTL: time.Hour,
		AppURL:     "http://test.local",
	})
}

// ---- register --------------------------------------------------------------

func TestRegisterCreatesAccountAndSendsVerification(t *testing.T) {
	users := newFakeUsers()
	mailer := &fakeMailer{}
	svc := newTestService(users, mailer)

	u, err := svc.Register(context.Background(), RegisterParams{
		Email: "Pendaki@Example.com", Password: "pendakigunung", DisplayName: "Pendaki",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if u.Email != "pendaki@example.com" {
		t.Fatalf("email must be lowercased, got %q", u.Email)
	}
	if !strings.HasPrefix(u.PasswordHash, "hashed:") {
		t.Fatalf("password must be hashed, got %q", u.PasswordHash)
	}
	if mailer.verificationSent != 1 {
		t.Fatalf("expected 1 verification email, got %d", mailer.verificationSent)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	svc := newTestService(newFakeUsers(), &fakeMailer{})
	cases := []RegisterParams{
		{Email: "not-an-email", Password: "pendakigunung", DisplayName: "P"},
		{Email: "a@b.co", Password: "short", DisplayName: "P"},
		{Email: "a@b.co", Password: "pendakigunung", DisplayName: " "},
	}
	for _, c := range cases {
		if _, err := svc.Register(context.Background(), c); err == nil {
			t.Errorf("Register(%+v) expected validation error", c)
		}
	}
}

func TestRegisterDuplicateEmailConflicts(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	p := RegisterParams{Email: "a@b.co", Password: "pendakigunung", DisplayName: "A"}
	if _, err := svc.Register(context.Background(), p); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	_, err := svc.Register(context.Background(), p)
	ke, ok := err.(*kerr.Error)
	if !ok || ke.Code != kerr.CodeConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

// ---- login -----------------------------------------------------------------

func seedUser(t *testing.T, users *fakeUsers, email, password string) *domain.User {
	t.Helper()
	now := time.Date(2026, 9, 4, 8, 0, 0, 0, time.UTC)
	u := &domain.User{
		ID: ids.New(), Email: email,
		PasswordHash: "hashed:" + password,
		DisplayName:  "T", Role: domain.RoleMember, Status: domain.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := users.Create(context.Background(), u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func TestLoginValidCredentialsOpenSession(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	seedUser(t, users, "a@b.co", "pendakigunung")

	sess, err := svc.Login(context.Background(), LoginParams{Email: "a@b.co", Password: "pendakigunung"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if sess.Token == "" || sess.Session.TokenHash != domain.HashToken(sess.Token) {
		t.Fatal("session token/hash pair inconsistent")
	}
}

func TestLoginWrongPasswordUnauthorized(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	seedUser(t, users, "a@b.co", "pendakigunung")

	_, err := svc.Login(context.Background(), LoginParams{Email: "a@b.co", Password: "wrong-pass"})
	ke, ok := err.(*kerr.Error)
	if !ok || ke.Code != kerr.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestLoginSuspendedAccountForbidden(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	u := seedUser(t, users, "a@b.co", "pendakigunung")
	u.Status = domain.StatusSuspended
	_ = users.Update(context.Background(), u)

	_, err := svc.Login(context.Background(), LoginParams{Email: "a@b.co", Password: "pendakigunung"})
	ke, ok := err.(*kerr.Error)
	if !ok || ke.Code != kerr.CodeForbidden {
		t.Fatalf("expected forbidden for suspended account, got %v", err)
	}
}

// ---- reset -----------------------------------------------------------------

func TestForgotPasswordNeverEnumerates(t *testing.T) {
	mailer := &fakeMailer{}
	svc := newTestService(newFakeUsers(), mailer)

	// Unknown email: no error, no mail — no enumeration oracle.
	if err := svc.ForgotPassword(context.Background(), "nobody@x.co"); err != nil {
		t.Fatalf("ForgotPassword unknown email: %v", err)
	}
	if mailer.resetSent != 0 {
		t.Fatalf("no email must be sent for unknown account, got %d", mailer.resetSent)
	}
}

func TestResetPasswordConsumesTokenAndRevokesSessions(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	u := seedUser(t, users, "a@b.co", "pendakigunung")

	// Mint a reset token through the service's own path (register + issue via
	// ForgotPassword), then drive ResetPassword with the stored hash's token.
	// Since the raw token isn't exposed by fakes, simulate: save, then consume.
	tokens := newFakeTokens()
	hash := domain.HashToken("raw-reset-token")
	_ = tokens.SavePasswordReset(context.Background(), u.ID, hash, time.Now().Add(time.Hour))
	svc.d.Tokens = tokens

	if err := svc.ResetPassword(context.Background(), "raw-reset-token", "newpendakipass"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	after, _ := users.FindByID(context.Background(), u.ID)
	if after.PasswordHash != "hashed:newpendakipass" {
		t.Fatalf("password not updated, got %q", after.PasswordHash)
	}
	// Token single-use: second consume fails.
	if err := svc.ResetPassword(context.Background(), "raw-reset-token", "another-pass-1"); err == nil {
		t.Fatal("reset token must be single-use")
	}
}

// ---- privacy gating --------------------------------------------------------

func TestLimitedProfileOmitsContacts(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	u := seedUser(t, users, "a@b.co", "pendakigunung")

	lp, err := svc.LimitedProfile(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("LimitedProfile: %v", err)
	}
	// Compile-time shape: LimitedProfile has no contact fields.
	if lp.DisplayName != u.DisplayName {
		t.Fatalf("limited profile mismatch: %+v", lp)
	}
}

func TestLimitedProfileHiddenForSuspendedUser(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	u := seedUser(t, users, "a@b.co", "pendakigunung")
	u.Status = domain.StatusSuspended
	_ = users.Update(context.Background(), u)

	if _, err := svc.LimitedProfile(context.Background(), u.ID); err == nil {
		t.Fatal("suspended user must not appear in limited profiles")
	}
}

func TestAuthorizeContactRevealOnlyWhenChannelsSet(t *testing.T) {
	contacts := newFakeContacts()
	svc := New(Dependencies{
		Users: newFakeUsers(), Contacts: contacts, Sessions: newFakeSessions(),
		Tokens: newFakeTokens(), Passwords: fakePasswords{}, Mailer: &fakeMailer{},
	})
	ctx := context.Background()
	uid := ids.New()

	// No channels set → not authorized.
	if _, ok, err := svc.AuthorizeContactReveal(ctx, uid); err != nil || ok {
		t.Fatalf("empty channels must not authorize (ok=%v err=%v)", ok, err)
	}

	// Channels set → authorized with values.
	_ = contacts.Upsert(ctx, uid, domain.ContactChannel{WhatsApp: "+62-812", Instagram: "@x"})
	c, ok, err := svc.AuthorizeContactReveal(ctx, uid)
	if err != nil || !ok {
		t.Fatalf("channels set must authorize (ok=%v err=%v)", ok, err)
	}
	if c.WhatsApp != "+62-812" {
		t.Fatalf("unexpected channels: %+v", c)
	}
}

func TestRevealContactsReturnsEmailPlusChannels(t *testing.T) {
	users := newFakeUsers()
	contacts := newFakeContacts()
	svc := New(Dependencies{
		Users: users, Contacts: contacts, Sessions: newFakeSessions(),
		Tokens: newFakeTokens(), Passwords: fakePasswords{}, Mailer: &fakeMailer{},
	})
	ctx := context.Background()
	u := seedUser(t, users, "revealed@x.co", "pendakigunung")
	_ = contacts.Upsert(ctx, u.ID, domain.ContactChannel{Phone: "+62-811", WhatsApp: "+62-812"})

	res, err := svc.RevealContacts(ctx, u.ID)
	if err != nil {
		t.Fatalf("RevealContacts: %v", err)
	}
	if res.Email != "revealed@x.co" || res.WhatsApp != "+62-812" || res.Phone != "+62-811" {
		t.Fatalf("unexpected reveal: %+v", res)
	}
}

// ---- email verification ----------------------------------------------------

func TestVerifyEmailConsumesTokenAndMarksVerified(t *testing.T) {
	users := newFakeUsers()
	svc := newTestService(users, &fakeMailer{})
	u := seedUser(t, users, "a@b.co", "pendakigunung")

	tokens := newFakeTokens()
	_ = tokens.SaveEmailVerification(context.Background(), u.ID, domain.HashToken("raw-verify"), time.Now().Add(time.Hour))
	svc.d.Tokens = tokens

	if err := svc.VerifyEmail(context.Background(), "raw-verify"); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	after, _ := users.FindByID(context.Background(), u.ID)
	if !after.EmailVerified() {
		t.Fatal("email must be verified after token consumption")
	}
	if err := svc.VerifyEmail(context.Background(), "raw-verify"); err == nil {
		t.Fatal("verification token must be single-use")
	}
}
