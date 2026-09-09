package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	catDomain "hikingfo/backend/internal/catalogue/domain"
	identityInfra "hikingfo/backend/internal/identity/infrastructure"
	jrnApp "hikingfo/backend/internal/journey/application"
	jrnInfra "hikingfo/backend/internal/journey/infrastructure"
	jrnHTTP "hikingfo/backend/internal/journey/interfaces"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/ids"
)

// journeyFixture wires the journey HTTP stack (session auth) over one pool.
type journeyFixture struct {
	router *gin.Engine
	pool   *pgxpool.Pool
}

func newJourneyFixture(t *testing.T) *journeyFixture {
	t.Helper()
	pool := newPostgres(t)
	gin.SetMode(gin.TestMode)

	jrnSvc := jrnApp.New(jrnApp.Dependencies{
		Hikes: jrnInfra.NewHikeLogRepository(pool),
		Posts: jrnInfra.NewJourneyPostRepository(pool),
	})
	h := jrnHTTP.NewHandler(jrnSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if tok := c.GetHeader("X-Test-Token"); tok != "" {
			c.Request.AddCookie(&http.Cookie{Name: "hikingfo_session", Value: tok})
		}
		c.Next()
	})
	r.Use(plathttp.SessionAuth("hikingfo_session", identityInfra.NewSessionLookupAdapter(pool)))
	jrnHTTP.RegisterRoutes(r.Group("/api/v1"), h)

	return &journeyFixture{router: r, pool: pool}
}

func (f *journeyFixture) doAs(t *testing.T, token, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Test-Token", token)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

// T015 (004 US2): feed + report endpoints are signed-in-only, published posts
// are immutable via the service guard (covered in stories), and the HTTP route
// table itself 401s without a session.
func TestJourneyFeedRequiresAuth(t *testing.T) {
	f := newJourneyFixture(t)
	ctx := context.Background()

	// Seed one published post to make the feed non-trivial.
	author := createUser(t, f.pool, "jauthor@test.local", "Author")
	mtn := seedMountain(t, f.pool, "gunung-jauth", "Gunung JAuth", "JAuth Mount", catDomain.RegionJawa, 2, 2000)
	jrnSvc := jrnApp.New(jrnApp.Dependencies{
		Hikes: jrnInfra.NewHikeLogRepository(f.pool),
		Posts: jrnInfra.NewJourneyPostRepository(f.pool),
	})
	res, err := jrnSvc.RecordHike(ctx, author.ID, jrnApp.RecordHikeInput{
		MountainID: mtn, ClimbDate: "2026-08-01",
		EvidencePhotoKeys: []string{"evidence/x.jpg"},
		Title:             strPtr("Laporan"), Visibility: strPtr("published"),
	})
	if err != nil {
		t.Fatalf("seed hike+post: %v", err)
	}

	// Signed out → every feed route 401s.
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/journeys"},
		{http.MethodGet, "/api/v1/journeys/00000000-0000-0000-0000-000000000000"},
		{http.MethodPost, "/api/v1/journeys/" + string(res.ID) + "/reports"},
	} {
		rec := f.doAs(t, "", tc.method, tc.path, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s signed-out: got %d, want 401 (%s)", tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}

	// Signed in → feed readable.
	rec := f.doAs(t, author.Token, http.MethodGet, "/api/v1/journeys", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("signed-in feed: got %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)

	// Non-author cannot delete the post; author can.
	postID := *postIDFromHike(t, jrnSvc, ctx, author.ID)
	other := createUser(t, f.pool, "jother@test.local", "Other")
	if err := jrnSvc.DeletePost(ctx, postID, other.ID); err == nil {
		t.Fatal("non-author delete must be rejected")
	}
	if err := jrnSvc.DeletePost(ctx, postID, author.ID); err != nil {
		t.Fatalf("author delete: %v", err)
	}
}

// postIDFromHike returns the linked journey post id of the author's first hike.
func postIDFromHike(t *testing.T, svc *jrnApp.Service, ctx context.Context, userID ids.ID) *ids.ID {
	t.Helper()
	hikes, err := svc.ListMyHikes(ctx, userID, 1, 10)
	if err != nil {
		t.Fatalf("ListMyHikes: %v", err)
	}
	for _, h := range hikes.Items {
		entry, err := svc.GetMyHike(ctx, h.ID, userID)
		if err != nil {
			t.Fatalf("GetMyHike(%v): %v", h.ID, err)
		}
		if entry.PublishedPostID != nil {
			return entry.PublishedPostID
		}
	}
	t.Fatal("no linked post found")
	return nil
}
