package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRateLimiterBlocksBursts verifies the limiter 429s after the burst budget
// is exhausted.
func TestRateLimiterBlocksBursts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/hit", RateLimiter(0.01, 3), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	codes := make([]int, 5)
	for i := range codes {
		codes[i] = performRequest(r, "POST", "/hit").Code
	}
	// First 3 pass, remaining 2 are rate limited.
	for i, want := range []int{200, 200, 200, 429, 429} {
		if codes[i] != want {
			t.Errorf("request %d: got %d, want %d", i, codes[i], want)
		}
	}
}

// TestSecurityHeadersPresent checks the hardened headers land on responses.
func TestSecurityHeadersPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := performRequest(r, "GET", "/x")
	for h, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	} {
		if got := w.Header().Get(h); got != want {
			t.Errorf("%s = %q, want %q", h, got, want)
		}
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy missing")
	}
}

// TestCSRFEnforcement verifies the CSRF middleware rejects mutating requests
// without a matching token and accepts them with one.
func TestCSRFEnforcement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CSRF("hikingfo_csrf"))
	r.POST("/mutate", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// No cookie → 403.
	if w := performRequestWithCookie(r, "POST", "/mutate", "", ""); w.Code != http.StatusForbidden {
		t.Errorf("no cookie: got %d, want 403", w.Code)
	}
	// Cookie present but header missing → 403.
	if w := performRequestWithCookie(r, "POST", "/mutate", "tok123", ""); w.Code != http.StatusForbidden {
		t.Errorf("header missing: got %d, want 403", w.Code)
	}
	// Matching cookie + header → 200.
	if w := performRequestWithCookie(r, "POST", "/mutate", "tok123", "tok123"); w.Code != http.StatusOK {
		t.Errorf("matching token: got %d, want 200", w.Code)
	}
}

// ---- helpers --------------------------------------------------------------

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	return performRequestWithCookie(r, method, path, "", "")
}

func performRequestWithCookie(r http.Handler, method, path, cookie, csrfHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "hikingfo_csrf", Value: cookie})
	}
	if csrfHeader != "" {
		req.Header.Set("X-CSRF-Token", csrfHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
