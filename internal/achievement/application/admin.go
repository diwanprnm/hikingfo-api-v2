// Admin use-cases for badge configs (contracts §9 → GET/POST/PATCH
// /admin/badge-configs).
package application

import (
	"context"

	"hikingfo/backend/internal/achievement/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	"hikingfo/backend/internal/shared/langtext"
)

// AdminAll returns every badge config, active or not.
func (s *Service) AdminAll(ctx context.Context) ([]domain.BadgeConfig, error) {
	admin, ok := s.deps.Badges.(domain.BadgeConfigAdmin)
	if !ok {
		return nil, kerr.Internal("badge admin not wired")
	}
	items, err := admin.All(ctx)
	if err != nil {
		return nil, kerr.WrapInternal("could not list badge configs", err)
	}
	return items, nil
}

// BadgeConfigInput is the create/update body for a badge config.
type BadgeConfigInput struct {
	Key         string  `json:"key" binding:"required"`
	Threshold   int     `json:"threshold" binding:"required,min=1"`
	NameID      string  `json:"name_id" binding:"required"`
	NameEN      string  `json:"name_en"`
	DescID      string  `json:"description_id"`
	DescEN      string  `json:"description_en"`
	IconKey     string  `json:"icon_key"`
	Design      string  `json:"design"`
	SortOrder   int     `json:"sort_order"`
	Active      *bool   `json:"active"`
	ID          *ids.ID `json:"id"`
}

// AdminUpsertBadge creates or updates a badge config by key.
func (s *Service) AdminUpsertBadge(ctx context.Context, in BadgeConfigInput) (*domain.BadgeConfig, error) {
	admin, ok := s.deps.Badges.(domain.BadgeConfigAdmin)
	if !ok {
		return nil, kerr.Internal("badge admin not wired")
	}
	bc := &domain.BadgeConfig{
		Key:         in.Key,
		Threshold:   in.Threshold,
		Name:        langtext.From(in.NameID, in.NameEN),
		Description: langtext.From(in.DescID, in.DescEN),
		IconKey:     in.IconKey,
		Design:      in.Design,
		SortOrder:   in.SortOrder,
		Active:      in.Active == nil || *in.Active,
	}
	if in.ID != nil {
		bc.ID = *in.ID
	}
	if err := admin.Upsert(ctx, bc); err != nil {
		return nil, kerr.WrapInternal("could not save badge config", err)
	}
	return bc, nil
}
