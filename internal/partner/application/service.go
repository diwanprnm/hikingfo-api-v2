// Package application holds the partner matching bounded context's use-cases.
// Pure Go — depends only on partner/domain and the shared kernel.
package application

import (
	"context"
	"time"

	"hikingfo/backend/internal/partner/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// Dependencies the partner use-cases need.
type Dependencies struct {
	Notices  domain.NoticeRepository
	Requests domain.RequestRepository
	Profiles domain.ProfileReader // cross-context: identity limited profiles
	Revealer domain.ContactRevealer
	Now      func() time.Time
}

// Service exposes the partner use-cases to the interfaces layer.
type Service struct{ d Dependencies }

// New wires a Service.
func New(d Dependencies) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{d: d}
}

// ---- Search ----------------------------------------------------------------

// SearchCandidates returns user profiles that posted open notices matching
// the given mountain and date range. Returns limited profiles only (no contacts).
func (s *Service) SearchCandidates(ctx context.Context, userID ids.ID, filter domain.CandidateFilter) (*domain.SearchResult, error) {
	notices, total, err := s.d.Notices.ListOpen(ctx, filter.MountainID, filter.TripStart, filter.TripEnd, filter.Limit, filter.Offset)
	if err != nil {
		return nil, kerr.WrapInternal("could not search candidates", err)
	}

	// Deduplicate by user — one profile per user.
	seen := make(map[ids.ID]bool)
	var results []domain.CandidateResult
	for _, n := range notices {
		if n.UserID == userID {
			continue // skip self
		}
		if seen[n.UserID] {
			continue
		}
		seen[n.UserID] = true

		profile, err := s.d.Profiles.LimitedProfile(ctx, n.UserID)
		if err != nil {
			continue // skip unresolvable profiles
		}
		results = append(results, domain.CandidateResult{
			User:      *profile,
			Relevance: 1.0, // v1: uniform relevance
		})
	}

	return &domain.SearchResult{Items: results, Total: total}, nil
}

// ---- Notices ---------------------------------------------------------------

// PublishNotice creates a new open partner notice.
func (s *Service) PublishNotice(ctx context.Context, userID ids.ID, mountainID ids.ID, tripStart, tripEnd time.Time, note string) (*domain.PartnerNotice, error) {
	now := s.d.Now()
	n := &domain.PartnerNotice{
		ID:         ids.New(),
		UserID:     userID,
		MountainID: mountainID,
		TripStart:  tripStart,
		TripEnd:    tripEnd,
		Note:       note,
		Status:     domain.NoticeStatusOpen,
		ExpiresAt:  domain.ComputeExpiry(tripEnd),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.d.Notices.Create(ctx, n); err != nil {
		return nil, kerr.WrapInternal("could not create notice", err)
	}
	return n, nil
}

// ListNotices returns the current user's notices.
func (s *Service) ListNotices(ctx context.Context, userID ids.ID, filter domain.NoticeFilter) ([]domain.PartnerNotice, int, error) {
	return s.d.Notices.ListByUser(ctx, userID, filter)
}

// WithdrawNotice marks a notice as withdrawn (owner only).
func (s *Service) WithdrawNotice(ctx context.Context, noticeID, userID ids.ID) error {
	return s.d.Notices.Withdraw(ctx, noticeID, userID)
}

// ---- Requests --------------------------------------------------------------

// SendRequest creates a new partner request.
func (s *Service) SendRequest(ctx context.Context, fromUserID ids.ID, toUserID ids.ID, mountainID ids.ID, tripStart, tripEnd time.Time, noticeID *ids.ID, message string) (*domain.PartnerRequest, error) {
	if fromUserID == toUserID {
		return nil, kerr.Validation("cannot send request to yourself")
	}

	// Check for duplicate accepted match.
	exists, err := s.d.Requests.HasAcceptedMatch(ctx, fromUserID, toUserID, mountainID)
	if err != nil {
		return nil, kerr.WrapInternal("could not check existing match", err)
	}
	if exists {
		return nil, kerr.Conflict("already matched with this user on this mountain")
	}

	now := s.d.Now()
	req := &domain.PartnerRequest{
		ID:         ids.New(),
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		NoticeID:   noticeID,
		MountainID: mountainID,
		TripStart:  tripStart,
		TripEnd:    tripEnd,
		Message:    message,
		Status:     domain.RequestStatusPending,
		CreatedAt:  now,
		ExpiresAt:  domain.ComputeExpiry(tripEnd),
	}
	if err := s.d.Requests.Create(ctx, req); err != nil {
		return nil, kerr.WrapInternal("could not create request", err)
	}
	return req, nil
}

// ListRequests returns the current user's requests (sent or received).
func (s *Service) ListRequests(ctx context.Context, userID ids.ID, filter domain.RequestFilter) ([]domain.PartnerRequest, int, error) {
	return s.d.Requests.ListByUser(ctx, userID, filter)
}

// AcceptRequest marks a request as accepted (recipient only).
// Also marks the reverse direction request if it exists, and updates the notice.
func (s *Service) AcceptRequest(ctx context.Context, requestID, recipientID ids.ID) error {
	req, err := s.d.Requests.FindByID(ctx, requestID)
	if err != nil {
		return kerr.NotFound("request not found")
	}
	if req.ToUserID != recipientID {
		return kerr.Forbidden("only the recipient can accept")
	}
	if req.Status != domain.RequestStatusPending {
		return kerr.Validation("request is not pending")
	}

	now := s.d.Now()

	// Accept this request.
	if err := s.d.Requests.UpdateStatus(ctx, requestID, domain.RequestStatusAccepted, &now); err != nil {
		return kerr.WrapInternal("could not accept request", err)
	}

	// Also accept the reverse direction if it exists and is pending.
	reverseFilter := domain.RequestFilter{Direction: "sent", Status: ptrStatus(domain.RequestStatusPending)}
	reverseReqs, _, _ := s.d.Requests.ListByUser(ctx, req.FromUserID, reverseFilter)
	for _, r := range reverseReqs {
		if r.ToUserID == req.FromUserID && r.MountainID == req.MountainID &&
			r.TripStart.Equal(req.TripStart) && r.TripEnd.Equal(req.TripEnd) {
			_ = s.d.Requests.UpdateStatus(ctx, r.ID, domain.RequestStatusAccepted, &now)
		}
	}

	// If this request is linked to a notice, mark it matched.
	if req.NoticeID != nil {
		_ = s.d.Notices.UpdateStatus(ctx, *req.NoticeID, domain.NoticeStatusMatched)
	}

	return nil
}

// DeclineRequest marks a request as declined (recipient only).
func (s *Service) DeclineRequest(ctx context.Context, requestID, recipientID ids.ID) error {
	req, err := s.d.Requests.FindByID(ctx, requestID)
	if err != nil {
		return kerr.NotFound("request not found")
	}
	if req.ToUserID != recipientID {
		return kerr.Forbidden("only the recipient can decline")
	}
	if req.Status != domain.RequestStatusPending {
		return kerr.Validation("request is not pending")
	}

	return s.d.Requests.UpdateStatus(ctx, requestID, domain.RequestStatusDeclined, nil)
}

// WithdrawRequest cancels a pending request (sender only).
func (s *Service) WithdrawRequest(ctx context.Context, requestID, senderID ids.ID) error {
	return s.d.Requests.Withdraw(ctx, requestID, senderID)
}

// ---- helpers ---------------------------------------------------------------

func ptrStatus(s domain.RequestStatus) *domain.RequestStatus { return &s }
