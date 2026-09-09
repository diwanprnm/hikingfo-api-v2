package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hikingfo/backend/internal/platform/storage"
	"hikingfo/backend/internal/shared/ids"
)

func storageKey(s string) storage.Key { return storage.Key(s) }

// newPhotoMountain creates a mountain via the admin API and returns its id.
func newPhotoMountain(t *testing.T, f *adminFixture, slug string) ids.ID {
	t.Helper()
	rec, out := f.do(t, "POST", "/api/v1/admin/mountains", map[string]any{
		"slug": slug, "name": map[string]string{"id": "Gunung " + slug},
		"region": "Jawa", "location": map[string]string{"id": "Jawa"},
		"peak_name": map[string]string{"id": "Puncak " + slug}, "peak_height_m": 1000,
		"difficulty": 2, "status": "published",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", slug, rec.Code, rec.Body.String())
	}
	idStr, _ := out["id"].(string)
	if idStr == "" {
		idStr, _ = out["ID"].(string)
	}
	return ids.ID(idStr)
}

func putPhoto(t *testing.T, f *adminFixture, mountainID ids.ID, body map[string]any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	return f.do(t, http.MethodPut, "/api/v1/admin/mountains/"+string(mountainID)+"/photo", body)
}

// T008: PUT/DELETE photo endpoints — presigned url, replace + prior-key
// cleanup, remove → null, 404, revision + edit-log provenance.
func TestAdminPhoto(t *testing.T) {
	f := newAdminFixture(t)
	id := newPhotoMountain(t, f, "photo-mtn")
	key1 := "uploads/" + ids.New().String() + ".png"
	key2 := "uploads/" + ids.New().String() + ".png"

	// Unknown mountain → 404.
	if rec, _ := putPhoto(t, f, ids.New(), map[string]any{"key": key1}); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown mountain: got %d, want 404", rec.Code)
	}
	// Key not shaped like an upload → 400 naming `key`.
	rec, body := putPhoto(t, f, id, map[string]any{"key": "secrets/steal.png"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad key: got %d %s, want 400", rec.Code, rec.Body.String())
	}
	if fb, ok := body["error"].(map[string]any)["fields"].(map[string]any); !ok || fb["key"] == nil {
		t.Fatalf("error fields must name key: %s", rec.Body.String())
	}

	// Set photo → response carries presigned photo_url.
	f.blobs.Put(context.Background(), storageKey(key1), strings.NewReader("png"), "image/png")
	rec, body = putPhoto(t, f, id, map[string]any{"key": key1, "reason": "summit shot"})
	if rec.Code != http.StatusOK {
		t.Fatalf("set photo: %d %s", rec.Code, rec.Body.String())
	}
	if url, _ := body["photo_url"].(string); url == "" || !strings.Contains(url, key1) {
		t.Fatalf("photo_url = %v, want presigned url for %q", body["photo_url"], key1)
	}
	if _, leaked := body["photo_key"]; leaked {
		t.Fatalf("photo_key must never serialize: %s", rec.Body.String())
	}

	// Replace → prior key deleted best-effort.
	f.blobs.Put(context.Background(), storageKey(key2), strings.NewReader("png2"), "image/png")
	rec, body = putPhoto(t, f, id, map[string]any{"key": key2})
	if rec.Code != http.StatusOK {
		t.Fatalf("replace photo: %d %s", rec.Code, rec.Body.String())
	}
	if !f.blobs.has(key2) {
		t.Fatalf("new key %q should be stored", key2)
	}
	found := false
	for _, k := range f.blobs.deletedKeys() {
		if k == key1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("prior key %q not cleaned up; deletes = %v", key1, f.blobs.deletedKeys())
	}

	// Edit log has photo_set rows with admin + reason; revision snapshot made.
	rec, log := f.do(t, "GET", "/api/v1/admin/mountains/"+string(id)+"/edit-log", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit-log: %d", rec.Code)
	}
	items := log["items"].([]any)
	if len(items) < 2 {
		t.Fatalf("want ≥2 log rows (photo_set x2), got %d", len(items))
	}
	latest, _ := items[0].(map[string]any)
	if latest["action"] != "photo_set" || latest["entity_type"] != "mountain" {
		t.Fatalf("latest log row = %v, want photo_set/mountain", latest)
	}
	if latest["admin_id"] != string(f.admin.ID) {
		t.Fatalf("log admin_id = %v, want %v", latest["admin_id"], f.admin.ID)
	}
	if latest["reason"] != "" {
		t.Fatalf("second PUT had no reason, got %v", latest["reason"])
	}
	second, _ := items[1].(map[string]any)
	if second["reason"] != "summit shot" {
		t.Fatalf("first PUT reason = %v, want \"summit shot\"", second["reason"])
	}

	rec, revs := f.do(t, "GET", "/api/v1/admin/mountains/"+string(id)+"/revisions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("revisions: %d", rec.Code)
	}
	if r, ok := revs["items"].([]any); !ok || len(r) < 2 {
		t.Fatalf("photo changes must snapshot revisions, got %v", revs["items"])
	}

	// Remove photo → null/absent photo_url, photo_remove logged, key deleted.
	rec, _ = f.do(t, http.MethodDelete, "/api/v1/admin/mountains/"+string(id)+"/photo",
		map[string]any{"reason": "bad angle"})
	if rec.Code != http.StatusOK {
		t.Fatalf("remove photo: %d %s", rec.Code, rec.Body.String())
	}
	rec, m := f.do(t, "GET", "/api/v1/admin/mountains/"+string(id), nil)
	if url, ok := m["photo_url"]; ok && url != "" && url != nil {
		t.Fatalf("after remove photo_url = %v, want empty", url)
	}
	rec, log = f.do(t, "GET", "/api/v1/admin/mountains/"+string(id)+"/edit-log", nil)
	items = log["items"].([]any)
	if top, _ := items[0].(map[string]any); top["action"] != "photo_remove" || top["reason"] != "bad angle" {
		t.Fatalf("top log = %v, want photo_remove/bad angle", top)
	}

	// Mountain delete removes the blob best-effort (FR-014).
	id2 := newPhotoMountain(t, f, "photo-mtn-2")
	key3 := "uploads/" + ids.New().String() + ".png"
	f.blobs.Put(context.Background(), storageKey(key3), strings.NewReader("p"), "image/png")
	if rec, _ = putPhoto(t, f, id2, map[string]any{"key": key3}); rec.Code != http.StatusOK {
		t.Fatalf("set photo 2: %d %s", rec.Code, rec.Body.String())
	}
	rec, _ = f.do(t, http.MethodDelete, "/api/v1/admin/mountains/"+string(id2), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete mountain: %d", rec.Code)
	}
	deleted := false
	for _, k := range f.blobs.deletedKeys() {
		if k == key3 {
			deleted = true
		}
	}
	if !deleted {
		t.Fatalf("mountain delete must clean its photo; deletes=%v", f.blobs.deletedKeys())
	}

	// Edit log survives the delete (audit outlives entity — research R7).
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM admin_edit_log WHERE mountain_id = $1`, string(id2)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatalf("edit log for deleted mountain gone; audit must outlive the entity")
	}
}
