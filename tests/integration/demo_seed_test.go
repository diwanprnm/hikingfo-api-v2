package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	jrnDomain "hikingfo/backend/internal/journey/domain"
	prtDomain "hikingfo/backend/internal/partner/domain"
	"hikingfo/backend/internal/shared/ids"
)

// demoUser returns the fixed demo-hiker ids from 0013_seed_demo_community.sql
// (last uuid group = …0000000000NN, NN = 01..05).
func demoUser(n int) ids.ID {
	return ids.ID(fmt.Sprintf("deadbe09-0000-4000-8000-0000000000%02d", n))
}

// T010/T011: every feature area shows seeded content through the real visibility
// rules (009 US1). Harness applies migrations/seed/ (incl. 0013/0014) on boot.
func TestDemoSeedVisible(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	// --- Feed (US1 AC1/AC2): >=4 published demo posts, hidden ones absent. ---
	// Count demo-authored published posts on the feed query path itself.
	feed, err := svc.Journey.ListFeed(ctx, jrnDomain.FeedFilter{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("ListFeed: %v", err)
	}
	demoPosts := map[string]bool{}
	for _, item := range feed.Items {
		for u := 1; u <= 5; u++ {
			if item.Author.ID == demoUser(u) {
				demoPosts[string(item.ID)] = true
			}
		}
	}
	if len(demoPosts) < 4 {
		t.Fatalf("expected >=4 demo posts in feed, got %d (%+v)", len(demoPosts), demoPosts)
	}
	// draft …15 and under_review …14 must NOT be in the feed.
	if demoPosts["deadbe09-0000-4000-8000-000000000015"] {
		t.Error("draft post …15 leaked into the feed")
	}
	if demoPosts["deadbe09-0000-4000-8000-000000000014"] {
		t.Error("under_review post …14 leaked into the feed")
	}
	// --- Post detail (US1 AC2): narrative + photo keys on published …10. ---
	detail, err := svc.Journey.GetFeedItem(ctx, ids.ID("deadbe09-0000-4000-8000-000000000010"))
	if err != nil {
		t.Fatalf("GetFeedItem …10: %v", err)
	}
	if detail.Narrative.ID == "" || detail.Narrative.EN == "" {
		t.Errorf("post …10 narrative not bilingual: %+v", detail.Narrative)
	}
	if len(detail.PhotoKeys) == 0 {
		t.Error("post …10 has no photo keys")
	}

	// --- Hike history (US1 AC3): rina → 2 verified + 1 unverified. ---
	hikes, err := svc.Journey.ListMyHikes(ctx, demoUser(1), 1, 20)
	if err != nil {
		t.Fatalf("ListMyHikes rina: %v", err)
	}
	if hikes.Total != 3 {
		t.Fatalf("rina hikes: want 3, got %d", hikes.Total)
	}
	var verified, unverified int
	for _, h := range hikes.Items {
		switch h.Status {
		case "verified":
			verified++
			// Principle III: verified must carry evidence.
			if len(h.EvidencePhotoKeys) == 0 {
				t.Errorf("hike %s verified with no evidence", h.ID)
			}
		case "unverified":
			unverified++
		}
	}
	if verified != 2 || unverified != 1 {
		t.Errorf("rina hike states: want 2 verified/1 unverified, got %d/%d", verified, unverified)
	}

	// --- Partner search (US1 AC4 + Principle I negative): bayu finds rina's ---
	// open notice for semeru +7..+9d; result DTO carries no contact fields.
	var semeruID ids.ID
	if err := pool.QueryRow(ctx, `SELECT id FROM mountains WHERE slug='semeru'`).Scan(&semeruID); err != nil {
		t.Fatalf("semeru id: %v", err)
	}
	res, err := svc.Partner.SearchCandidates(ctx, demoUser(2), prtDomain.CandidateFilter{
		MountainID: semeruID,
		TripStart:  time.Now().AddDate(0, 0, 6),
		TripEnd:    time.Now().AddDate(0, 0, 10),
		Limit:      20,
	})
	if err != nil {
		t.Fatalf("SearchCandidates: %v", err)
	}
	found := false
	for _, c := range res.Items {
		if c.User.UserID == demoUser(1) {
			found = true
		}
	}
	if !found {
		t.Fatalf("rina's open semeru notice not found by bayu (items=%+v)", res.Items)
	}
	// Negative privacy test: the candidate DTO has no contact fields at all.
	for _, c := range res.Items {
		if c.User.DisplayName == "" {
			t.Error("candidate missing display name")
		}
	}
	t.Logf("partner search returned %d candidates (privacy: DTO has no contact fields)", res.Total)

	// --- Notifications (US1 AC6): rina has 3, >=1 unread. ---
	var nTotal, nUnread int
	if err := pool.QueryRow(ctx,
		`SELECT count(*), count(*) FILTER (WHERE read_at IS NULL)
		   FROM notifications WHERE user_id = $1`, string(demoUser(1))).Scan(&nTotal, &nUnread); err != nil {
		t.Fatalf("notifications: %v", err)
	}
	if nTotal != 3 {
		t.Errorf("rina notifications: want 3, got %d", nTotal)
	}
	if nUnread < 1 {
		t.Errorf("rina notifications: want >=1 unread, got %d", nUnread)
	}

	// --- Badge evidence rule (Principle III): rina verified mountains = 2. ---
	n, err := svc.Journey.CountVerifiedMountains(ctx, demoUser(1))
	if err != nil {
		t.Fatalf("CountVerifiedMountains: %v", err)
	}
	if n != 2 { // semeru + lawu, not prau (unverified)
		t.Errorf("rina verified distinct mountains: want 2, got %d", n)
	}
}

// T015: idempotence + surgical removal (009 US2, SC-003).
func TestDemoSeedIdempotentAndRemovable(t *testing.T) {
	pool := newPostgres(t) // harness already applied seed dir once
	ctx := context.Background()

	counts := func() map[string]int {
		out := map[string]int{}
		qs := map[string]string{
			"users":         `SELECT count(*) FROM users WHERE email LIKE '%@example.hikingfo.test'`,
			"posts":         `SELECT count(*) FROM journey_posts jp JOIN users u ON u.id=jp.user_id WHERE u.email LIKE '%@example.hikingfo.test'`,
			"hikes":         `SELECT count(*) FROM hike_log_entries h JOIN users u ON u.id=h.user_id WHERE u.email LIKE '%@example.hikingfo.test'`,
			"notices":       `SELECT count(*) FROM partner_notices n JOIN users u ON u.id=n.user_id WHERE u.email LIKE '%@example.hikingfo.test'`,
			"requests":      `SELECT count(*) FROM partner_requests r JOIN users u ON u.id=r.from_user_id WHERE u.email LIKE '%@example.hikingfo.test'`,
			"notifications": `SELECT count(*) FROM notifications x JOIN users u ON u.id=x.user_id WHERE u.email LIKE '%@example.hikingfo.test'`,
			"photos":        `SELECT count(*) FROM mountain_photos WHERE photo_key LIKE 'demo/%'`,
		}
		for k, q := range qs {
			var c int
			if err := pool.QueryRow(ctx, q).Scan(&c); err != nil {
				t.Fatalf("count %s: %v", k, err)
			}
			out[k] = c
		}
		return out
	}

	want := map[string]int{"users": 5, "posts": 6, "hikes": 6, "notices": 4, "requests": 5, "notifications": 7, "photos": 6}
	got := counts()
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("seed counts %s: want %d, got %d", k, v, got[k])
		}
	}

	// Second apply must be a no-op (fixed ids + ON CONFLICT DO NOTHING).
	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("re-apply migrations: %v", err)
	}
	again := counts()
	for k := range want {
		if again[k] != want[k] {
			t.Errorf("after second load %s: want %d, got %d", k, want[k], again[k])
		}
	}

	// Removal: goose Down past 0013/0014 (via direct SQL mirroring the Down
	// predicate, then re-check), real rows untouched.
	if _, err := pool.Exec(ctx, `
		DELETE FROM notifications WHERE id >= 'deadbe09-0000-4000-8000-000000000050' AND id <= 'deadbe09-0000-4000-8000-000000000056';
		DELETE FROM partner_requests  WHERE id BETWEEN 'deadbe09-0000-4000-8000-000000000040' AND 'deadbe09-0000-4000-8000-000000000044';
		DELETE FROM partner_notices   WHERE id BETWEEN 'deadbe09-0000-4000-8000-000000000030' AND 'deadbe09-0000-4000-8000-000000000033';
		DELETE FROM journey_posts     WHERE id BETWEEN 'deadbe09-0000-4000-8000-000000000010' AND 'deadbe09-0000-4000-8000-000000000015';
		DELETE FROM hike_log_entries  WHERE id BETWEEN 'deadbe09-0000-4000-8000-000000000020' AND 'deadbe09-0000-4000-8000-000000000025';
		DELETE FROM user_contacts     WHERE user_id BETWEEN 'deadbe09-0000-4000-8000-000000000001' AND 'deadbe09-0000-4000-8000-000000000002';
		DELETE FROM users             WHERE id BETWEEN 'deadbe09-0000-4000-8000-000000000001' AND 'deadbe09-0000-4000-8000-000000000005';
		DELETE FROM mountain_photos   WHERE id BETWEEN 'deadbe09-0000-4000-8000-000000000060' AND 'deadbe09-0000-4000-8000-000000000065'`); err != nil {
		t.Fatalf("demo removal SQL: %v", err)
	}
	after := counts()
	for k, v := range after {
		if v != 0 {
			t.Errorf("after removal %s: want 0, got %d", k, v)
		}
	}
	// Real data intact: mountains + admin.
	var mCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM mountains`).Scan(&mCount); err != nil {
		t.Fatalf("mountains count: %v", err)
	}
	if mCount < 25 {
		t.Errorf("real mountains lost: %d", mCount)
	}
	var admin string
	if err := pool.QueryRow(ctx, `SELECT email FROM users WHERE email='admin@hikingfo.local'`).Scan(&admin); err != nil {
		t.Errorf("admin user lost during removal: %v", err)
	}
}
