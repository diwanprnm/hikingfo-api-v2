package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	achApp "hikingfo/backend/internal/achievement/application"
	achInfra "hikingfo/backend/internal/achievement/infrastructure"
	achHTTP "hikingfo/backend/internal/achievement/interfaces"
	catApp "hikingfo/backend/internal/catalogue/application"
	catDomain "hikingfo/backend/internal/catalogue/domain"
	catInfra "hikingfo/backend/internal/catalogue/infrastructure"
	catHTTP "hikingfo/backend/internal/catalogue/interfaces"
	idApp "hikingfo/backend/internal/identity/application"
	identityInfra "hikingfo/backend/internal/identity/infrastructure"
	identityHTTP "hikingfo/backend/internal/identity/interfaces"
	jrnInfra "hikingfo/backend/internal/journey/infrastructure"
	modApp "hikingfo/backend/internal/moderation/application"
	modInfra "hikingfo/backend/internal/moderation/infrastructure"
	modHTTP "hikingfo/backend/internal/moderation/interfaces"
	notifApp "hikingfo/backend/internal/notification/application"
	notifInfra "hikingfo/backend/internal/notification/infrastructure"
	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/ids"
)

// adminFixture wires the full HTTP stack (session auth + admin guards) over
// one pool, mirroring cmd/api/main.go minus config/mailer.
type adminFixture struct {
	pool   *pgxpool.Pool
	router *gin.Engine
	admin  *testUser // session token with role=admin
	modSvc *modApp.Service
	blobs  *fakeBlobStore // 002: in-memory MinIO stand-in for photo/upload tests
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	pool := newPostgres(t)
	gin.SetMode(gin.TestMode)
	f := &adminFixture{pool: pool, blobs: newFakeBlobStore()}

	notifSvc := notifApp.New(notifInfra.NewNotificationRepository(pool))
	jrnPosts := jrnInfra.NewJourneyPostRepository(pool)
	jrnHikes := jrnInfra.NewHikeLogRepository(pool)
	modOwners := modInfra.NewOwners(pool)
	f.modSvc = modApp.New(modInfra.NewReportRepository(pool), modApp.SideEffects{
		HidePost: func(ctx context.Context, postID ids.ID) error {
			return jrnPosts.SetModerationStatus(ctx, postID, "hidden")
		},
		RemoveEvidence: func(ctx context.Context, hikeID ids.ID) error {
			return jrnHikes.SetStatus(ctx, hikeID, "removed")
		},
		Notify: func(ctx context.Context, userID ids.ID, payload map[string]any) error {
			return notifSvc.Record(ctx, userID, "moderation_outcome", payload)
		},
	})
	f.modSvc.JourneyOwner = modOwners.JourneyOwner
	f.modSvc.HikeOwner = modOwners.HikeOwner

	catSvc := catApp.New(catApp.Dependencies{
		Mountains:   catInfra.NewMountainRepository(pool),
		AdminWriter: catInfra.NewMountainWriter(pool),
		Weather:     catInfra.NewWeatherRepository(pool),
		Gallery:     catInfra.NewGalleryRepository(pool),
		Blobs:       f.blobs,
	})
	achSvc := achApp.New(achApp.Dependencies{
		Badges:   achInfra.NewBadgeConfigRepo(pool),
		Journeys: achInfra.NewJourneyQuery(pool),
		Now:      fixedNow,
	})
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

	lookup := identityInfra.NewSessionLookupAdapter(pool)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if tok := c.GetHeader("X-Test-Token"); tok != "" {
			c.Request.AddCookie(&http.Cookie{Name: "hikingfo_session", Value: tok})
		}
		c.Next()
	})
	r.Use(plathttp.SessionAuth("hikingfo_session", lookup))

	api := r.Group("/api/v1")
	catHandler := catHTTP.NewHandler(catSvc)
	catHTTP.RegisterRoutes(api, catHandler)
	catHTTP.RegisterAdminRoutes(api, catHandler)
	modHandler := modHTTP.NewHandler(f.modSvc)
	modHTTP.RegisterRoutes(api, modHandler)
	modHTTP.RegisterAdminRoutes(api, modHandler)
	achHandler := achHTTP.NewHandler(achSvc)
	achHTTP.RegisterAdminRoutes(api, achHandler)
	identityHandler := identityHTTP.NewHandler(identitySvc, identityHTTP.Config{CookieName: "hikingfo_session"})
	identityHTTP.RegisterAdminRoutes(api, identityHandler)
	identityHandler.SetStats(func(ctx context.Context) (map[string]any, error) {
		mountains, err := catInfra.NewMountainWriter(pool).ListAll(ctx)
		if err != nil {
			return nil, err
		}
		complete := 0
		for _, m := range mountains {
			if m.Location.ID != "" && len(m.DataMeta) > 0 {
				complete++
			}
		}
		reportCounts, err := f.modSvc.Counts(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"mountains": map[string]any{"total": len(mountains), "complete": complete},
			"reports":   reportCounts,
		}, nil
	})

	f.router = r

	// Admin user + session.
	u := createUser(t, pool, "admin-smoke@test.local", "Admin Smoke")
	if _, err := pool.Exec(context.Background(),
		`UPDATE users SET role = 'admin' WHERE id = $1`, string(u.ID)); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	f.admin = u
	return f
}

