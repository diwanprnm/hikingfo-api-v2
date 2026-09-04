package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/achievement/domain"
	"hikingfo/backend/internal/shared/ids"
)

// BadgeConfigRepo is the pgx adapter for domain.BadgeConfigRepository.
type BadgeConfigRepo struct{ pool *pgxpool.Pool }

// NewBadgeConfigRepo wires the adapter over a shared pool.
func NewBadgeConfigRepo(pool *pgxpool.Pool) *BadgeConfigRepo {
	return &BadgeConfigRepo{pool: pool}
}

var _ domain.BadgeConfigRepository = (*BadgeConfigRepo)(nil)

func (r *BadgeConfigRepo) Active(ctx context.Context) ([]domain.BadgeConfig, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, key, threshold, name, description, icon_key, design, sort_order, active, created_at, updated_at
		   FROM badge_configs
		  WHERE active = true
		  ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.BadgeConfig
	for rows.Next() {
		var bc domain.BadgeConfig
		var id string
		if err := rows.Scan(
			&id, &bc.Key, &bc.Threshold, &bc.Name, &bc.Description,
			&bc.IconKey, &bc.Design, &bc.SortOrder, &bc.Active,
			&bc.CreatedAt, &bc.UpdatedAt,
		); err != nil {
			if err == pgx.ErrNoRows {
				return out, nil
			}
			return nil, err
		}
		bc.ID = ids.ID(id)
		out = append(out, bc)
	}
	return out, rows.Err()
}
