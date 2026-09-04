// Package domain holds the achievement bounded context's entities, value objects
// and ports (plan.md → DDD). Pure Go — no framework, no DB driver.
package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// BadgeConfig is the admin-configurable badge definition stored in badge_configs.
type BadgeConfig struct {
	ID          ids.ID
	Key         string
	Threshold   int
	Name        langtext.Text
	Description langtext.Text
	IconKey     string
	Design      string
	SortOrder   int
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BadgeAward is the derived earned-badge view returned to clients.
type BadgeAward struct {
	BadgeConfig
	Earned bool `json:"earned"`
}

// ExperienceLevel is the hiking experience bucket derived from verified distinct-mountain count.
type ExperienceLevel struct {
	Key   string         `json:"key"`
	Name  langtext.Text  `json:"name"`
	Range string         `json:"range"`
}

// UserBadgesView is the full response assembled by the application service.
type UserBadgesView struct {
	DistinctMountains int              `json:"distinct_mountains"`
	ExperienceLevel   ExperienceLevel  `json:"experience_level"`
	Badges            []BadgeAward     `json:"badges"`
}

// BadgeConfigRepository is the persistence port for badge_configs.
type BadgeConfigRepository interface {
	// Active returns all active badge configs ordered by sort_order.
	Active(ctx context.Context) ([]BadgeConfig, error)
}

// JourneyQueryPort reads journey data (cross-context READ from hike_log_entries).
type JourneyQueryPort interface {
	// CountVerifiedDistinctMountains returns the count of distinct mountains
	// a user has verifiedly hiked.
	CountVerifiedDistinctMountains(ctx context.Context, userID ids.ID) (int, error)
}
