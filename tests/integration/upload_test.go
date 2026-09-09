package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	plathttp "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/platform/storage"
)

// multipartUpload posts one file field to a bare /uploads route over the
// shared platform handler (contracts §8). store is a storage.BlobStore so the
// nil test passes a truly nil interface (not a typed nil pointer).
func multipartUpload(t *testing.T, store storage.BlobStore, filename string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/uploads", plathttp.UploadHandler(store))

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("write body: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/uploads", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// T007: POST /uploads — magic-byte sniffing, type whitelist, 10 MB cap.
func TestUploads(t *testing.T) {
	store := newFakeBlobStore()

	t.Run("valid_png_returns_key_and_presigned_url", func(t *testing.T) {
		rec := multipartUpload(t, store, "shot.png", tinyPNG)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d %s, want 201", rec.Code, rec.Body.String())
		}
		var out map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		key := out["key"]
		if !strings.HasPrefix(key, "uploads/") || !strings.HasSuffix(key, ".png") {
			t.Fatalf("key %q, want uploads/<uuid>.png", key)
		}
		if !strings.Contains(out["url"], "24h0m0s") && !strings.Contains(out["url"], "24h") {
			t.Fatalf("url %q lacks 24h ttl marker", out["url"])
		}
		if !store.has(key) {
			t.Fatalf("object %q not stored", key)
		}
	})

	t.Run("txt_renamed_png_rejected_400", func(t *testing.T) {
		rec := multipartUpload(t, store, "malware.png", []byte("this is plain text, not a png at all"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d, want 400", rec.Code)
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		errObj, _ := body["error"].(map[string]any)
		msg, _ := errObj["message"].(string)
		if !strings.Contains(msg, "image") {
			t.Fatalf("message %q must name the file-type problem", msg)
		}
	})

	t.Run("oversize_rejected_413", func(t *testing.T) {
		big := make([]byte, 11<<20) // 11 MB of zeros; MaxBytesReader fires mid-parse
		rec := multipartUpload(t, store, "big.png", big)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("got %d, want 413 (body %s)", rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		errObj, _ := body["error"].(map[string]any)
		msg, _ := errObj["message"].(string)
		if !strings.Contains(msg, "10 MB") {
			t.Fatalf("message %q must name the size limit", msg)
		}
	})

	t.Run("jpeg_accepted", func(t *testing.T) {
		jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, bytes.Repeat([]byte{0}, 600)...)
		rec := multipartUpload(t, store, "photo.jpg", jpeg)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d %s, want 201", rec.Code, rec.Body.String())
		}
	})
}

// TestUploadsNilStore proves the minio-less degradation path: a nil BlobStore
// answers with the placeholder shape instead of panicking.
func TestUploadsNilStore(t *testing.T) {
	rec := multipartUpload(t, nil, "x.png", tinyPNG)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 placeholder", rec.Code)
	}
}
