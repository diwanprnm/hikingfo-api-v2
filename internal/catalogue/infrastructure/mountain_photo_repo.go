package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/catalogue/domain"
	"hikingfo/backend/internal/shared/ids"
)

// GalleryRepository is the pgx adapter for domain.GalleryRepository (004).
type GalleryRepository struct{ pool *pgxpool.Pool }

// NewGalleryRepository wires the adapter over a shared pool.
func NewGalleryRepository(pool *pgxpool.Pool) *GalleryRepository {
	return &GalleryRepository{pool: pool}
}

var _ domain.GalleryRepository = (*GalleryRepository)(nil)

func (r *GalleryRepository) ListByMountain(ctx context.Context, mountainID ids.ID) ([]domain.MountainPhoto, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, photo_key, created_at FROM mountain_photos
		  WHERE mountain_id = $1 ORDER BY created_at`, string(mountainID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.MountainPhoto{}
	for rows.Next() {
		var p domain.MountainPhoto
		var id, key string
		if err := rows.Scan(&id, &key, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.ID, p.PhotoKey = ids.ID(id), key
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *GalleryRepository) Insert(ctx context.Context, p *domain.MountainPhoto) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO mountain_photos (id, mountain_id, photo_key) VALUES ($1, $2, $3)`,
		string(p.ID), string(p.MountainID), p.PhotoKey)
	return err
}

func (r *GalleryRepository) Delete(ctx context.Context, photoID, mountainID ids.ID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM mountain_photos WHERE id = $1 AND mountain_id = $2`,
		string(photoID), string(mountainID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.PhotoNotFound{}
	}
	return nil
}
