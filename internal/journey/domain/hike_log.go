// Package domain holds the journey bounded context's entities, value objects
// and repository ports. Pure Go — no framework, no DB driver.
package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// HikeStatus is the lifecycle status of a hike log entry.
type HikeStatus string

const (
	HikeStatusVerified HikeStatus = "verified"
	HikeStatusDisputed HikeStatus = "disputed"
	HikeStatusRemoved  HikeStatus = "removed"
)

// HikeLogEntry is the user's hiking evidence record (data-model → hike_log_entries).
type HikeLogEntry struct {
	ID               ids.ID
	UserID           ids.ID
	MountainID       ids.ID
	RouteID          *ids.ID       // nullable
	ClimbDate        time.Time     // date only
	EvidencePhotoKeys []string     // ≥1 required
	Status           HikeStatus
	PublishedPostID  *ids.ID       // nullable, linked journey post
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// HikeLogRepository is the persistence port for hike log entries.
type HikeLogRepository interface {
	Create(ctx context.Context, h *HikeLogEntry) error
	FindByID(ctx context.Context, id ids.ID) (*HikeLogEntry, error)
	ListByUser(ctx context.Context, userID ids.ID, limit, offset int) ([]HikeLogEntry, int, error)
	Delete(ctx context.Context, id, userID ids.ID) error
	CountVerifiedDistinctMountains(ctx context.Context, userID ids.ID) (int, error)
}
