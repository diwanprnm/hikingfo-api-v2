// Package domain holds the moderation bounded context: the report queue and
// its resolution side effects (data-model → reports/blocks; contracts §9).
package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// TargetType is the polymorphic report target (report_target_type enum).
type TargetType string

const (
	TargetJourneyPost   TargetType = "journey_post"
	TargetUserProfile   TargetType = "user_profile"
	TargetHikeEvidence  TargetType = "hike_evidence"
	TargetMountainField TargetType = "mountain_field"
)

// Reason is why something was reported (report_reason enum).
type Reason string

const (
	ReasonSpam           Reason = "spam"
	ReasonFalseInfo      Reason = "false_info"
	ReasonSafetyCritical Reason = "safety_critical"
	ReasonOffensive      Reason = "offensive"
	ReasonOther          Reason = "other"
)

// Status is the queue lifecycle (report_status enum).
type Status string

const (
	StatusOpen         Status = "open"
	StatusUnderReview  Status = "under_review"
	StatusResolved     Status = "resolved"
	StatusDismissed    Status = "dismissed"
)

// Report is one row of the moderation queue.
type Report struct {
	ID         ids.ID
	ReporterID *ids.ID
	TargetType TargetType
	TargetID   ids.ID
	FieldRef   string
	Reason     Reason
	Detail     string
	Status     Status
	Resolution string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

// Resolution is the admin decision on a report.
type Resolution string

const (
	// ResolutionHideContent hides a journey post (moderation_status → hidden).
	ResolutionHideContent Resolution = "hide_content"
	// ResolutionDismiss keeps the content and closes the report.
	ResolutionDismiss Resolution = "dismiss"
	// ResolutionRemoveEvidence de-counts hike evidence from badge math.
	ResolutionRemoveEvidence Resolution = "remove_evidence"
	// ResolutionEditField fixes a reported mountain field.
	ResolutionEditField Resolution = "edit_field"
)

// Repository is the persistence port for the report queue.
type Repository interface {
	Create(ctx context.Context, r *Report) error
	List(ctx context.Context, status Status, limit int) ([]Report, error)
	FindByID(ctx context.Context, id ids.ID) (*Report, error)
	UpdateStatus(ctx context.Context, id ids.ID, status Status, resolution string) error
	// Counts feeds GET /admin/stats report-aging.
	CountByStatus(ctx context.Context) (map[string]int, error)
}

// BlockRepository is the persistence port for directed user blocks
// (data-model → blocks; contracts §5 POST/DELETE /users/{id}/blocks).
type BlockRepository interface {
	// Insert records blocker → blocked. Idempotent (no error on re-block).
	Insert(ctx context.Context, blocker, blocked ids.ID) error
	// Delete removes the block; nil when it did not exist.
	Delete(ctx context.Context, blocker, blocked ids.ID) error
	// Exists reports whether the block is active.
	Exists(ctx context.Context, blocker, blocked ids.ID) (bool, error)
}
