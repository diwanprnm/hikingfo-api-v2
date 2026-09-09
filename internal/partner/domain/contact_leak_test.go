// T017 (003) — Principle I evidence: the partner search payload type carries
// only LimitedProfile fields. Marshal the response structs and assert no
// contact channel key can ever appear (email/phone/whatsapp/instagram are
// exclusively RevealedContact fields, reachable only via the reveal endpoint).
package domain_test

import (
	"encoding/json"
	"strings"
	"testing"

	"hikingfo/backend/internal/partner/domain"
)

func TestSearchPayloadNeverCarriesContacts(t *testing.T) {
	res := domain.SearchResult{
		Items: []domain.CandidateResult{{
			User: domain.CandidateProfile{
				DisplayName:     "Astri",
				ExperienceLevel: "intermediate",
				HomeRegion:      "Jawa Barat",
				Bio:             "suka sunrise",
			},
			Relevance: 0.8,
		}},
		Total: 1,
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	for _, secret := range []string{"email", "phone", "whatsapp", "instagram"} {
		if strings.Contains(strings.ToLower(got), secret) {
			t.Errorf("search payload leaks %q: %s", secret, got)
		}
	}

	// Positive control: the reveal type does carry them, and is a distinct type.
	rc, err := json.Marshal(domain.RevealedContact{Email: "a@b.c"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(rc), "email") {
		t.Errorf("RevealedContact should be the only contact carrier, got %s", rc)
	}
}
