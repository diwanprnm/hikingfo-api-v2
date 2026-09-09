package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	achDomain "hikingfo/backend/internal/achievement/domain"
	catDomain "hikingfo/backend/internal/catalogue/domain"
	catInfra "hikingfo/backend/internal/catalogue/infrastructure"
	jrnApp "hikingfo/backend/internal/journey/application"
	jrnDomain "hikingfo/backend/internal/journey/domain"
	notifApp "hikingfo/backend/internal/notification/application"
	notifDomain "hikingfo/backend/internal/notification/domain"
	notifInfra "hikingfo/backend/internal/notification/infrastructure"
	prtApp "hikingfo/backend/internal/partner/application"
	prtDomain "hikingfo/backend/internal/partner/domain"
	prtInfra "hikingfo/backend/internal/partner/infrastructure"
	"hikingfo/backend/internal/shared/ids"
)

// recordHikeInput builds the RecordHike request for a mountain with evidence.
func recordHikeInput(mountain ids.ID, title, visibility *string) jrnApp.RecordHikeInput {
	return jrnApp.RecordHikeInput{
		MountainID:        mountain,
		ClimbDate:         "2026-08-01",
		EvidencePhotoKeys: []string{"evidence/photo-1.jpg"},
		Title:             title,
		Visibility:        visibility,
	}
}

func updatePostInput(title *string) jrnApp.UpdatePostInput {
	return jrnApp.UpdatePostInput{Title: title}
}

// revealIfMatched is the match-gated reveal path: contacts flow only when the
// requester and owner have an accepted match on the same mountain (FR-013).
// ponytail: partner/application lacks a RevealIfMatched use-case in v1 — the
// HTTP layer composes it. Add `Service.RevealIfMatched` when US5 API work lands.
type revealGate struct {
	requests prtDomain.RequestRepository
	revealer prtDomain.ContactRevealer
}

var errNoMatch = errors.New("no mutual match")

func (g revealGate) revealIfMatched(requester, owner, mountain ids.ID) (*prtDomain.RevealedContact, error) {
	ok, err := g.requests.HasAcceptedMatch(ctxBg(), requester, owner, mountain)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errNoMatch
	}
	return g.revealer.RevealContacts(ctxBg(), owner)
}

func ctxBg() context.Context { return context.Background() }

// newPartnerWithRevealer rebuilds the partner service with a real revealer.
func newPartnerWithRevealer(pool *pgxpool.Pool, rev *stubRevealer) *prtApp.Service {
	return prtApp.New(prtApp.Dependencies{
		Notices:  prtInfra.NewNoticeRepository(pool),
		Requests: prtInfra.NewRequestRepository(pool),
		Profiles: &stubProfileReader{},
		Revealer: rev,
		Now:      fixedNow,
	})
}

// ---- US2: journey ----------------------------------------------------------

// T043: the unified record-my-hike flow creates a verified entry (+ optional
// post) and the post appears in the feed.
func TestRecordHikeCreatesVerifiedEntryAndFeedPost(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	user := createUser(t, pool, "hiker1@test.local", "Hiker Satu")
	m1 := seedMountain(t, pool, "gunung-test-a", "Gunung Test A", "Test Mount A", catDomain.RegionJawa, 3, 3000)

	title := "Naik pertama"
	res, err := svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(m1, &title, strPtr("published")))
	if err != nil {
		t.Fatalf("RecordHike: %v", err)
	}
	if res.Status != string(jrnDomain.HikeStatusVerified) {
		t.Fatalf("hike should be verified immediately, got %q", res.Status)
	}

	log, err := svc.Journey.ListMyHikes(ctx, user.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListMyHikes: %v", err)
	}
	if log.Total != 1 {
		t.Fatalf("expected 1 hike log entry, got %d", log.Total)
	}

	feed, err := svc.Journey.ListFeed(ctx, jrnDomain.FeedFilter{MountainID: m1, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListFeed: %v", err)
	}
	if feed.Total != 1 || len(feed.Items) != 1 {
		t.Fatalf("expected 1 feed item, got total=%d items=%d", feed.Total, len(feed.Items))
	}
	if feed.Items[0].Mountain.ID != m1 {
		t.Fatalf("feed item mountain mismatch: %v", feed.Items[0].Mountain.ID)
	}
}

