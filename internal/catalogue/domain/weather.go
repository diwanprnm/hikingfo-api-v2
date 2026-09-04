package domain

import "time"

// WeatherSnapshot is the cached weather payload for a mountain (data-model →
// mountain_weather). The raw Open-Meteo JSON is stored in Payload; the
// application layer decides how to render it.
type WeatherSnapshot struct {
	MountainID string         `json:"mountain_id"`
	CapturedAt time.Time      `json:"captured_at"`
	IsLive     bool           `json:"is_live"`
	ExpiresAt  time.Time      `json:"-"`
	Payload    map[string]any `json:"data"` // Open-Meteo forecast shape
}
