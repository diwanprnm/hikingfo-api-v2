package integration

import (
	"context"
	"net/http"
	"testing"

	"hikingfo/backend/internal/shared/ids"
)

// editLogCount counts admin_edit_log rows for a mountain.
func editLogCount(t *testing.T, f *adminFixture, mountainID ids.ID) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM admin_edit_log WHERE mountain_id = $1`, string(mountainID)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// T016/T020 (integration half): full route + basecamp CRUD through the admin
// endpoints; each 2xx appends an admin_edit_log row; validation 400s name the
// field; unknown ids 404.
func TestAdminRoutesBasecamps(t *testing.T) {
	f := newAdminFixture(t)
	mtn := newPhotoMountain(t, f, "subobj-mtn")

	// ---- routes -------------------------------------------------------------

	routeBody := func(distance, duration float64, gain int, reason string) map[string]any {
		return map[string]any{
			"name":          map[string]string{"id": "Via Sawah", "en": "Paddy Route"},
			"distance_km":   distance,
			"duration_hours": duration,
			"elevation_gain_m": gain,
			"entry_requirements": map[string]string{"id": "Simaksi"},
			"reason":        reason,
		}
	}

	// Validation first — 400s must not log.
	rec, body := f.do(t, "POST", "/api/v1/admin/mountains/"+string(mtn)+"/routes", routeBody(7, 0, 1500, "zero duration"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("duration 0: got %d %s, want 400", rec.Code, rec.Body.String())
	}
	if fb, ok := body["error"].(map[string]any)["fields"].(map[string]any); !ok || fb["duration_hours"] == nil {
		t.Fatalf("400 must name duration_hours: %s", rec.Body.String())
	}
	if editLogCount(t, f, mtn) != 0 {
		t.Fatalf("rejected writes must not append edit log")
	}
	if _, body = f.do(t, "POST", "/api/v1/admin/mountains/"+string(mtn)+"/routes", routeBody(-1, 6, 1500, "x")); rec.Code != http.StatusBadRequest {
		t.Fatalf("negative distance: %d, want 400", rec.Code)
	}
	if _, body = f.do(t, "POST", "/api/v1/admin/mountains/"+string(mtn)+"/routes", routeBody(7, 6, -5, "x")); rec.Code != http.StatusBadRequest {
		t.Fatalf("negative gain: %d, want 400", rec.Code)
	}

	// Create (valid).
	rec, out := f.do(t, "POST", "/api/v1/admin/mountains/"+string(mtn)+"/routes", routeBody(7.5, 6, 1500, "data dari pemdes"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create route: %d %s", rec.Code, rec.Body.String())
	}
	routeID := ids.ID(out["id"].(string))
	if editLogCount(t, f, mtn) != 1 {
		t.Fatalf("create route logged once, got %d", editLogCount(t, f, mtn))
	}
	// Same call again with different route = 2 logs.
	rec, _ = f.do(t, "POST", "/api/v1/admin/mountains/"+string(mtn)+"/routes", routeBody(4, 3, 800, ""))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create route 2: %d", rec.Code)
	}

	// Update.
	rec, _ = f.do(t, "PATCH", "/api/v1/admin/admin-routes/"+string(routeID), routeBody(8, 7, 1600, "koreksi"))
	if rec.Code != http.StatusOK {
		t.Fatalf("update route: %d %s", rec.Code, rec.Body.String())
	}
	// Unknown route → 404.
	if rec, _ = f.do(t, "PATCH", "/api/v1/admin/admin-routes/"+string(ids.New()), routeBody(1, 1, 1, "")); rec.Code != http.StatusNotFound {
		t.Fatalf("update unknown route: %d, want 404", rec.Code)
	}

	// Delete (no body).
	if rec, _ = f.do(t, "DELETE", "/api/v1/admin/admin-routes/"+string(routeID), nil); rec.Code != http.StatusOK {
		t.Fatalf("delete route: %d %s", rec.Code, rec.Body.String())
	}
	if rec, _ = f.do(t, "DELETE", "/api/v1/admin/admin-routes/"+string(routeID), nil); rec.Code != http.StatusNotFound {
		t.Fatalf("re-delete route: %d, want 404", rec.Code)
	}
	// create(2) + update(1) + delete(1) = 4 rows.
	if got := editLogCount(t, f, mtn); got != 4 {
		t.Fatalf("route lifecycle logged 4 rows, got %d", got)
	}

	// ---- basecamps ----------------------------------------------------------

	bcBody := func(lat, lon *float64) map[string]any {
		m := map[string]any{
			"name":            map[string]string{"id": "Basecamp Kali", "en": "Kali Basecamp"},
			"facilities":      map[string]string{"id": "Parkir, musala"},
			"is_permit_point": true,
			"reason":          "survei",
		}
		if lat != nil {
			m["latitude"] = *lat
		}
		if lon != nil {
			m["longitude"] = *lon
		}
		return m
	}
	p := func(v float64) *float64 { return &v }

	// Partial coords → 400 naming the missing half.
	rec, body = f.do(t, "POST", "/api/v1/admin/mountains/"+string(mtn)+"/basecamps", bcBody(p(-7.5), nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("partial coords: got %d %s, want 400", rec.Code, rec.Body.String())
	}
	if fb, ok := body["error"].(map[string]any)["fields"].(map[string]any); !ok || fb["longitude"] == nil {
		t.Fatalf("400 must name longitude: %s", rec.Body.String())
	}

	// No coords = valid (FR-010 edge case).
	rec, out = f.do(t, "POST", "/api/v1/admin/mountains/"+string(mtn)+"/basecamps", bcBody(nil, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("coordless basecamp: %d %s, want 201", rec.Code, rec.Body.String())
	}
	bcID := ids.ID(out["id"].(string))

	// Update with valid coords.
	rec, _ = f.do(t, "PATCH", "/api/v1/admin/admin-basecamps/"+string(bcID), bcBody(p(-7.4), p(110.4)))
	if rec.Code != http.StatusOK {
		t.Fatalf("update basecamp: %d %s", rec.Code, rec.Body.String())
	}
	// Out-of-range lat → 400.
	if rec, _ = f.do(t, "PATCH", "/api/v1/admin/admin-basecamps/"+string(bcID), bcBody(p(91), p(0))); rec.Code != http.StatusBadRequest {
		t.Fatalf("lat 91: %d, want 400", rec.Code)
	}
	// Delete.
	if rec, _ = f.do(t, "DELETE", "/api/v1/admin/admin-basecamps/"+string(bcID), map[string]any{"reason": "pindah lokasi"}); rec.Code != http.StatusOK {
		t.Fatalf("delete basecamp: %d", rec.Code)
	}
	// + create + update + delete = 3 more → 7 total.
	if got := editLogCount(t, f, mtn); got != 7 {
		t.Fatalf("route+basecamp lifecycle logged 7 rows, got %d", got)
	}

	// edit-log view: newest first, correct entity types.
	rec, log := f.do(t, "GET", "/api/v1/admin/mountains/"+string(mtn)+"/edit-log", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit-log: %d", rec.Code)
	}
	items := log["items"].([]any)
	if len(items) != 7 {
		t.Fatalf("edit-log view length = %d, want 7", len(items))
	}
	top, _ := items[0].(map[string]any)
	if top["entity_type"] != "basecamp" || top["action"] != "delete" || top["reason"] != "pindah lokasi" {
		t.Fatalf("top entry = %v, want basecamp/delete/'pindah lokasi'", top)
	}
	if top["admin_id"] != string(f.admin.ID) {
		t.Fatalf("attribution missing: %v", top["admin_id"])
	}
}
