package application

import (
	"strings"
	"testing"
)

// T015: route numeric validation (FR-012, SC-004) — every rejection names the
// exact offending field so the admin UI can highlight it.

func rt(distance, duration float64, gain int) RouteInput {
	return RouteInput{
		Name:          map[string]string{"id": "Via normal"},
		DistanceKm:    distance,
		DurationHours: duration,
		ElevationGainM: gain,
	}
}

func TestValidateRoute(t *testing.T) {
	cases := []struct {
		name    string
		in      RouteInput
		wantErr string // substring of the message; "" = must pass
		field   string // field key that must appear in Fields
	}{
		{"valid", rt(7, 6, 1500), "", ""},
		{"zero distance allowed", rt(0, 6, 1500), "", ""},
		{"zero elevation allowed", rt(7, 6, 0), "", ""},
		{"negative distance", rt(-1, 6, 1500), "distance_km", "distance_km"},
		{"zero duration rejected", rt(7, 0, 1500), "duration_hours", "duration_hours"},
		{"negative duration rejected", rt(7, -2, 1500), "duration_hours", "duration_hours"},
		{"negative elevation rejected", rt(7, 6, -10), "elevation_gain_m", "elevation_gain_m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRoute(tc.in)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected accept, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected rejection naming %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Message, tc.wantErr) {
				t.Fatalf("message %q must name field %q", err.Message, tc.wantErr)
			}
			if _, ok := err.Fields[tc.field]; !ok {
				t.Fatalf("Fields %v must contain key %q", err.Fields, tc.field)
			}
		})
	}
}

func bc(lat, lon *float64) BasecampInput {
	return BasecampInput{Name: map[string]string{"id": "Basecamp Satu"}, Latitude: lat, Longitude: lon}
}

// T020: basecamp coordinate validation (FR-010) — both-or-none, within range.
func TestValidateBasecamp(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		name  string
		in    BasecampInput
		field string // "" = must pass
	}{
		{"both null accepted", bc(nil, nil), ""},
		{"both valid accepted", bc(f(-7.5), f(110.2)), ""},
		{"edge values accepted", bc(f(-90), f(180)), ""},
		{"lat without lon", bc(f(-7.5), nil), "longitude"},
		{"lon without lat", bc(nil, f(110.2)), "latitude"},
		{"lat out of range", bc(f(91), f(0)), "latitude"},
		{"lat below range", bc(f(-91), f(0)), "latitude"},
		{"lon out of range", bc(f(0), f(181)), "longitude"},
		{"lon below range", bc(f(0), f(-181)), "longitude"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBasecamp(tc.in)
			if tc.field == "" {
				if err != nil {
					t.Fatalf("expected accept, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected rejection naming %q, got nil", tc.field)
			}
			if !strings.Contains(err.Message, tc.field) {
				t.Fatalf("message %q must name field %q", err.Message, tc.field)
			}
			if _, ok := err.Fields[tc.field]; !ok {
				t.Fatalf("Fields %v must contain key %q", err.Fields, tc.field)
			}
		})
	}
}
