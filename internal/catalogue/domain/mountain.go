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
	ID           ids.ID
	Slug         string
	Name         langtext.Text
	Aliases      []string
	Region       Region
	Province     string
	Location     langtext.Text
	Latitude     float64
	Longitude    float64
	PeakName     langtext.Text
	PeakHeightM  int
	Difficulty   int // 1–5 integer scale
	Status       PublishStatus
	DataMeta     map[string]FieldMeta // per-field provenance
	PhotoURL     string               // presigned URL or blob key
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// FieldMeta records the provenance of a single field (data-model → data_meta).
type FieldMeta struct {
	Source     string     `json:"source"`
	Reliability string   `json:"reliability"` // "official" | "community" | "reported"
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// Route is a mountain route (data-model → mountain_routes).
type Route struct {
	ID               ids.ID
	MountainID       ids.ID
	Name             langtext.Text
	DistanceKm       float64
	DurationHours    float64
	ElevationGainM   int
	EntryRequirement langtext.Text
	SortOrder        int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Basecamp is a mountain basecamp (data-model → mountain_basecamps).
type Basecamp struct {
	ID             ids.ID
	MountainID     ids.ID
	Name           langtext.Text
	Facilities     langtext.Text
	CostEstimate   langtext.Text
	IsPermitPoint  bool
	Latitude       *float64
	Longitude      *float64
	SortOrder      int
}

// MountainProfile is the full profile view returned by the application service.
type MountainProfile struct {
	Mountain   Mountain
	Routes     []Route
	Basecamps  []Basecamp
	Weather    *WeatherSnapshot
}
