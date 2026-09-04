package http

import (
	"testing"

	"hikingfo/backend/internal/identity/domain"
)

// TestHashTokenCompatibility verifies that platform.HashToken produces the
// same output as identity/domain.HashToken for the same input — the session
// cookie flow depends on this.
func TestHashTokenCompatibility(t *testing.T) {
	tokens := []string{"abc123", "", "a-very-long-token-with-special-chars!@#$%"}
	for _, tok := range tokens {
		platform := HashToken(tok)
		domain := domain.HashToken(tok)
		if platform != domain {
			t.Errorf("hash mismatch for %q: platform=%s domain=%s", tok, platform, domain)
		}
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"", "id"},
		{"en-US,en;q=0.9", "en"},
		{"id,en;q=0.8", "id"},
		{"zh-CN,zh;q=0.9", "id"}, // unsupported → default
		{"en", "en"},
		{"ID", "id"},
	}
	for _, tt := range tests {
		got := parseAcceptLanguage(tt.header)
		if got != tt.want {
			t.Errorf("parseAcceptLanguage(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestSessionUserIsActive(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"active", true},
		{"suspended", false},
		{"banned", false},
		{"", false},
	}
	for _, tt := range tests {
		u := &SessionUser{Status: tt.status}
		if got := u.IsActive(); got != tt.want {
			t.Errorf("SessionUser{Status:%q}.IsActive() = %v, want %v", tt.status, got, tt.want)
		}
	}
}