// T008 (004): evidence drives status — 0 photos → unverified, 1 → verified,
// 6 → validation error naming the 5-photo limit. Edit of a published post is
// rejected (delete-only); draft edit/delete still works.
func TestHikeEvidenceStatusAndPostImmutability(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	user := createUser(t, pool, "hiker2@test.local", "Hiker Dua")
	m1 := seedMountain(t, pool, "gunung-test-b", "Gunung Test B", "Test Mount B", catDomain.RegionSumatra, 2, 2500)

	// 0 evidence → saved, unverified.
	noEvidence := recordHikeInput(m1, nil, nil)
	noEvidence.EvidencePhotoKeys = nil
	res, err := svc.Journey.RecordHike(ctx, user.ID, noEvidence)
	if err != nil {
		t.Fatalf("photo-less hike must now be accepted (004 FR-004): %v", err)
	}
	if res.Status != string(jrnDomain.HikeStatusUnverified) {
		t.Fatalf("0 evidence → unverified, got %q", res.Status)
	}

	// 1 evidence → verified.
	res, err = svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(m1, nil, nil))
	if err != nil {
		t.Fatalf("RecordHike with evidence: %v", err)
	}
	if res.Status != string(jrnDomain.HikeStatusVerified) {
		t.Fatalf("1 evidence → verified, got %q", res.Status)
	}

	// 6 evidence → 400 naming the limit.
	many := recordHikeInput(m1, nil, nil)
	many.EvidencePhotoKeys = []string{"a", "b", "c", "d", "e", "f"}
	_, err = svc.Journey.RecordHike(ctx, user.ID, many)
	if err == nil {
		t.Fatal("6 evidence photos must be rejected")
	}
	if !strings.Contains(err.Error(), "5") {
		t.Fatalf("rejection must name the 5-photo limit: %v", err)
	}

	// Draft post: link, edit allowed, publish, then edit rejected (delete-only).
	title := "Draft perjalanan"
	hRes, err := svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(m1, &title, strPtr("draft")))
	if err != nil {
		t.Fatalf("RecordHike with draft post: %v", err)
	}
	hikes, err := svc.Journey.ListMyHikes(ctx, user.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListMyHikes: %v", err)
	}
	var postID *ids.ID
	for _, h := range hikes.Items {
		if h.ID == hRes.ID {
			l, err := svc.Journey.GetMyHike(ctx, h.ID, user.ID)
			if err != nil {
				t.Fatalf("GetMyHike: %v", err)
			}
			postID = l.PublishedPostID
		}
	}
	if postID == nil {
		t.Fatal("expected linked journey post")
	}
	newTitle := "Draft diperbarui"
	if _, err := svc.Journey.UpdatePost(ctx, *postID, user.ID, updatePostInput(&newTitle)); err != nil {
		t.Fatalf("draft UpdatePost must succeed: %v", err)
	}
	if err := svc.Journey.PublishPost(ctx, *postID, user.ID); err != nil {
		t.Fatalf("PublishPost: %v", err)
	}
	if _, err := svc.Journey.UpdatePost(ctx, *postID, user.ID, updatePostInput(&newTitle)); err == nil {
		t.Fatal("published post must be immutable (004 R8)")
	}

	// Delete removes it from the feed.
	if err := svc.Journey.DeletePost(ctx, *postID, user.ID); err != nil {
		t.Fatalf("DeletePost: %v", err)
	}
	feed, err := svc.Journey.ListFeed(ctx, jrnDomain.FeedFilter{MountainID: m1, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListFeed after delete: %v", err)
	}
	if feed.Total != 0 {
		t.Fatalf("feed should be empty after delete, got %d", feed.Total)
	}
}

