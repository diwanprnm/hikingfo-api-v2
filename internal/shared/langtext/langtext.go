// Package langtext models the bilingual {id, en} text value used across the
// catalogue and profile fields. EN falls back to ID at read time; a value may
// be null/empty for either language but never renders blank to the user
// (spec Decisions → i18n; data-model notation JSONB{id,en}).
package langtext

import (
	"encoding/json"
	"errors"
	"strings"
)

// Text holds the Indonesian and English renderings of a curated string.
// JSON shape mirrors the DB JSONB column: {"id": "...", "en": "..."}.
type Text struct {
	ID string `json:"id"`
	EN string `json:"en"`
}

// Nil is the zero value for an absent bilingual field ("not yet available").
var Nil = Text{}

// From builds a Text from two optional language renderings.
func From(id, en string) Text {
	return Text{ID: id, EN: en}
}

// IDOnly builds a Text carrying only Indonesian (used for optional-public
// user content that is ID-only in v1).
func IDOnly(id string) Text {
	return Text{ID: id}
}

// Resolve returns the rendering for lang ("id" or "en"); any non-"en" value is
// treated as Indonesian. English resolves to the Indonesian text when no
// English rendering exists (EN→ID fallback), so a field is never blank.
func (t Text) Resolve(lang string) string {
	if strings.EqualFold(lang, "en") {
		if t.EN != "" {
			return t.EN
		}
		return t.ID
	}
	return t.ID
}

// Has reports whether either rendering is present.
func (t Text) Has() bool {
	return t.ID != "" || t.EN != ""
}

// MarshalJSON serialises {id,en}, omitting empty members.
func (t Text) MarshalJSON() ([]byte, error) {
	aux := struct {
		ID string `json:"id,omitempty"`
		EN string `json:"en,omitempty"`
	}{ID: t.ID, EN: t.EN}
	return json.Marshal(aux)
}

// UnmarshalJSON accepts both {"id":...} and {"id":...,"en":...} documents.
func (t *Text) UnmarshalJSON(data []byte) error {
	aux := struct {
		ID *string `json:"id"`
		EN *string `json:"en"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ID != nil {
		t.ID = *aux.ID
	}
	if aux.EN != nil {
		t.EN = *aux.EN
	}
	return nil
}

// Scan implements sql.Scanner for a JSONB column.
func (t *Text) Scan(src any) error {
	if src == nil {
		*t = Nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	default:
		return errors.New("langtext: unsupported Scan source")
	}
}

// Value implements driver.Valuer for a JSONB column (nil when empty).
func (t Text) Value() (any, error) {
	if !t.Has() {
		return nil, nil
	}
	return json.Marshal(t)
}
