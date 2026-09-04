package domain

import (
	"context"

	"hikingfo/backend/internal/shared/ids"
)

// MountainRepository is the persistence port for the catalogue.
type MountainRepository interface {
	// SearchAndFilter returns mountains matching the given filters, paginated.
	SearchAndFilter(ctx context.Context, f SearchFilter) ([]Mountain, int, error)
	// FindBySlug loads a mountain by its URL slug (includes data_meta).
	FindBySlug(ctx context.Context, slug string) (*Mountain, error)
	// FindByID loads a mountain by id.
	FindByID(ctx context.Context, id ids.ID) (*Mountain, error)
	// Routes returns all routes for a mountain, ordered by sort_order.
	Routes(ctx context.Context, mountainID ids.ID) ([]Route, error)
	// Basecamps returns all basecamps for a mountain, ordered by sort_order.
	Basecamps(ctx context.Context, mountainID ids.ID) ([]Basecamp, error)
	// Regions returns the distinct regions that have published mountains.
	Regions(ctx context.Context) ([]Region, error)
}

// SearchFilter carries the query parameters for mountain search/browse.
type SearchFilter struct {
	Query      string  // free-text search (name, aliases)
	Region     Region  // filter by island cluster
	Province   string  // filter by province
	Difficulty int     // exact difficulty match (0 = any)
	MinHeight  int     // min peak height in metres (0 = any)
	MaxHeight  int     // max peak height in metres (0 = any)
	Page       int     // 1-based page
	PageSize   int     // items per page (default 20)
}

// WeatherStore is the port for reading/writing cached weather data.
type WeatherStore interface {
	// Get returns the latest cached weather for a mountain (may be stale).
	Get(ctx context.Context, mountainID ids.ID) (*WeatherSnapshot, error)
	// Upsert writes or replaces the cached weather payload.
	Upsert(ctx context.Context, s *WeatherSnapshot) error
}
