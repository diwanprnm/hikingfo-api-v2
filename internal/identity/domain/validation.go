package domain

import (
	"regexp"
	"strings"

	"hikingfo/backend/internal/shared/kerr"
)

// Validation rules for identity inputs (spec Decisions → Auth). Messages are
// short English defaults; the frontend owns locale display.

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// ValidateEmail returns a validation error for a malformed email.
func ValidateEmail(email string) *kerr.Error {
	email = strings.TrimSpace(email)
	if email == "" {
		return kerr.Validation("email is required").WithField("email", "required")
	}
	if len(email) > 254 {
		return kerr.Validation("email is too long").WithField("email", "max_length")
	}
	if !emailRe.MatchString(email) {
		return kerr.Validation("email is invalid").WithField("email", "invalid_format")
	}
	return nil
}

// ValidatePassword enforces the signup/reset policy: minimum 8 chars, not all
// spaces. Longer, passphrase-friendly (no forced charset mix — users in ID are
// often email+short password; strength is length-first).
func ValidatePassword(password string) *kerr.Error {
	if password == "" {
		return kerr.Validation("password is required").WithField("password", "required")
	}
	if len(password) < 8 {
		return kerr.Validation("password must be at least 8 characters").
			WithField("password", "min_length")
	}
	if len(password) > 128 {
		return kerr.Validation("password must be at most 128 characters").
			WithField("password", "max_length")
	}
	return nil
}

// ValidateDisplayName ensures a non-empty, length-bounded display name.
func ValidateDisplayName(name string) *kerr.Error {
	name = strings.TrimSpace(name)
	if name == "" {
		return kerr.Validation("display name is required").WithField("display_name", "required")
	}
	if len([]rune(name)) > 60 {
		return kerr.Validation("display name is too long").WithField("display_name", "max_length")
	}
	return nil
}