// do performs a request as the admin session.
func (f *adminFixture) do(t *testing.T, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	return f.doAs(t, f.admin.Token, method, path, body)
}

func (f *adminFixture) doAs(t *testing.T, token, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Token", token)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

// ---- T097 admin smoke ------------------------------------------------------

func TestAdminSmoke(t *testing.T) {
	f := newAdminFixture(t)
	ctx := context.Background()

	t.Run("member_forbidden", func(t *testing.T) {
		member := createUser(t, f.pool, "member-smoke@test.local", "Member")
		rec, _ := f.doAs(t, member.Token, "GET", "/api/v1/admin/mountains", nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("member got %d, want 403", rec.Code)
		}
	})

	t.Run("mountain_crud_with_revision_and_rollback", func(t *testing.T) {
		// Create.
		rec, out := f.do(t, "POST", "/api/v1/admin/mountains", map[string]any{
			"slug": "smoke-mountain", "name": map[string]string{"id": "Gunung Smoke", "en": "Smoke Mountain"},
			"region": "Jawa", "location": map[string]string{"id": "Jawa Timur", "en": "East Java"},
			"peak_name": map[string]string{"id": "Puncak Smoke"}, "peak_height_m": 3000,
			"difficulty": 3, "status": "published",
			"data_meta": map[string]any{"location": map[string]any{"source": "smoke", "reliability": "official"}},
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
		}
		// Mountain.ID serializes as "id" (json tag).
		idRaw, ok := out["id"].(string)
		if !ok || idRaw == "" {
			t.Fatalf("create returned no id: %s", rec.Body.String())
		}
		id := ids.ID(idRaw)

		// Duplicate slug → 409.
		rec, _ = f.do(t, "POST", "/api/v1/admin/mountains", map[string]any{
			"slug": "smoke-mountain", "name": map[string]string{"id": "X"},
			"region": "Jawa", "location": map[string]string{"id": "Jawa Timur"},
			"peak_name": map[string]string{"id": "X"}, "difficulty": 1,
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("duplicate slug: %d, want 409", rec.Code)
		}

		// Update → revision snapshot.
		rec, _ = f.do(t, "PATCH", "/api/v1/admin/mountains/"+string(id), map[string]any{
			"peak_height_m": 3200, "reason": "smoke edit",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
		}

		// Revisions list.
		rec, revs := f.do(t, "GET", "/api/v1/admin/mountains/"+string(id)+"/revisions", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("revisions: %d", rec.Code)
		}
		items := revs["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("want 1 revision, got %d", len(items))
		}
		revRaw, ok := items[0].(map[string]any)["ID"].(string)
		if !ok || revRaw == "" {
			t.Fatalf("revision has no ID: %s", rec.Body.String())
		}
		revID := ids.ID(revRaw)

		// Rollback → height back to 3000.
		rec, _ = f.do(t, "POST", "/api/v1/admin/mountains/"+string(id)+"/revisions/"+string(revID)+"/rollback", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("rollback: %d %s", rec.Code, rec.Body.String())
		}
		m, err := f.admin2Mountain(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if m.PeakHeightM != 3000 {
			t.Fatalf("after rollback height = %d, want 3000", m.PeakHeightM)
		}

		// Delete.
		rec, _ = f.do(t, "DELETE", "/api/v1/admin/mountains/"+string(id), nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("delete: %d", rec.Code)
		}
	})

	t.Run("report_resolve_hides_post_and_notifies", func(t *testing.T) {
		// Seed a journey post via the journey repo.
		author := createUser(t, f.pool, "author-smoke@test.local", "Author")
		postID := ids.New()
		_, err := f.pool.Exec(ctx, `
			INSERT INTO journey_posts (id, user_id, mountain_id, title, narrative, photo_keys, visibility, moderation_status)
			VALUES ($1, $2, (SELECT id FROM mountains LIMIT 1), $3, '{"id":"cerita"}'::jsonb, '{}', 'published', 'visible')`,
			string(postID), string(author.ID), "Test Post")
		if err != nil {
			t.Fatalf("seed post: %v", err)
		}

		// File a report (public endpoint).
		rec, _ := f.doAs(t, author.Token, "POST", "/api/v1/reports", map[string]any{
			"target_type": "journey_post", "target_id": string(postID),
			"reason": "spam", "detail": "iklan",
		})
		if rec.Code != http.StatusAccepted {
			t.Fatalf("file report: %d %s", rec.Code, rec.Body.String())
		}

		// Admin lists queue → 1 open.
		rec, queue := f.do(t, "GET", "/api/v1/admin/queue/reports?status=open", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("queue: %d", rec.Code)
		}
		items := queue["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("want 1 open report, got %d", len(items))
		}
		reportID := ids.ID(items[0].(map[string]any)["id"].(string))

		// Resolve → hide_content.
		rec, _ = f.do(t, "PATCH", "/api/v1/admin/queue/reports/"+string(reportID), map[string]any{
			"action": "hide_content", "resolution": "confirmed spam",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("resolve: %d %s", rec.Code, rec.Body.String())
		}

		// Post hidden in DB + owner notified.
		var modStatus string
		if err := f.pool.QueryRow(ctx,
			`SELECT moderation_status FROM journey_posts WHERE id = $1`, string(postID)).Scan(&modStatus); err != nil {
			t.Fatal(err)
		}
		if modStatus != "hidden" {
			t.Fatalf("moderation_status = %s, want hidden", modStatus)
		}
		var notifs int
		if err := f.pool.QueryRow(ctx,
			`SELECT count(*) FROM notifications WHERE user_id = $1 AND type = 'moderation_outcome'`,
			string(author.ID)).Scan(&notifs); err != nil {
			t.Fatal(err)
		}
		if notifs != 1 {
			t.Fatalf("want 1 moderation_outcome notification, got %d", notifs)
		}

		// Double-resolve → 409.
		rec, _ = f.do(t, "PATCH", "/api/v1/admin/queue/reports/"+string(reportID), map[string]any{
			"action": "dismiss",
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("re-resolve: %d, want 409", rec.Code)
		}
	})

	t.Run("badge_config_upsert", func(t *testing.T) {
		rec, list := f.do(t, "GET", "/api/v1/admin/badge-configs", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list: %d", rec.Code)
		}
		if len(list["items"].([]any)) < 6 {
			t.Fatalf("expected ≥6 seeded badge configs")
		}

		rec, _ = f.do(t, "POST", "/api/v1/admin/badge-configs", map[string]any{
			"key": "mountain_3", "threshold": 3, "name_id": "Penjelajah Muda", "name_en": "Young Explorer",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("upsert: %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("user_status_flow_and_stats", func(t *testing.T) {
		target := createUser(t, f.pool, "target-smoke@test.local", "Target")
		rec, _ := f.do(t, "PATCH", "/api/v1/admin/users/"+string(target.ID), map[string]any{"status": "suspended"})
		if rec.Code != http.StatusOK {
			t.Fatalf("suspend: %d %s", rec.Code, rec.Body.String())
		}
		var status string
		if err := f.pool.QueryRow(ctx, `SELECT status FROM users WHERE id = $1`, string(target.ID)).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status != "suspended" {
			t.Fatalf("status = %s, want suspended", status)
		}

		rec, stats := f.do(t, "GET", "/api/v1/admin/stats", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("stats: %d %s", rec.Code, rec.Body.String())
		}
		if _, ok := stats["mountains"]; !ok {
			t.Fatalf("stats missing mountains block: %v", stats)
		}
	})
}

// admin2Mountain re-reads a mountain through the writer for assertions.
func (f *adminFixture) admin2Mountain(ctx context.Context, id ids.ID) (*catDomain.Mountain, error) {
	w := catInfra.NewMountainWriter(f.pool)
	all, err := w.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, context.Canceled
}
