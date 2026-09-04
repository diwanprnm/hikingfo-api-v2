// Package application holds the journey bounded context's use-cases.
// Pure Go — depends only on journey/domain and the shared kernel.
package application

import (
	"context"
	"time"

	"hikingfo/backend/internal/journey/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	"hikingfo/backend/internal/shared/langtext"
)

// Dependencies the journey use-cases need.
type Dependencies struct {
	Hikes domain.HikeLogRepository
	Posts domain.JourneyPostRepository
}

// Service exposes the journey use-cases to the interfaces layer.
type Service struct{ deps Dependencies }

// New wires a Service.
func New(deps Dependencies) *Service {
	return &Service{deps: deps}
}

// ---- Hike Log -------------------------------------------------------------

// RecordHikeInput is the request body for POST /hikes.
type RecordHikeInput struct {
	MountainID       ids.ID    `json:"mountain_id" binding:"required"`
	RouteID          *ids.ID   `json:"route_id"`
	ClimbDate        string    `json:"climb_date" binding:"required"`
	EvidencePhotoKeys []string `json:"evidence_photo_keys" binding:"required,min=1"`
	Title            *string   `json:"title"`
	Summary          *string   `json:"summary"`
	Narrative        *string   `json:"narrative"`
	PhotoKeys        *[]string `json:"photo_keys"`
	Visibility       *string   `json:"visibility"`
}

// RecordHikeResult is the response for POST /hikes.
type RecordHikeResult struct {
	ID     ids.ID `json:"id"`
	Status string `json:"status"`
}

// RecordHike creates a hike log entry (and optionally a journey post).
func (s *Service) RecordHike(ctx context.Context, userID ids.ID, in RecordHikeInput) (*RecordHikeResult, error) {
	if len(in.EvidencePhotoKeys) == 0 {
		return nil, kerr.Validation("evidence_photo_keys must contain at least 1 item")
	}

	climbDate, err := time.Parse("2006-01-02", in.ClimbDate)
	if err != nil {
		return nil, kerr.Validation("climb_date must be YYYY-MM-DD")
	}

	hike := &domain.HikeLogEntry{
		ID:                ids.New(),
		UserID:            userID,
		MountainID:        in.MountainID,
		RouteID:           in.RouteID,
		ClimbDate:         climbDate,
		EvidencePhotoKeys: in.EvidencePhotoKeys,
		Status:            domain.HikeStatusVerified,
	}

	if err := s.deps.Hikes.Create(ctx, hike); err != nil {
		return nil, kerr.WrapInternal("could not create hike log", err)
	}

	// If title is provided, also create a draft journey post.
	if in.Title != nil && *in.Title != "" {
		vis := domain.VisibilityDraft
		if in.Visibility != nil {
			vis = domain.PostVisibility(*in.Visibility)
		}
		post := &domain.JourneyPost{
			ID:               ids.New(),
			UserID:           userID,
			MountainID:       in.MountainID,
			RouteID:          in.RouteID,
			Title:            *in.Title,
			Summary:          make(map[string]any),
			Visibility:       vis,
			ModerationStatus: domain.ModerationVisible,
		}
		if in.Summary != nil {
			_ = []byte(*in.Summary) // validate JSON later if needed
		}
		if in.PhotoKeys != nil {
			post.PhotoKeys = *in.PhotoKeys
		}
		if err := s.deps.Posts.Create(ctx, post); err != nil {
			return nil, kerr.WrapInternal("could not create journey post", err)
		}
		hike.PublishedPostID = &post.ID
	}

	return &RecordHikeResult{ID: hike.ID, Status: string(hike.Status)}, nil
}

