// Package domain holds the catalogue bounded context's entities, value objects
// and repository ports (plan.md → DDD). Pure Go — no framework, no DB driver.
package domain

import (
	"time"

	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// Region is the island-cluster enum (data-model → mountains.region).
type Region string

const (
	RegionJawa                Region = "Jawa"
	RegionSumatra             Region = "Sumatra"
	RegionBaliNusaTenggara    Region = "Bali & Nusa Tenggara"
	RegionKalimantan          Region = "Kalimantan"
	RegionSulawesi            Region = "Sulawesi"
	RegionMaluku              Region = "Maluku"
	RegionPapua               Region = "Papua"
)

// PublishStatus is the admin publish lifecycle (data-model → mountains.status).
type PublishStatus string

const (
	StatusDraft    PublishStatus = "draft"
	StatusPublished PublishStatus = "published"
	StatusHidden   PublishStatus = "hidden"
)

// Mountain is the core curated reference object (data-model → mountains).
type Mountain struct {
	ID          ids.ID               `json:"id"`
	Slug        string               `json:"slug"`
	Name        langtext.Text        `json:"name"`
	Aliases     []string             `json:"aliases"`
	Region      Region               `json:"region"`
	Province    string               `json:"province"`
	Location    langtext.Text        `json:"location"`
	Latitude    float64              `json:"latitude"`
	Longitude   float64              `json:"longitude"`
	PeakName    langtext.Text        `json:"peak_name"`
	PeakHeightM int                  `json:"peak_height_m"`
	Difficulty  int                  `json:"difficulty"` // 1–5 integer scale
	Status      PublishStatus        `json:"status"`
	DataMeta    map[string]FieldMeta `json:"data_meta"` // per-field provenance
	PhotoKey    string               `json:"-"`         // opaque blob key (never public)
	PhotoURL    string               `json:"photo_url,omitempty"` // presigned read URL (derived, not stored)
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// FieldMeta records the provenance of a single field (data-model → data_meta).
type FieldMeta struct {
	Source     string     `json:"source"`
	Reliability string   `json:"reliability"` // "official" | "community" | "reported"
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// Route is a mountain route (data-model → mountain_routes).
type Route struct {
	ID               ids.ID        `json:"id"`
	MountainID       ids.ID        `json:"mountain_id"`
	Name             langtext.Text `json:"name"`
	DistanceKm       float64       `json:"distance_km"`
	DurationHours    float64       `json:"duration_hours"`
	ElevationGainM   int           `json:"elevation_gain_m"`
	EntryRequirement langtext.Text `json:"entry_requirements"`
	SortOrder        int           `json:"-"`
	CreatedAt        time.Time     `json:"-"`
	UpdatedAt        time.Time     `json:"-"`
}

// Basecamp is a mountain basecamp (data-model → mountain_basecamps).
type Basecamp struct {
	ID            ids.ID        `json:"id"`
	MountainID    ids.ID        `json:"mountain_id"`
	Name          langtext.Text `json:"name"`
	Facilities    langtext.Text `json:"facilities"`
	CostEstimate  langtext.Text `json:"cost_estimate"`
	IsPermitPoint bool          `json:"is_permit_point"`
	Latitude      *float64      `json:"latitude,omitempty"`
	Longitude     *float64      `json:"longitude,omitempty"`
	SortOrder     int           `json:"-"`
}

// MountainPhoto is a curated gallery image attached to a mountain
// (004 data-model → mountain_photos). FR-001/FR-013/FR-014.
type MountainPhoto struct {
	ID         ids.ID     `json:"id"`
	MountainID ids.ID     `json:"-"`
	PhotoKey   string     `json:"-"` // opaque blob key, never public
	PhotoURL   string     `json:"photo_url,omitempty"` // presigned read URL (derived)
	CreatedAt  time.Time  `json:"created_at"`
}

// MountainProfile is the full profile view returned by the application service.
type MountainProfile struct {
	Mountain  Mountain         `json:"mountain"`
	Routes    []Route          `json:"routes"`
	Basecamps []Basecamp       `json:"basecamps"`
	Weather   *WeatherSnapshot `json:"weather,omitempty"`
}
