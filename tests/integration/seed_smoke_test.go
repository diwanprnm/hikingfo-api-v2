package integration

import (
	"context"
	"testing"

	catDomain "hikingfo/backend/internal/catalogue/domain"
)

// T102 / T041 companion: every seeded mountain is published and carries the
// core sourced fields — location, ≥1 route, cost provenance.
func TestSeedSmoke(t *testing.T) {
	pool := newPostgres(t) // migrations + migrations/seed applied by the harness
	svc := newServices(pool)
	ctx := context.Background()

	// pageSize 100 > seed size so one page covers all rows.
	res, err := svc.Catalogue.SearchAndFilter(ctx, catDomain.SearchFilter{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("SearchAndFilter: %v", err)
	}
	if res.Total < 25 {
		t.Fatalf("expected ≥25 seeded mountains, got %d", res.Total)
	}

	for _, m := range res.Items {
		if m.Location.ID == "" && m.Location.EN == "" {
			t.Errorf("%s: location missing", m.Slug)
		}
		if _, ok := m.DataMeta["location"]; !ok {
			t.Errorf("%s: location provenance missing", m.Slug)
		}
		if _, ok := m.DataMeta["cost"]; !ok {
			t.Errorf("%s: cost provenance missing", m.Slug)
		}
		profile, err := svc.Catalogue.GetProfile(ctx, m.Slug)
		if err != nil {
			t.Fatalf("%s: GetProfile: %v", m.Slug, err)
		}
		if len(profile.Routes) < 1 {
			t.Errorf("%s: no route", m.Slug)
		}
		if len(profile.Basecamps) < 1 {
			t.Errorf("%s: no basecamp", m.Slug)
		}
	}
}