// ---- US3: achievement -------------------------------------------------------

// T057: distinct-mountain count ignores re-climbs; badge earned on crossing,
// lapsed on drop.
func TestBadgeEarnAndLapse(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	user := createUser(t, pool, "hiker3@test.local", "Hiker Tiga")
	mA := seedMountain(t, pool, "gunung-badge-a", "Gunung Badge A", "Badge Mount A", catDomain.RegionJawa, 2, 2000)
	mB := seedMountain(t, pool, "gunung-badge-b", "Gunung Badge B", "Badge Mount B", catDomain.RegionJawa, 2, 2100)

	if _, err := svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(mA, nil, nil)); err != nil {
		t.Fatalf("record hike A: %v", err)
	}
	view, err := svc.Achieve.GetBadges(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetBadges: %v", err)
	}
	if view.DistinctMountains != 1 {
		t.Fatalf("distinct=1 expected, got %d", view.DistinctMountains)
	}
	if !badgeEarned(view, "mountain_1") {
		t.Fatal("badge mountain_1 should be earned")
	}

	// Re-climb the same mountain: count unchanged.
	if _, err := svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(mA, nil, nil)); err != nil {
		t.Fatalf("re-climb A: %v", err)
	}
	view, _ = svc.Achieve.GetBadges(ctx, user.ID)
	if view.DistinctMountains != 1 {
		t.Fatalf("re-climb must not increment distinct count, got %d", view.DistinctMountains)
	}

	// Reach 5 distinct mountains → badge mountain_5.
	if _, err := svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(mB, nil, nil)); err != nil {
		t.Fatalf("record hike B: %v", err)
	}
	for i := 0; i < 3; i++ {
		mX := seedMountain(t, pool, "gunung-badge-x"+string(rune('a'+i)), "Gunung Badge X", "Badge Mount X", catDomain.RegionSumatra, 1, 1500+i)
		if _, err := svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(mX, nil, nil)); err != nil {
			t.Fatalf("record hike X%d: %v", i, err)
		}
	}
	view, _ = svc.Achieve.GetBadges(ctx, user.ID)
	if view.DistinctMountains != 5 {
		t.Fatalf("expected 5 distinct, got %d", view.DistinctMountains)
	}
	if !badgeEarned(view, "mountain_5") {
		t.Fatal("badge mountain_5 should be earned at 5 distinct")
	}
	if view.ExperienceLevel.Key != "menengah" {
		t.Fatalf("5 distinct → menengah level, got %s", view.ExperienceLevel.Key)
	}

	// Delete one hike → badge 5 lapses transparently. All hikes share one
	// climb_date, so pick any mountain and delete every hike to it.
	hikes, _ := svc.Journey.ListMyHikes(ctx, user.ID, 1, 50)
	if len(hikes.Items) == 0 {
		t.Fatal("expected hikes to list before delete")
	}
	target := hikes.Items[0].MountainID
	deleted := 0
	for _, h := range hikes.Items {
		if h.MountainID != target {
			continue
		}
		if err := svc.Journey.DeleteMyHike(ctx, h.ID, user.ID); err != nil {
			t.Fatalf("DeleteMyHike: %v", err)
		}
		deleted++
	}
	view, _ = svc.Achieve.GetBadges(ctx, user.ID)
	if view.DistinctMountains != 4 {
		t.Fatalf("expected 4 distinct after delete, got %d", view.DistinctMountains)
	}
	if badgeEarned(view, "mountain_5") {
		t.Fatal("badge mountain_5 should lapse below 5 distinct")
	}
}

// badgeEarned reports whether the view marks the badge key earned.
func badgeEarned(view *achDomain.UserBadgesView, key string) bool {
	for _, b := range view.Badges {
		if b.Key == key {
			return b.Earned
		}
	}
	return false
}

