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

// All returns every badge config, active or not (admin path).
func (r *BadgeConfigRepo) All(ctx context.Context) ([]domain.BadgeConfig, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, key, threshold, name, description, icon_key, design, sort_order, active, created_at, updated_at
		   FROM badge_configs ORDER BY sort_order`)
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
			return nil, err
		}
		bc.ID = ids.ID(id)
		out = append(out, bc)
	}
	return out, rows.Err()
}

// Upsert inserts or updates a badge config by key (admin path).
func (r *BadgeConfigRepo) Upsert(ctx context.Context, bc *domain.BadgeConfig) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO badge_configs (id, key, threshold, name, description, icon_key, design, sort_order, active)
		VALUES (COALESCE($1::uuid, gen_random_uuid()), $2, $3, $4::jsonb, $5::jsonb, $6, $7, $8, $9)
		ON CONFLICT (key) DO UPDATE SET
			threshold = EXCLUDED.threshold,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			icon_key = EXCLUDED.icon_key,
			design = EXCLUDED.design,
			sort_order = EXCLUDED.sort_order,
			active = EXCLUDED.active,
			updated_at = now()
		RETURNING id, created_at, updated_at`,
		nullableUUID(bc.ID), bc.Key, bc.Threshold,
		[]byte(`{"id":"`+bc.Name.ID+`","en":"`+bc.Name.EN+`"}`), []byte(`{"id":"`+bc.Description.ID+`","en":"`+bc.Description.EN+`"}`),
		bc.IconKey, bc.Design, bc.SortOrder, bc.Active,
	).Scan(&bc.ID, &bc.CreatedAt, &bc.UpdatedAt)
}

func nullableUUID(id ids.ID) any {
	if id.IsNil() {
		return nil
	}
	return string(id)
}
