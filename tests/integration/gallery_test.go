package integration

import (
	"net/http"
	"testing"

	"hikingfo/backend/internal/shared/ids"
)

// T025 (004 US3): admin gallery add/delete — profile exposes photos[],
// admin_edit_log rows carry photo_set/photo_remove, non-admin is 403, delete
// of a foreign photo id is 404.
func TestGalleryAdminAddDelete(t *testing.T) {
	f := newAdminFixture(t)
	id := newPhotoMountain(t, f, "gallery-mtn")

	// Empty gallery → photos [] (FR-003 always an array).
	rec, out := f.do(t, http.MethodGet, "/api/v1/mountains/gallery-mtn", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile: %d %s", rec.Code, rec.Body.String())
	}
	prof := out["photos"]
	arr, ok := prof.([]any)
	if !ok || len(arr) != 0 {
		t.Fatalf("expected empty photos array, got %#v", prof)
	}

	// Add two photos.
	key1 := "uploads/" + ids.New().String() + ".png"
	key2 := "uploads/" + ids.New().String() + ".png"
	rec, out = f.do(t, "POST", "/api/v1/admin/mountains/"+string(id)+"/photos", map[string]any{"photo_key": key1})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add photo 1: %d %s", rec.Code, rec.Body.String())
	}
	photo1 := ids.ID(out["id"].(string))
	if _, out2 := f.do(t, "POST", "/api/v1/admin/mountains/"+string(id)+"/photos", map[string]any{"photo_key": key2}); out2 == nil {
		t.Fatal("add photo 2: no body")
	}

	// Profile now lists both, presigned URLs present.
	rec, out = f.do(t, http.MethodGet, "/api/v1/mountains/gallery-mtn", nil)
	arr = out["photos"].([]any)
	if len(arr) != 2 {
		t.Fatalf("expected 2 photos, got %#v", out["photos"])
	}
	if p0, ok := arr[0].(map[string]any); !ok || p0["photo_url"] == nil || p0["photo_url"] == "" {
		t.Fatalf("photo missing presigned url: %#v", arr[0])
	}

	// Bad key → 400.
	if rec, _ := f.do(t, "POST", "/api/v1/admin/mountains/"+string(id)+"/photos", map[string]any{"photo_key": "secrets/x.png"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad key: got %d, want 400", rec.Code)
	}

	// Edit log carries attribution rows (constitution II).
	rec, out = f.do(t, http.MethodGet, "/api/v1/admin/mountains/"+string(id)+"/edit-log", nil)
	items := out["items"].([]any)
	seenSet := 0
	for _, it := range items {
		m := it.(map[string]any)
		if m["action"] == "photo_set" && m["entity_id"] != nil {
			seenSet++
		}
		if m["action"] == "photo_remove" {
			t.Fatal("unexpected photo_remove before any delete")
		}
	}
	if seenSet < 2 {
		t.Fatalf("expected ≥2 photo_set rows, got %d (items=%v)", seenSet, len(items))
	}

	// Non-admin → 403.
	member := createUser(t, f.pool, "gallery-member@test.local", "Member")
	if rec, _ := f.doAs(t, member.Token, "POST", "/api/v1/admin/mountains/"+string(id)+"/photos", map[string]any{"photo_key": "uploads/" + ids.New().String() + ".png"}); rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin add: got %d, want 403", rec.Code)
	}

	// Delete → 204, gone from profile, photo_remove logged.
	if rec, _ := f.do(t, http.MethodDelete, "/api/v1/admin/mountains/"+string(id)+"/photos/"+string(photo1), nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", rec.Code)
	}
	rec, out = f.do(t, http.MethodGet, "/api/v1/mountains/gallery-mtn", nil)
	if len(out["photos"].([]any)) != 1 {
		t.Fatalf("expected 1 photo after delete, got %v", out["photos"])
	}
	rec, out = f.do(t, http.MethodGet, "/api/v1/admin/mountains/"+string(id)+"/edit-log", nil)
	seenRemove := false
	for _, it := range out["items"].([]any) {
		if it.(map[string]any)["action"] == "photo_remove" {
			seenRemove = true
		}
	}
	if !seenRemove {
		t.Fatal("missing photo_remove edit-log row")
	}

	// Delete of a foreign id → 404.
	if rec, _ := f.do(t, http.MethodDelete, "/api/v1/admin/mountains/"+string(id)+"/photos/"+ids.New().String(), nil); rec.Code != http.StatusNotFound {
		t.Fatalf("foreign delete: got %d, want 404", rec.Code)
	}
}
