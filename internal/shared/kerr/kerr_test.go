package kerr

import (
	"errors"
	"testing"
)

func TestFromPreservesKerr(t *testing.T) {
	in := Validation("nope").WithField("title", "required")
	out := From(in)
	if out != in {
		t.Errorf("From did not preserve *Error identity")
	}
	if out.Fields["title"] != "required" {
		t.Errorf("Fields lost in From: %+v", out.Fields)
	}
}

func TestFromWrapsUnknown(t *testing.T) {
	cause := errors.New("boom")
	out := From(cause)
	if out.Code != CodeInternal {
		t.Errorf("From(unknown) code = %q, want internal", out.Code)
	}
	if !errors.Is(out, cause) {
		t.Errorf("From should wrap the cause for errors.Is")
	}
	if out.Err == nil {
		t.Errorf("cause not retained")
	}
}

func TestWithFieldIsImmutable(t *testing.T) {
	base := Validation("bad")
	aug := base.WithField("a", "x")
	_ = aug.WithField("b", "y")
	if _, ok := base.Fields["a"]; ok {
		t.Errorf("WithField mutated the receiver")
	}
	if len(aug.Fields) != 1 {
		t.Errorf("aug has %d fields, want 1", len(aug.Fields))
	}
}
