package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"hikingfo/backend/internal/platform/storage"
)

// T024: Principle I/II hygiene — public reads carry photo_url but never
// photo_key, admin ids, or edit-log data.
func TestPublicReadsNeverLeakAdminData(t *testing.T) {
	f := newAdminFixture(t)
	id := newPhotoMountain(t, f, "leak-mtn")

	// Attach a photo + make an attributed sub-object edit so the DB really has
	// admin data to leak.
	key := "uploads/" + id.String() + ".png"
	f.blobs.Put(t.Context(), storage.Key(key), strings.NewReader("p"), "image/png")
	if rec, _ := putPhoto(t, f, id, map[string]any{"key": key, "reason": "secret admin note"}); rec.Code != http.StatusOK {
		t.Fatalf("set photo: %d %s", rec.Code, rec.Body.String())
	}
	rec, _ := f.do(t, "POST", "/api/v1/admin/mountains/"+string(id)+"/routes", map[string]any{
		"name": map[string]string{"id": "Via"}, "duration_hours": 1,
		"distance_km": 1, "elevation_gain_m": 1, "reason": "internal memo",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create route: %d %s", rec.Code, rec.Body.String())
	}

	scan := func(t *testing.T, body []byte) {
		t.Helper()
		s := string(body)
		// NOTE: the presigned photo_url embeds the key path by design (R1) —
		// so we never scan for the raw key itself; photo_key/edit-log/admin
		// material must be absent.
		for _, needle := range []string{"photo_key", "admin_edit_log", "secret admin note", "internal memo", string(f.admin.ID)} {
			if strings.Contains(s, needle) {
				t.Fatalf("public body leaks %q: %s", needle, s)
			}
		}
	}

	t.Run("list", func(t *testing.T) {
		rec, _ := f.doAs(t, "", "GET", "/api/v1/mountains?q=leak", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("public list: %d (unauthenticated member token?)", rec.Code)
		}
		scan(t, rec.Body.Bytes())
		// The photo must still be visible to the public as a URL.
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		items, _ := out["items"].([]any)
		found := false
		for _, it := range items {
			m, _ := it.(map[string]any)
			if m["slug"] == "leak-mtn" {
				found = true
				if url, _ := m["photo_url"].(string); !strings.Contains(url, key) {
					t.Fatalf("list item lacks presigned photo_url: %v", m["photo_url"])
				}
			}
		}
		if !found {
			t.Fatalf("mountain missing from public list")
		}
	})

	t.Run("profile", func(t *testing.T) {
		rec, _ := f.doAs(t, "", "GET", "/api/v1/mountains/leak-mtn", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("public profile: %d", rec.Code)
		}
		scan(t, rec.Body.Bytes())
	})

	// Edit-log is admin-surface only: anonymous request must 401/403.
	if rec, _ := f.doAs(t, "", "GET", "/api/v1/admin/mountains/"+string(id)+"/edit-log", nil); rec.Code < 400 {
		t.Fatalf("unauthenticated edit-log got %d, want 401/403", rec.Code)
	}
}
