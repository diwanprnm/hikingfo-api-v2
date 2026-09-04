// Package ids provides the UUID identifier used across every bounded context.
// It wraps github.com/google/uuid so contexts depend on this kernel package
// rather than importing the third-party type directly (plan.md → shared kernel).
package ids

import (
	"encoding/json"
	"fmt"

	guuid "github.com/google/uuid"
)

// ID is a UUIDv4 rendered as the canonical 36-char string.
type ID string

// Nil is the zero, never-valid ID.
const Nil ID = ""

// New returns a fresh random (version 4) UUID.
func New() ID {
	return ID(guuid.NewString())
}

// Parse validates a canonical 36-char UUID string.
func Parse(s string) (ID, error) {
	u, err := guuid.Parse(s)
	if err != nil {
		return Nil, fmt.Errorf("ids: %w", err)
	}
	return ID(u.String()), nil
}

// Valid reports whether s is a well-formed canonical UUID.
func Valid(s string) bool {
	_, err := guuid.Parse(s)
	return err == nil
}

// String returns the canonical 36-char form.
func (id ID) String() string { return string(id) }

// IsNil reports whether this is the zero ID.
func (id ID) IsNil() bool { return id == Nil }

// MarshalJSON renders the ID as a bare JSON string.
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(id))
}

// UnmarshalJSON accepts a bare string.
func (id *ID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*id = Nil
		return nil
	}
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
