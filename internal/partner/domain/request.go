package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// RequestStatus is the lifecycle state of a partner request (data-model → partner_requests.status).
type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusAccepted  RequestStatus = "accepted"
	RequestStatusDeclined  RequestStatus = "declined"
	RequestStatusExpired   RequestStatus = "expired"
	RequestStatusWithdrawn RequestStatus = "withdrawn"
)

// PartnerRequest is a directed request from one user to another to hike
// together on a given mountain and date range (data-model → partner_requests).
type PartnerRequest struct {
	ID          ids.ID
	FromUserID  ids.ID
	ToUserID    ids.ID
	NoticeID    *ids.ID
	MountainID  ids.ID
	TripStart   time.Time
	TripEnd     time.Time
	Message     string
	Status      RequestStatus
	MatchedAt   *time.Time
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// RequestFilter carries query parameters for listing requests.
type RequestFilter struct {
	Direction string // "sent" or "received"
	Status    *RequestStatus
	Limit     int
	Offset    int
}

// RequestRepository is the persistence port for partner requests.
type RequestRepository interface {
	// Create persists a new request.
	Create(ctx context.Context, r *PartnerRequest) error
	// FindByID loads a request by id.
	FindByID(ctx context.Context, id ids.ID) (*PartnerRequest, error)
	// ListByUser returns requests where the user is sender or receiver, filtered.
	ListByUser(ctx context.Context, userID ids.ID, filter RequestFilter) ([]PartnerRequest, int, error)
	// UpdateStatus sets the status and optionally matched_at.
	UpdateStatus(ctx context.Context, id ids.ID, status RequestStatus, matchedAt *time.Time) error
	// Withdraw marks a request as withdrawn (sender only, pending only).
	Withdraw(ctx context.Context, id, userID ids.ID) error
	// HasAcceptedMatch checks whether two users already have an accepted match on a mountain.
	HasAcceptedMatch(ctx context.Context, userA, userB, mountainID ids.ID) (bool, error)
}

// CandidateFilter carries the search query for finding partner candidates.
type CandidateFilter struct {
	MountainID ids.ID
	TripStart  time.Time
	TripEnd    time.Time
	Limit      int
	Offset     int
}

// CandidateProfile is the limited user profile returned by search (no contacts).
type CandidateProfile struct {
	UserID           ids.ID  `json:"id"`
	DisplayName      string  `json:"display_name"`
	AvatarURL        string  `json:"avatar_url"`
	ExperienceLevel  string  `json:"experience_level"`
	HomeRegion       string  `json:"home_region"`
	Bio              string  `json:"bio"`
}

// CandidateResult wraps a single search result with a relevance score.
type CandidateResult struct {
	User      CandidateProfile `json:"user"`
	Relevance float64           `json:"relevance"`
}

// SearchResult is a page of candidate results.
type SearchResult struct {
	Items []CandidateResult `json:"items"`
	Total int               `json:"total"`
}

// ProfileReader is a cross-context port for reading limited user profiles.
// Implemented by the identity application service.
type ProfileReader interface {
	// LimitedProfile returns the public-safe profile fields for a set of user IDs.
	LimitedProfile(ctx context.Context, userID ids.ID) (*CandidateProfile, error)
}

// RevealedContact mirrors the identity private channels in partner terms. It is
// the ONLY carrier of contact data that leaves the identity context, and it may
// only be fetched through Revealer after a mutual accepted match (FR-013/SC-006).
type RevealedContact struct {
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	WhatsApp  string `json:"whatsapp"`
	Instagram string `json:"instagram"`
}

// ContactRevealer is the cross-context port that returns a matched counterpart's
// private channels. The partner context owns the mutual-match check and calls
// this ONLY after it has confirmed status=accepted in both directions; identity
// simply supplies the private data for the already-authorised user (DDD rule:
// partner authorises, identity never re-checks match state it cannot see).
// Implemented by the identity application service via the composition root.
type ContactRevealer interface {
	// RevealContacts returns the private channels of ownerID. The caller must
	// have established a mutual accepted match before invoking this.
	RevealContacts(ctx context.Context, ownerID ids.ID) (*RevealedContact, error)
}
