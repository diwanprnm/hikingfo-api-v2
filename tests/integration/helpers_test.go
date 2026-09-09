package integration

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/achievement/application"
	achDomain "hikingfo/backend/internal/achievement/domain"
	achInfra "hikingfo/backend/internal/achievement/infrastructure"
	catApp "hikingfo/backend/internal/catalogue/application"
	catDomain "hikingfo/backend/internal/catalogue/domain"
	catInfra "hikingfo/backend/internal/catalogue/infrastructure"
	idApp "hikingfo/backend/internal/identity/application"
	identityDomain "hikingfo/backend/internal/identity/domain"
	identityInfra "hikingfo/backend/internal/identity/infrastructure"
	jrnApp "hikingfo/backend/internal/journey/application"
	jrnInfra "hikingfo/backend/internal/journey/infrastructure"
	prtApp "hikingfo/backend/internal/partner/application"
	prtDomain "hikingfo/backend/internal/partner/domain"
	prtInfra "hikingfo/backend/internal/partner/infrastructure"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// ---- test doubles ----------------------------------------------------------

// noopMailer swallows email sends.
type noopMailer struct{ sent []string }

func (m *noopMailer) SendVerification(ctx context.Context, email, name, link string) error {
	m.sent = append(m.sent, link)
	return nil
}
func (m *noopMailer) SendPasswordReset(ctx context.Context, email, name, link string) error {
	m.sent = append(m.sent, link)
	return nil
}

// fixedNow gives deterministic timestamps.
func fixedNow() time.Time { return time.Date(2026, 9, 4, 8, 0, 0, 0, time.UTC) }

// ---- seeded users ----------------------------------------------------------

type testUser struct {
	ID    ids.ID
	Token string // raw session token (cookie value)
}

// createUser inserts a verified, active user and opens a session; it returns
// the raw session token for direct repo-level session checks.
func createUser(t *testing.T, pool *pgxpool.Pool, email, displayName string) *testUser {
	t.Helper()
	ctx := context.Background()

	u := &identityDomain.User{
		ID:               ids.New(),
		Email:            email,
		PasswordHash:     "$argon2id$test",
		EmailVerifiedAt:  ptrTime(fixedNow()),
		DisplayName:      displayName,
		Role:             identityDomain.RoleMember,
		Status:           identityDomain.StatusActive,
		CreatedAt:        fixedNow(),
		UpdatedAt:        fixedNow(),
	}
	repo := identityInfra.NewUserRepository(pool)
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}

	// Open a session row with a known token.
	rawToken := fmt.Sprintf("test-token-%s", email)
	s := &identityDomain.Session{
		ID:        ids.New(),
		UserID:    u.ID,
		TokenHash: hashTokenForTest(rawToken),
		CSRFToken: "csrf-" + email,
		CreatedAt: fixedNow(),
		// Real-time expiry: session lookup compares expires_at > now() against
		// the container clock, so fixedNow()+24h would rot the day after.
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := identityInfra.NewSessionRepository(pool).Create(ctx, s); err != nil {
		t.Fatalf("create session for %s: %v", email, err)
	}
	return &testUser{ID: u.ID, Token: rawToken}
}

// ptrTime returns a pointer to t.
func ptrTime(t time.Time) *time.Time { return &t }

// hashTokenForTest mirrors identity/domain.IssueSessionToken's persisted hash
// (base64 raw-url SHA-256) without minting randomness.
func hashTokenForTest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// ---- services --------------------------------------------------------------

// services bundles the wired application services used by integration tests.
type services struct {
	Identity  *idApp.Service
	Journey   *jrnApp.Service
	Achieve   *application.Service
	Catalogue *catApp.Service
	Partner   *prtApp.Service
}

func newServices(pool *pgxpool.Pool) *services {
	identitySvc := idApp.New(idApp.Dependencies{
		Users:      identityInfra.NewUserRepository(pool),
		Contacts:   identityInfra.NewContactRepository(pool),
		Sessions:   identityInfra.NewSessionRepository(pool),
		Tokens:     identityInfra.NewEmailVerificationStore(pool),
		Passwords:  identityInfra.NewArgon2Hasher(),
		Mailer:     &noopMailer{},
		Now:        fixedNow,
		SessionTTL: 24 * time.Hour,
		AppURL:     "http://test.local",
	})

	jrnSvc := jrnApp.New(jrnApp.Dependencies{
		Hikes: jrnInfra.NewHikeLogRepository(pool),
		Posts: jrnInfra.NewJourneyPostRepository(pool),
	})

	achSvc := application.New(application.Dependencies{
		Badges:   achInfra.NewBadgeConfigRepo(pool),
		Journeys: achInfra.NewJourneyQuery(pool),
		Now:      fixedNow,
	})

	catSvc := catApp.New(catApp.Dependencies{
		Mountains:      catInfra.NewMountainRepository(pool),
		Weather:        catInfra.NewWeatherRepository(pool),
		WeatherBaseURL: "", // disabled upstream; stale-degrade path exercised directly
		WeatherTTL:     30 * time.Minute,
	})

	prtSvc := prtApp.New(prtApp.Dependencies{
		Notices:  prtInfra.NewNoticeRepository(pool),
		Requests: prtInfra.NewRequestRepository(pool),
		Profiles: &stubProfileReader{},
		Revealer: nil, // set per-test when contact reveal is exercised
		Now:      fixedNow,
	})

	return &services{Identity: identitySvc, Journey: jrnSvc, Achieve: achSvc, Catalogue: catSvc, Partner: prtSvc}
}