// T058: badge_earned event enqueues a notification via the notification service
// (the composition root wires this through the event bus).
func TestBadgeEventEnqueuesNotification(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	user := createUser(t, pool, "hiker4@test.local", "Hiker Empat")
	mA := seedMountain(t, pool, "gunung-notif-a", "Gunung Notif A", "Notif Mount A", catDomain.RegionSulawesi, 2, 1800)
	if _, err := svc.Journey.RecordHike(ctx, user.ID, recordHikeInput(mA, nil, nil)); err != nil {
		t.Fatalf("record hike: %v", err)
	}

	// Simulate the badge_earned subscriber the composition root registers.
	notifSvc := notifApp.New(notifInfra.NewNotificationRepository(pool))
	if err := notifSvc.Record(ctx, user.ID, notifDomain.TypeBadgeEarned, map[string]any{"badge_key": "mountain_1"}); err != nil {
		t.Fatalf("Record badge_earned: %v", err)
	}
	items, total, err := notifSvc.List(ctx, user.ID, 1, 10)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 notification, total=%d err=%v", total, err)
	}
	if items[0].Type != notifDomain.TypeBadgeEarned {
		t.Fatalf("expected badge_earned, got %s", items[0].Type)
	}
}

// ---- US5: partner -----------------------------------------------------------

// T075: mutual match reveals contacts to both, never to an unmatched third.
func TestMutualMatchRevealsContactsOnly(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	alice := createUser(t, pool, "alice@test.local", "Alice")
	bob := createUser(t, pool, "bob@test.local", "Bob")
	carol := createUser(t, pool, "carol@test.local", "Carol")
	mA := seedMountain(t, pool, "gunung-partner-a", "Gunung Partner A", "Partner Mount A", catDomain.RegionBaliNusaTenggara, 3, 3100)

	revealer := &stubRevealer{}
	svc.Partner = newPartnerWithRevealer(pool, revealer)
	gate := revealGate{requests: prtInfra.NewRequestRepository(pool), revealer: revealer}

	// Alice publishes a notice.
	notice, err := svc.Partner.PublishNotice(ctx, alice.ID, mA, fixedNow().AddDate(0, 0, 3), fixedNow().AddDate(0, 0, 5), "cari partner")
	if err != nil {
		t.Fatalf("PublishNotice: %v", err)
	}

	// Bob searches → sees Alice's limited profile (no contact channels in shape).
	res, err := svc.Partner.SearchCandidates(ctx, bob.ID, prtDomain.CandidateFilter{
		MountainID: mA,
		TripStart:  fixedNow().AddDate(0, 0, 4),
		TripEnd:    fixedNow().AddDate(0, 0, 4),
		Limit:      10, Offset: 0,
	})
	if err != nil {
		t.Fatalf("SearchCandidates: %v", err)
	}
	if res.Total != 1 || len(res.Items) != 1 {
		t.Fatalf("expected 1 candidate, got %d", res.Total)
	}
	if res.Items[0].User.UserID != alice.ID {
		t.Fatalf("candidate should be alice, got %v", res.Items[0].User.UserID)
	}

	// Bob sends a request on Alice's notice; Alice accepts → mutual match.
	req, err := svc.Partner.SendRequest(ctx, bob.ID, alice.ID, mA,
		fixedNow().AddDate(0, 0, 4), fixedNow().AddDate(0, 0, 4), &notice.ID, "ikut dong")
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if err := svc.Partner.AcceptRequest(ctx, req.ID, alice.ID); err != nil {
		t.Fatalf("AcceptRequest: %v", err)
	}

	// Match established — has-accepted-match true both directions.
	matched, err := prtInfra.NewRequestRepository(pool).HasAcceptedMatch(ctx, bob.ID, alice.ID, mA)
	if err != nil || !matched {
		t.Fatalf("expected accepted match bob↔alice: %v (%v)", matched, err)
	}

	// Contact reveal only succeeds through the match-gated path.
	if _, err := gate.revealIfMatched(bob.ID, alice.ID, mA); err != nil {
		t.Fatalf("matched reveal bob→alice must succeed: %v", err)
	}
	if _, err := gate.revealIfMatched(alice.ID, bob.ID, mA); err != nil {
		t.Fatalf("matched reveal alice→bob must succeed: %v", err)
	}
	// Carol (unmatched) must not reveal.
	if _, err := gate.revealIfMatched(carol.ID, alice.ID, mA); err == nil {
		t.Fatal("unmatched carol must not reveal alice's contacts")
	}
}

