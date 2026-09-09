package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// PostVisibility controls who can see a journey post.
type PostVisibility string

const (
	VisibilityPublished PostVisibility = "published"
	VisibilityDraft     PostVisibility = "draft"
)

// ModerationStatus tracks content moderation state.
type ModerationStatus string

const (
	ModerationVisible    ModerationStatus = "visible"
	ModerationUnderReview ModerationStatus = "under_review"
	ModerationHidden     ModerationStatus = "hidden"
)

// JourneyPost is a user-authored narrative about a hike (data-model → journey_posts).
type JourneyPost struct {
	ID               ids.ID
	UserID           ids.ID
	MountainID       ids.ID
	RouteID          *ids.ID          // nullable
	Title            string
	Summary          map[string]any   // structured: dates, cost, weather, timing
	Narrative        langtext.Text    // ID-only in v1
	PhotoKeys        []string         // ~10 max
	Visibility       PostVisibility
	ModerationStatus ModerationStatus
	PublishedAt      *time.Time       // nullable, set on publish
	UpdatedAt        time.Time
}

// FeedItem is the lightweight projection for the public feed listing.
type FeedItem struct {
	ID            ids.ID            `json:"id"`
	Title         string            `json:"title"`
	Summary       map[string]any    `json:"summary"`
	Mountain      FeedMountain      `json:"mountain"`
	Author        FeedAuthor        `json:"author"`
	PublishedAt   *time.Time        `json:"published_at,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// FeedMountain carries the mountain subset shown in feed cards.
type FeedMountain struct {
	ID     ids.ID `json:"id"`
	Name   langtext.Text `json:"name"`
	Slug   string `json:"slug"`
	Region string `json:"region"`
}

// FeedAuthor carries the author subset shown in feed cards.
type FeedAuthor struct {
	ID          ids.ID  `json:"id"`
	DisplayName string  `json:"display_name"`
	AvatarURL   string  `json:"avatar_url,omitempty"`
}

// FeedDetail is the full journey view for a single post.
type FeedDetail struct {
	JourneyPost
	Mountain FeedMountain `json:"mountain"`
	Author   FeedAuthor   `json:"author"`
}

// FeedFilter carries query parameters for the public feed.
type FeedFilter struct {
	MountainID ids.ID
	Region     string
	Page       int
	PageSize   int
}

// JourneyPostRepository is the persistence port for journey posts.
type JourneyPostRepository interface {
	Create(ctx context.Context, p *JourneyPost) error
	FindByID(ctx context.Context, id ids.ID) (*JourneyPost, error)
	Update(ctx context.Context, p *JourneyPost) error
	Delete(ctx context.Context, id, userID ids.ID) error
	Publish(ctx context.Context, id, userID ids.ID) error
	ListFeed(ctx context.Context, filter FeedFilter) ([]FeedItem, int, error)
	GetFeedItem(ctx context.Context, id ids.ID) (*FeedDetail, error)
	// SetModerationStatus flips moderation_status directly — admin path, no
	// ownership check (contracts §9 → resolve report → hide_content).
	SetModerationStatus(ctx context.Context, id ids.ID, status ModerationStatus) error
}