// stubProfileReader satisfies partner/domain.ProfileReader without identity.
type stubProfileReader struct{}

func (s *stubProfileReader) LimitedProfile(ctx context.Context, userID ids.ID) (*prtDomain.CandidateProfile, error) {
	return &prtDomain.CandidateProfile{UserID: userID, DisplayName: "Stub"}, nil
}

// stubRevealer returns canned contacts; stands in for identity reveal.
type stubRevealer struct{ called int }

func (r *stubRevealer) RevealContacts(ctx context.Context, ownerID ids.ID) (*prtDomain.RevealedContact, error) {
	r.called++
	return &prtDomain.RevealedContact{Email: "revealed@test.local", WhatsApp: "+62-812"}, nil
}

// ---- seeded mountains ------------------------------------------------------

// seedMountain inserts a published mountain with one route; returns its id.
func seedMountain(t *testing.T, pool *pgxpool.Pool, slug, nameID, nameEN string, region catDomain.Region, difficulty, height int) ids.ID {
	t.Helper()
	ctx := context.Background()

	m := &catDomain.Mountain{
		ID:          ids.New(),
		Slug:        slug,
		Name:        langtext.From(nameID, nameEN),
		Aliases:     []string{nameID},
		Region:      region,
		Province:    "Test Province",
		Location:    langtext.From("lokasi "+nameID, "location "+nameEN),
		Latitude:    -8.4,
		Longitude:   116.4,
		PeakName:    langtext.From("Puncak "+nameID, nameEN+" Peak"),
		PeakHeightM: height,
		Difficulty:  difficulty,
		Status:      catDomain.StatusPublished,
		DataMeta: map[string]catDomain.FieldMeta{
			"location": {Source: "test-suite", Reliability: "official", UpdatedAt: ptrTime(fixedNow())},
		},
		CreatedAt: fixedNow(),
		UpdatedAt: fixedNow(),
	}
	if err := insertMountainForTest(ctx, pool, m); err != nil {
		t.Fatalf("seed mountain %s: %v", slug, err)
	}
	return m.ID
}

// insertMountainForTest writes a mountain row directly (the catalogue repo is
// read-only in v1 — admin CRUD lands in Phase 8).
func insertMountainForTest(ctx context.Context, pool *pgxpool.Pool, m *catDomain.Mountain) error {
	nameJSON, _ := json.Marshal(m.Name)
	locJSON, _ := json.Marshal(m.Location)
	peakJSON, _ := json.Marshal(m.PeakName)
	metaJSON, _ := json.Marshal(m.DataMeta)
	searchText := strings.ToLower(m.Name.ID + " " + m.Name.EN + " " + strings.Join(m.Aliases, " "))
	_, err := pool.Exec(ctx, `
		INSERT INTO mountains (id, slug, name, aliases, region, province, location,
			latitude, longitude, peak_name, peak_height_m, difficulty, status,
			data_meta, search_text, created_at, updated_at)
		VALUES ($1::uuid, $2, $3::jsonb, $4, $5::mountain_region, $6, $7::jsonb,
			$8, $9, $10::jsonb, $11, $12, $13::mountain_status, $14::jsonb, $15, $16, $17)`,
		string(m.ID), m.Slug, nameJSON, m.Aliases, string(m.Region), m.Province, locJSON,
		m.Latitude, m.Longitude, peakJSON, m.PeakHeightM, m.Difficulty,
		string(m.Status), metaJSON, searchText, m.CreatedAt, m.UpdatedAt)
	return err
}

// compile-time interface guards.
var (
	_ idApp.Mailer                  = (*noopMailer)(nil)
	_ prtDomain.ProfileReader       = (*stubProfileReader)(nil)
	_ prtDomain.ContactRevealer     = (*stubRevealer)(nil)
	_ achDomain.BadgeConfigRepository = (*achInfra.BadgeConfigRepo)(nil)
)
