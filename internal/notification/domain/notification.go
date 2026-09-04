// Package domain holds the notification bounded context's entities and
// repository port (plan.md → DDD). Pure Go — no framework, no DB driver.
package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// Type enumerates the four in-app notification kinds in v1 (data-model →
// notifications; Decisions → Notifications).
type Type string

const (
	TypeBadgeEarned           Type = "badge_earned"
	TypePartnerRequestReceived Type = "partner_request_received"
	TypePartnerRequestAccepted Type = "partner_request_accepted"
	TypeModerationOutcome     Type = "moderation_outcome"
)

// Notification is a single in-app notification (data-model → notifications).
type Notification struct {
	ID        ids.ID
	UserID    ids.ID
	Type      Type
	Payload   map[string]any // flexible per-type JSON payload
	ReadAt    *time.Time
	CreatedAt time.Time
}

// IsUnread reports whether the notification has not been read.
func (n Notification) IsUnread() bool { return n.ReadAt == nil }

// Repository is the persistence port for notifications.
type Repository interface {
	// Create persists a new notification.
	Create(ctx context.Context, n *Notification) error
	// ListByUser returns notifications for a user, newest first, with limit/offset.
	ListByUser(ctx context.Context, userID ids.ID, limit, offset int) ([]Notification, int, error)
	// MarkRead sets read_at on a notification owned by the user.
	MarkRead(ctx context.Context, userID, notificationID ids.ID) error
	// UnreadCount returns the number of unread notifications for a user.
	UnreadCount(ctx context.Context, userID ids.ID) (int, error)
}
