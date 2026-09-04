package langtext

import "testing"

func TestResolveFallback(t *testing.T) {
	// English rendering missing → falls back to Indonesian, never blank.
	txt := From("Gunung Rinjani", "")
	if got := txt.Resolve("en"); got != "Gunung Rinjani" {
		t.Errorf("Resolve(en) = %q, want ID fallback", got)
	}
	if got := txt.Resolve("id"); got != "Gunung Rinjani" {
		t.Errorf("Resolve(id) = %q, want ID", got)
	}

	full := From("Gunung Rinjani", "Mount Rinjani")
	if got := full.Resolve("en"); got != "Mount Rinjani" {
		t.Errorf("Resolve(en) = %q, want EN", got)
	}
}

func TestHas(t *testing.T) {
	if Nil.Has() {
		t.Error("Nil.Has() = true, want false")
	}
	if !From("x", "").Has() {
		t.Error("IDOnly.Has() = false, want true")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	in := From("Gunung Rinjani", "Mount Rinjani")
	b, err := in.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var out Text
	if err := out.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Errorf("round trip mismatch: %+v != %+v", out, in)
	}
}
