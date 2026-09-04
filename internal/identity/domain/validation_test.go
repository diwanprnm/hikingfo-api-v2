package domain

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	valid := []string{"pendaki@example.com", "a.b+c@sub.example.co.id"}
	for _, e := range valid {
		if err := ValidateEmail(e); err != nil {
			t.Errorf("ValidateEmail(%q) unexpected error: %v", e, err)
		}
	}
	invalid := []string{"", "  ", "not-an-email", "a@b", "@b.com", strings.Repeat("a", 255) + "@x.com"}
	for _, e := range invalid {
		if err := ValidateEmail(e); err == nil {
			t.Errorf("ValidateEmail(%q) expected error", e)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	valid := []string{"pendakigunung", "12345678", "Passw0rd!-L0ng"}
	for _, p := range valid {
		if err := ValidatePassword(p); err != nil {
			t.Errorf("ValidatePassword(%q) unexpected error: %v", p, err)
		}
	}
	invalid := []string{"", "short", "       "}
	for _, p := range invalid {
		if err := ValidatePassword(p); err == nil {
			t.Errorf("ValidatePassword(%q) expected error", p)
		}
	}
}

func TestValidateDisplayName(t *testing.T) {
	if err := ValidateDisplayName("Budi Pendaki"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	for _, n := range []string{"", "   "} {
		if err := ValidateDisplayName(n); err == nil {
			t.Errorf("ValidateDisplayName(%q) expected error", n)
		}
	}
}
