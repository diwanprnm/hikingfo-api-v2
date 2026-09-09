// Package domain holds the partner matching bounded context's entities,
// value objects and repository ports. Pure Go — no framework, no DB driver.
package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// NoticeStatus is the lifecycle state of a partner notice (data-model → partner_notices.status).
type NoticeStatus string

const (
	NoticeStatusOpen      NoticeStatus = "open"
	NoticeStatusMatched   NoticeStatus = "matched"
	NoticeStatusExpired   NoticeStatus = "expired"
	NoticeStatusWithdrawn NoticeStatus = "withdrawn"
)

// PartnerNotice is a public declaration that a user is looking for a hiking
// partner on a given mountain and date range (data-model → partner_notices).
type PartnerNotice struct {
	ID         ids.ID        `json:"id"`
	UserID     ids.ID        `json:"user_id"`
	MountainID ids.ID        `json:"mountain_id"`
	TripStart  time.Time     `json:"trip_start"`
	TripEnd    time.Time     `json:"trip_end"`
	Note       string        `json:"note"`
	Status     NoticeStatus  `json:"status"`
	ExpiresAt  time.Time     `json:"expires_at"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// NoticeFilter carries query parameters for listing notices.
type NoticeFilter struct {
	MountainID *ids.ID
	Status     *NoticeStatus
	Limit      int
	Offset     int
}

// NoticeRepository is the persistence port for partner notices.
type NoticeRepository interface {
	// Create persists a new notice.
	Create(ctx context.Context, n *PartnerNotice) error
	// FindByID loads a notice by id.
	FindByID(ctx context.Context, id ids.ID) (*PartnerNotice, error)
	// ListByUser returns notices owned by a user, filtered and paginated.
	ListByUser(ctx context.Context, userID ids.ID, filter NoticeFilter) ([]PartnerNotice, int, error)
	// ListOpen returns open, non-expired notices matching mountain + date overlap, limited to user profiles.
	ListOpen(ctx context.Context, mountainID ids.ID, tripStart, tripEnd time.Time, limit, offset int) ([]PartnerNotice, int, error)
	// UpdateStatus sets the status of a notice.
	UpdateStatus(ctx context.Context, id ids.ID, status NoticeStatus) error
	// Withdraw marks a notice as withdrawn (owner only).
	Withdraw(ctx context.Context, id, userID ids.ID) error
}
