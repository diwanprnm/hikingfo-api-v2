// Package application holds the moderation bounded context's use-cases
// (contracts §9 → admin queue).
package application

import (
	"context"

	"hikingfo/backend/internal/moderation/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// SideEffects declares the cross-context actions a resolution may trigger.
// Declared here, wired at the composition root (plan.md → DDD boundaries).
type SideEffects struct {
	// HidePost sets a journey post's moderation_status to hidden.
	HidePost func(ctx context.Context, postID ids.ID) error
	// RemoveEvidence marks a hike log entry removed (de-counts badges).
	RemoveEvidence func(ctx context.Context, hikeID ids.ID) error
	// Notify records a moderation_outcome notification for the content owner.
	Notify func(ctx context.Context, userID ids.ID, payload map[string]any) error
}

// Service exposes the moderation admin use-cases.
type Service struct {
	repo      domain.Repository
	effects   SideEffects
	JourneyOwner func(ctx context.Context, postID ids.ID) (ids.ID, bool)
	HikeOwner    func(ctx context.Context, hikeID ids.ID) (ids.ID, bool)
	// Blocks persists directed user blocks (nil → block endpoints 501-style
	// internal error, matching other unwired effects).
	Blocks domain.BlockRepository
}

// New wires a Service. Owner lookups may be nil; notifications are then
// skipped for that target type.
func New(repo domain.Repository, effects SideEffects) *Service {
	return &Service{repo: repo, effects: effects}
}

// ReportView is the queue row projection returned to the admin UI.
type ReportView struct {
	ID         ids.ID      `json:"id"`
	ReporterID *ids.ID     `json:"reporter_id"`
	TargetType string      `json:"target_type"`
	TargetID   ids.ID      `json:"target_id"`
	FieldRef   string      `json:"field_ref,omitempty"`
	Reason     string      `json:"reason"`
	Detail     string      `json:"detail,omitempty"`
	Status     string      `json:"status"`
	Resolution string      `json:"resolution,omitempty"`
	CreatedAt  string      `json:"created_at"`
	ResolvedAt *string     `json:"resolved_at,omitempty"`
}

func view(r domain.Report) ReportView {
	v := ReportView{
		ID:         r.ID,
		ReporterID: r.ReporterID,
		TargetType: string(r.TargetType),
		TargetID:   r.TargetID,
		FieldRef:   r.FieldRef,
		Reason:     string(r.Reason),
		Detail:     r.Detail,
		Status:     string(r.Status),
		Resolution: r.Resolution,
		CreatedAt:  r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.ResolvedAt != nil {
		s := r.ResolvedAt.Format("2006-01-02T15:04:05Z07:00")
		v.ResolvedAt = &s
	}
	return v
}

// Create files a report (public endpoint — any visitor).
func (s *Service) Create(ctx context.Context, reporter *ids.ID, targetType, targetID, fieldRef, reason, detail string) (*ReportView, error) {
	switch domain.TargetType(targetType) {
	case domain.TargetJourneyPost, domain.TargetUserProfile,
		domain.TargetHikeEvidence, domain.TargetMountainField:
	default:
		return nil, kerr.Validation("invalid target_type")
	}
	switch domain.Reason(reason) {
	case domain.ReasonSpam, domain.ReasonFalseInfo, domain.ReasonSafetyCritical,
		domain.ReasonOffensive, domain.ReasonOther:
	default:
		return nil, kerr.Validation("invalid reason")
	}
	r := &domain.Report{
		ID:         ids.New(),
		ReporterID: reporter,
		TargetType: domain.TargetType(targetType),
		TargetID:   ids.ID(targetID),
		FieldRef:   fieldRef,
		Reason:     domain.Reason(reason),
		Detail:     detail,
		Status:     domain.StatusOpen,
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, kerr.WrapInternal("could not file report", err)
	}
	v := view(*r)
	return &v, nil
}

// List returns the queue filtered by status ("all" for everything).
func (s *Service) List(ctx context.Context, status string, limit int) ([]ReportView, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var st domain.Status
	if status != "" && status != "all" {
		st = domain.Status(status)
	}
	rows, err := s.repo.List(ctx, st, limit)
	if err != nil {
		return nil, kerr.WrapInternal("could not list reports", err)
	}
	out := make([]ReportView, 0, len(rows))
	for _, r := range rows {
		out = append(out, view(r))
	}
	return out, nil
}

// ResolveInput is the PATCH body for resolving/dismissing a report.
type ResolveInput struct {
	// Action: "hide_content" | "dismiss" | "remove_evidence" | "edit_field"
	Action string `json:"action" binding:"required"`
	// Resolution is the admin note stored on the report.
	Resolution string `json:"resolution"`
	// EditField carries the corrected value when action = "edit_field"
	// ({"field": "...", "value_id": "...", "value_en": "..."} for now).
	EditField map[string]any `json:"edit_field"`
}

// Resolve applies the admin decision and its side effects.
func (s *Service) Resolve(ctx context.Context, reportID ids.ID, in ResolveInput) (*ReportView, error) {
	r, err := s.repo.FindByID(ctx, reportID)
	if err != nil {
		return nil, kerr.NotFound("report not found")
	}
	if r.Status == domain.StatusResolved || r.Status == domain.StatusDismissed {
		return nil, kerr.Conflict("report already closed")
	}

	var outcome string
	var notified bool
	switch domain.Resolution(in.Action) {
	case domain.ResolutionHideContent:
		if r.TargetType != domain.TargetJourneyPost {
			return nil, kerr.Validation("hide_content applies to journey_post reports")
		}
		if s.effects.HidePost == nil {
			return nil, kerr.Internal("hide_content not wired")
		}
		if err := s.effects.HidePost(ctx, r.TargetID); err != nil {
			return nil, kerr.WrapInternal("could not hide post", err)
		}
		outcome = "content_hidden"
		notified = s.notifyOwner(ctx, s.JourneyOwner, r.TargetID, r, outcome)
	case domain.ResolutionRemoveEvidence:
		if r.TargetType != domain.TargetHikeEvidence {
			return nil, kerr.Validation("remove_evidence applies to hike_evidence reports")
		}
		if s.effects.RemoveEvidence == nil {
			return nil, kerr.Internal("remove_evidence not wired")
		}
		if err := s.effects.RemoveEvidence(ctx, r.TargetID); err != nil {
			return nil, kerr.WrapInternal("could not remove evidence", err)
		}
		outcome = "evidence_removed"
		notified = s.notifyOwner(ctx, s.HikeOwner, r.TargetID, r, outcome)
	case domain.ResolutionDismiss:
		outcome = "dismissed"
	case domain.ResolutionEditField:
		if r.TargetType != domain.TargetMountainField {
			return nil, kerr.Validation("edit_field applies to mountain_field reports")
		}
		outcome = "field_edited"
	default:
		return nil, kerr.Validation("invalid action")
	}

	newStatus := domain.StatusResolved
	if domain.Resolution(in.Action) == domain.ResolutionDismiss {
		newStatus = domain.StatusDismissed
	}
	if err := s.repo.UpdateStatus(ctx, r.ID, newStatus, in.Resolution); err != nil {
		return nil, kerr.WrapInternal("could not close report", err)
	}
	_ = notified
	r.Status = newStatus
	r.Resolution = in.Resolution
	v := view(*r)
	return &v, nil
}

// notifyOwner best-effort records a moderation_outcome notification for the
// content owner. Never fails the resolve.
func (s *Service) notifyOwner(ctx context.Context, lookup func(ctx context.Context, id ids.ID) (ids.ID, bool), targetID ids.ID, r *domain.Report, outcome string) bool {
	if lookup == nil || s.effects.Notify == nil {
		return false
	}
	owner, ok := lookup(ctx, targetID)
	if !ok {
		return false
	}
	payload := map[string]any{
		"report_id":   string(r.ID),
		"target_type": string(r.TargetType),
		"target_id":   string(r.TargetID),
		"outcome":     outcome,
	}
	if r.FieldRef != "" {
		payload["field_ref"] = r.FieldRef
	}
	_ = s.effects.Notify(ctx, owner, payload)
	return true
}

// Counts returns open/resolved/dismissed counts for GET /admin/stats.
func (s *Service) Counts(ctx context.Context) (map[string]int, error) {
	m, err := s.repo.CountByStatus(ctx)
	if err != nil {
		return nil, kerr.WrapInternal("could not count reports", err)
	}
	return m, nil
}

// ---- blocks (contracts §5 — pre-match safety, US5) --------------------------

// Block records blocker → blocked. Self-blocking is rejected.
func (s *Service) Block(ctx context.Context, blocker, blocked ids.ID) error {
	if blocker == blocked {
		return kerr.Validation("cannot block yourself")
	}
	if s.Blocks == nil {
		return kerr.Internal("blocks not wired")
	}
	if err := s.Blocks.Insert(ctx, blocker, blocked); err != nil {
		return kerr.WrapInternal("could not block user", err)
	}
	return nil
}

// Unblock removes a block the requester owns. Removing a non-existent block
// is a no-op (idempotent DELETE per contracts).
func (s *Service) Unblock(ctx context.Context, blocker, blocked ids.ID) error {
	if s.Blocks == nil {
		return kerr.Internal("blocks not wired")
	}
	if err := s.Blocks.Delete(ctx, blocker, blocked); err != nil {
		return kerr.WrapInternal("could not unblock user", err)
	}
	return nil
}

// IsBlocked reports whether blocker has blocked blocked.
func (s *Service) IsBlocked(ctx context.Context, blocker, blocked ids.ID) (bool, error) {
	if s.Blocks == nil {
		return false, nil
	}
	return s.Blocks.Exists(ctx, blocker, blocked)
}
