// Package application holds the notification bounded context's use-cases.
// Pure Go — depends only on notification/domain and the shared kernel.
package application

import (
	"context"
	"time"

	"hikingfo/backend/internal/notification/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// Service exposes the notification use-cases to the interfaces layer.
type Service struct {
	repo domain.Repository
	now  func() time.Time
}

// New wires a Service.
func New(repo domain.Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Record creates a notification. Called by other contexts via the event bus
// (badge_earned, partner events, moderation outcome).
func (s *Service) Record(ctx context.Context, userID ids.ID, typ domain.Type, payload map[string]any) error {
	n := &domain.Notification{
		ID:        ids.New(),
		UserID:    userID,
		Type:      typ,
		Payload:   payload,
		CreatedAt: s.now(),
	}
	return s.repo.Create(ctx, n)
}

// List returns a page of notifications for the user (newest first).
func (s *Service) List(ctx context.Context, userID ids.ID, page, pageSize int) ([]domain.Notification, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	return s.repo.ListByUser(ctx, userID, pageSize, offset)
}

// MarkRead marks a single notification as read.
func (s *Service) MarkRead(ctx context.Context, userID, notificationID ids.ID) error {
	if err := s.repo.MarkRead(ctx, userID, notificationID); err != nil {
		return kerr.WrapNotFound("notification not found", err)
	}
	return nil
}

// UnreadCount returns the number of unread notifications.
func (s *Service) UnreadCount(ctx context.Context, userID ids.ID) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}