// HikeLogItem is a single hike log entry in list responses.
type HikeLogItem struct {
	ID                ids.ID     `json:"id"`
	MountainID        ids.ID     `json:"mountain_id"`
	RouteID           *ids.ID    `json:"route_id,omitempty"`
	ClimbDate         time.Time  `json:"climb_date"`
	EvidencePhotoKeys []string   `json:"evidence_photo_keys"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
}

// HikeLogListResult is the paginated list of a user's hike logs.
type HikeLogListResult struct {
	Items []HikeLogItem `json:"items"`
	Total int           `json:"total"`
}

// ListMyHikes returns the authenticated user's hike logs.
func (s *Service) ListMyHikes(ctx context.Context, userID ids.ID, page, pageSize int) (*HikeLogListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	items, total, err := s.deps.Hikes.ListByUser(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, kerr.WrapInternal("could not list hikes", err)
	}
	out := make([]HikeLogItem, len(items))
	for i, h := range items {
		out[i] = HikeLogItem{
			ID:                h.ID,
			MountainID:        h.MountainID,
			RouteID:           h.RouteID,
			ClimbDate:         h.ClimbDate,
			EvidencePhotoKeys: h.EvidencePhotoKeys,
			Status:            string(h.Status),
			CreatedAt:         h.CreatedAt,
		}
	}
	return &HikeLogListResult{Items: out, Total: total}, nil
}

// GetMyHike returns a single hike log entry owned by the user.
func (s *Service) GetMyHike(ctx context.Context, id, userID ids.ID) (*domain.HikeLogEntry, error) {
	h, err := s.deps.Hikes.FindByID(ctx, id)
	if err != nil {
		return nil, kerr.NotFound("hike not found")
	}
	if h.UserID != userID {
		return nil, kerr.Forbidden("not your hike")
	}
	return h, nil
}

// DeleteMyHike removes a hike log entry owned by the user.
func (s *Service) DeleteMyHike(ctx context.Context, id, userID ids.ID) error {
	if err := s.deps.Hikes.Delete(ctx, id, userID); err != nil {
		if _, ok := err.(domain.HikeLogNotFound); ok {
			return kerr.NotFound("hike not found")
		}
		return kerr.WrapInternal("could not delete hike", err)
	}
	return nil
}

// ---- Journey Post ---------------------------------------------------------

// CreatePostInput is the request body for POST /me/journeys.
type CreatePostInput struct {
	MountainID ids.ID          `json:"mountain_id" binding:"required"`
	RouteID    *ids.ID         `json:"route_id"`
	Title      string          `json:"title" binding:"required"`
	Summary    map[string]any  `json:"summary"`
	Narrative  string          `json:"narrative"`
	PhotoKeys  []string        `json:"photo_keys"`
	Visibility string          `json:"visibility"`
}

// CreatePost creates a new journey post (draft or published).
func (s *Service) CreatePost(ctx context.Context, userID ids.ID, in CreatePostInput) (*domain.JourneyPost, error) {
	vis := domain.VisibilityDraft
	if in.Visibility != "" {
		vis = domain.PostVisibility(in.Visibility)
	}
	post := &domain.JourneyPost{
		ID:               ids.New(),
		UserID:           userID,
		MountainID:       in.MountainID,
		RouteID:          in.RouteID,
		Title:            in.Title,
		Summary:          in.Summary,
		Visibility:       vis,
		ModerationStatus: domain.ModerationVisible,
	}
	if in.Narrative != "" {
		post.Narrative = langtext.IDOnly(in.Narrative)
	}
	if in.PhotoKeys != nil {
		post.PhotoKeys = in.PhotoKeys
	}
	if err := s.deps.Posts.Create(ctx, post); err != nil {
		return nil, kerr.WrapInternal("could not create post", err)
	}
	return post, nil
}

// UpdatePostInput is the request body for PATCH /me/journeys/:id.
type UpdatePostInput struct {
	Title      *string         `json:"title"`
	Summary    *map[string]any `json:"summary"`
	Narrative  *string         `json:"narrative"`
	PhotoKeys  *[]string       `json:"photo_keys"`
	Visibility *string         `json:"visibility"`
}

// UpdatePost patches a journey post owned by the user.
func (s *Service) UpdatePost(ctx context.Context, id, userID ids.ID, in UpdatePostInput) (*domain.JourneyPost, error) {
	post, err := s.deps.Posts.FindByID(ctx, id)
	if err != nil {
		return nil, kerr.NotFound("post not found")
	}
	if post.UserID != userID {
		return nil, kerr.Forbidden("not your post")
	}
	if in.Title != nil {
		post.Title = *in.Title
	}
	if in.Summary != nil {
		post.Summary = *in.Summary
	}
	if in.Narrative != nil {
		post.Narrative = langtext.IDOnly(*in.Narrative)
	}
	if in.PhotoKeys != nil {
		post.PhotoKeys = *in.PhotoKeys
	}
	if in.Visibility != nil {
		post.Visibility = domain.PostVisibility(*in.Visibility)
	}
	if err := s.deps.Posts.Update(ctx, post); err != nil {
		return nil, kerr.WrapInternal("could not update post", err)
	}
	return post, nil
}

// PublishPost transitions a draft post to published.
func (s *Service) PublishPost(ctx context.Context, id, userID ids.ID) error {
	if err := s.deps.Posts.Publish(ctx, id, userID); err != nil {
		if _, ok := err.(domain.JourneyPostNotFound); ok {
			return kerr.NotFound("post not found or already published")
		}
		return kerr.WrapInternal("could not publish post", err)
	}
	return nil
}

// DeletePost removes a journey post owned by the user.
func (s *Service) DeletePost(ctx context.Context, id, userID ids.ID) error {
	if err := s.deps.Posts.Delete(ctx, id, userID); err != nil {
		if _, ok := err.(domain.JourneyPostNotFound); ok {
			return kerr.NotFound("post not found")
		}
		return kerr.WrapInternal("could not delete post", err)
	}
	return nil
}

// ---- Public Feed ----------------------------------------------------------

// FeedResult is the paginated public feed.
type FeedResult struct {
	Items []domain.FeedItem `json:"items"`
	Total int               `json:"total"`
}

// ListFeed returns the public journey feed.
func (s *Service) ListFeed(ctx context.Context, filter domain.FeedFilter) (*FeedResult, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	items, total, err := s.deps.Posts.ListFeed(ctx, filter)
	if err != nil {
		return nil, kerr.WrapInternal("could not list feed", err)
	}
	return &FeedResult{Items: items, Total: total}, nil
}

// GetFeedItem returns a single journey post for the public feed.
func (s *Service) GetFeedItem(ctx context.Context, id ids.ID) (*domain.FeedDetail, error) {
	d, err := s.deps.Posts.GetFeedItem(ctx, id)
	if err != nil {
		if _, ok := err.(domain.JourneyPostNotFound); ok {
			return nil, kerr.NotFound("post not found")
		}
		return nil, kerr.WrapInternal("could not load post", err)
	}
	return d, nil
}

// CountVerifiedMountains returns how many distinct mountains the user has verified hikes for.
func (s *Service) CountVerifiedMountains(ctx context.Context, userID ids.ID) (int, error) {
	n, err := s.deps.Hikes.CountVerifiedDistinctMountains(ctx, userID)
	if err != nil {
		return 0, kerr.WrapInternal("could not count mountains", err)
	}
	return n, nil
}
