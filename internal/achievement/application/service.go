// Package application holds the achievement bounded context's use-cases.
// Pure Go — depends only on achievement/domain and the shared kernel.
package application

import (
	"context"
	"time"

	"hikingfo/backend/internal/achievement/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// Dependencies the achievement use-cases need.
type Dependencies struct {
	Badges   domain.BadgeConfigRepository
	Journeys domain.JourneyQueryPort
	// Now returns the current time (injectable for tests).
	Now func() time.Time
}

// Service exposes the achievement use-cases to the interfaces layer.
type Service struct{ deps Dependencies }

// New wires a Service.
func New(deps Dependencies) *Service {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &Service{deps: deps}
}

// GetBadges assembles the user's badge view: distinct-mountain count,
// experience level, and all badges with earned/unearned status.
func (s *Service) GetBadges(ctx context.Context, userID ids.ID) (*domain.UserBadgesView, error) {
	count, err := s.deps.Journeys.CountVerifiedDistinctMountains(ctx, userID)
	if err != nil {
		return nil, domain.JourneyQueryError(err)
	}

	badges, err := s.deps.Badges.Active(ctx)
	if err != nil {
		return nil, domain.BadgeConfigError(err)
	}

	awards := make([]domain.BadgeAward, len(badges))
	for i, bc := range badges {
		awards[i] = domain.BadgeAward{
			BadgeConfig: bc,
			Earned:      count >= bc.Threshold,
		}
	}

	return &domain.UserBadgesView{
		DistinctMountains: count,
		ExperienceLevel:   deriveExperienceLevel(count),
		Badges:            awards,
	}, nil
}

// deriveExperienceLevel buckets the verified distinct-mountain count into one
// of four bilingual experience levels.
func deriveExperienceLevel(count int) domain.ExperienceLevel {
	switch {
	case count >= 35:
		return domain.ExperienceLevel{
			Key:   "ahli",
			Name:  langtext.From("Ahli", "Expert"),
			Range: "35+",
		}
	case count >= 15:
		return domain.ExperienceLevel{
			Key:   "lanjut",
			Name:  langtext.From("Lanjut", "Advanced"),
			Range: "15-34",
		}
	case count >= 5:
		return domain.ExperienceLevel{
			Key:   "menengah",
			Name:  langtext.From("Menengah", "Intermediate"),
			Range: "5-14",
		}
	default:
		return domain.ExperienceLevel{
			Key:   "pemula",
			Name:  langtext.From("Pemula", "Beginner"),
			Range: "0-4",
		}
	}
}