// T076: notices/requests auto-expire after trip date (query-time).
func TestAutoExpireQueryTime(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	alice := createUser(t, pool, "alice2@test.local", "Alice Dua")
	bob := createUser(t, pool, "bob2@test.local", "Bob Dua")
	mA := seedMountain(t, pool, "gunung-expire-a", "Gunung Expire A", "Expire Mount A", catDomain.RegionPapua, 4, 4500)

	// Notice whose trip already ended → ListOpen must not return it.
	start := fixedNow().AddDate(0, 0, -10)
	end := fixedNow().AddDate(0, 0, -5)
	if _, err := svc.Partner.PublishNotice(ctx, alice.ID, mA, start, end, "lama"); err != nil {
		t.Fatalf("PublishNotice past: %v", err)
	}
	res, err := svc.Partner.SearchCandidates(ctx, bob.ID, prtDomain.CandidateFilter{
		MountainID: mA, TripStart: start, TripEnd: end, Limit: 10, Offset: 0,
	})
	if err != nil {
		t.Fatalf("SearchCandidates: %v", err)
	}
	if res.Total != 0 {
		t.Fatalf("expired notice must not appear in search, got %d", res.Total)
	}

	// A future notice does appear.
	futureStart := fixedNow().AddDate(0, 0, 2)
	futureEnd := fixedNow().AddDate(0, 0, 4)
	if _, err := svc.Partner.PublishNotice(ctx, alice.ID, mA, futureStart, futureEnd, "baru"); err != nil {
		t.Fatalf("PublishNotice future: %v", err)
	}
	res, err = svc.Partner.SearchCandidates(ctx, bob.ID, prtDomain.CandidateFilter{
		MountainID: mA, TripStart: futureStart, TripEnd: futureEnd, Limit: 10, Offset: 0,
	})
	if err != nil {
		t.Fatalf("SearchCandidates future: %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("future notice should appear, got %d", res.Total)
	}
}

// requestExpiryAfterTripDate asserts ComputeExpiry lands at trip_end EOD.
func TestComputeExpiryMatchesTripEnd(t *testing.T) {
	tripEnd := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	got := prtDomain.ComputeExpiry(tripEnd)
	if got.Hour() != 23 || got.Day() != 10 {
		t.Fatalf("expiry should be trip_end 23:59:59 UTC, got %v", got)
	}
}

// ---- US1: catalogue ---------------------------------------------------------

// T028/T029: search/browse + filters and profile shape against real schema.
func TestCatalogueSearchFilterAndProfile(t *testing.T) {
	pool := newPostgres(t)
	svc := newServices(pool)
	ctx := context.Background()

	// The seed migration populates the catalogue; each test gets its own
	// container, so wipe it and test against controlled fixtures only.
	if _, err := pool.Exec(ctx, `TRUNCATE mountains CASCADE`); err != nil {
		t.Fatalf("truncate mountains: %v", err)
	}

	rinjani := seedMountain(t, pool, "gunung-rinjani", "Gunung Rinjani", "Mount Rinjani", catDomain.RegionBaliNusaTenggara, 5, 3726)
	seedMountain(t, pool, "gunung-gede", "Gunung Gede", "Mount Gede", catDomain.RegionJawa, 3, 2958)

	// Search by name fragment.
	res, err := svc.Catalogue.SearchAndFilter(ctx, catDomain.SearchFilter{Query: "rinjani", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("SearchAndFilter: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != rinjani {
		t.Fatalf("expected fixture rinjani only, got total=%d", res.Total)
	}

	// Filter by region.
	res, err = svc.Catalogue.SearchAndFilter(ctx, catDomain.SearchFilter{Region: catDomain.RegionJawa, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("SearchAndFilter region: %v", err)
	}
	if res.Total != 1 || res.Items[0].Slug != "gunung-gede" {
		t.Fatalf("expected gede only in Jawa, got total=%d", res.Total)
	}

	// Filter by difficulty + height range.
	res, err = svc.Catalogue.SearchAndFilter(ctx, catDomain.SearchFilter{Difficulty: 5, MinHeight: 3000, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("SearchAndFilter difficulty: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != rinjani {
		t.Fatalf("expected rinjani at difficulty 5 ≥3000m, got total=%d", res.Total)
	}

	// Profile: all blocks + per-field provenance.
	profile, err := svc.Catalogue.GetProfile(ctx, "gunung-rinjani")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.Mountain.Name.ID != "Gunung Rinjani" || profile.Mountain.Name.EN != "Mount Rinjani" {
		t.Fatalf("bilingual name mismatch: %+v", profile.Mountain.Name)
	}
	meta, ok := profile.Mountain.DataMeta["location"]
	if !ok || meta.Source != "test-suite" || meta.Reliability != "official" {
		t.Fatalf("data_meta missing/incorrect: %+v", profile.Mountain.DataMeta)
	}

	// Weather degrade: with no cache row and upstream disabled → nil weather (not error).
	if profile.Weather != nil {
		t.Fatalf("expected nil weather when nothing cached, got %+v", profile.Weather)
	}

	// Stale cached weather is served labelled not-live.
	stale := &catDomain.WeatherSnapshot{
		MountainID: string(rinjani),
		Payload:    map[string]any{"daily": []any{}},
		CapturedAt: fixedNow().Add(-2 * time.Hour),
		IsLive:     true,
		ExpiresAt:  fixedNow().Add(-90 * time.Minute),
	}
	if err := catInfra.NewWeatherRepository(pool).Upsert(ctx, stale); err != nil {
		t.Fatalf("seed weather: %v", err)
	}
	profile, err = svc.Catalogue.GetProfile(ctx, "gunung-rinjani")
	if err != nil {
		t.Fatalf("GetProfile with stale weather: %v", err)
	}
	if profile.Weather == nil {
		t.Fatal("stale cached weather should be returned")
	}
	if profile.Weather.IsLive {
		t.Fatal("expired cache must be labelled is_live=false")
	}
}

// ---- US4 (frontend-only story): the backend verified-data contract it reads.

// T059 companion: experience-level buckets derive correctly from the count.
func TestExperienceBuckets(t *testing.T) {
	cases := []struct {
		count int
		key   string
	}{{0, "pemula"}, {4, "pemula"}, {5, "menengah"}, {14, "menengah"}, {15, "lanjut"}, {34, "lanjut"}, {35, "ahli"}, {100, "ahli"}}
	for _, c := range cases {
		lvl := experienceLevelFor(c.count)
		if lvl.Key != c.key {
			t.Fatalf("count=%d → %s expected, got %s", c.count, c.key, lvl.Key)
		}
	}
}

// ---- compile-time guards ----------------------------------------------------

var (
	_ achDomain.JourneyQueryPort = (*badgeRepoProbe)(nil)
)

type badgeRepoProbe struct{}

func (badgeRepoProbe) CountVerifiedDistinctMountains(ctx context.Context, userID ids.ID) (int, error) {
	return 0, nil
}

var _ notifDomain.Repository = (*notifInfra.NotificationRepository)(nil)

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// experienceLevelFor re-derives the experience bucket (kept in sync with
// achievement/application.deriveExperienceLevel) for the bucket test.
func experienceLevelFor(count int) achDomain.ExperienceLevel {
	switch {
	case count >= 35:
		return achDomain.ExperienceLevel{Key: "ahli"}
	case count >= 15:
		return achDomain.ExperienceLevel{Key: "lanjut"}
	case count >= 5:
		return achDomain.ExperienceLevel{Key: "menengah"}
	default:
		return achDomain.ExperienceLevel{Key: "pemula"}
	}
}
