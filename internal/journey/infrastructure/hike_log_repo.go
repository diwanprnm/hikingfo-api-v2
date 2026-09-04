package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/journey/domain"
	"hikingfo/backend/internal/shared/ids"
)

// HikeLogRepository is the pgx adapter for domain.HikeLogRepository.
type HikeLogRepository struct{ pool *pgxpool.Pool }

// NewHikeLogRepository wires the adapter over a shared pool.
func NewHikeLogRepository(pool *pgxpool.Pool) *HikeLogRepository {
	return &HikeLogRepository{pool: pool}
}

var _ domain.HikeLogRepository = (*HikeLogRepository)(nil)

func (r *HikeLogRepository) Create(ctx context.Context, h *domain.HikeLogEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO hike_log_entries
		   (id, user_id, mountain_id, route_id, climb_date, evidence_photo_keys, status, published_post_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		string(h.ID), string(h.UserID), string(h.MountainID),
		nullableID(h.RouteID), h.ClimbDate, h.EvidencePhotoKeys,
		string(h.Status), nullableID(h.PublishedPostID),
	)
	return err
}

func (r *HikeLogRepository) FindByID(ctx context.Context, id ids.ID) (*domain.HikeLogEntry, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, mountain_id, route_id, climb_date,
		        evidence_photo_keys, status, published_post_id, created_at, updated_at
		   FROM hike_log_entries WHERE id = $1`, string(id))
	return scanHikeLog(row)
}

func (r *HikeLogRepository) ListByUser(ctx context.Context, userID ids.ID, limit, offset int) ([]domain.HikeLogEntry, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM hike_log_entries WHERE user_id = $1`, string(userID),
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, mountain_id, route_id, climb_date,
		        evidence_photo_keys, status, published_post_id, created_at, updated_at
		   FROM hike_log_entries
		  WHERE user_id = $1
		  ORDER BY climb_date DESC
		  LIMIT $2 OFFSET $3`,
		string(userID), limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.HikeLogEntry
	for rows.Next() {
		h, err := scanHikeLog(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *h)
	}
	return out, total, rows.Err()
}

func (r *HikeLogRepository) Delete(ctx context.Context, id, userID ids.ID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM hike_log_entries WHERE id = $1 AND user_id = $2`,
		string(id), string(userID))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.HikeLogNotFound{}
	}
	return nil
}

func (r *HikeLogRepository) CountVerifiedDistinctMountains(ctx context.Context, userID ids.ID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT count(DISTINCT mountain_id) FROM hike_log_entries
		  WHERE user_id = $1 AND status = 'verified'`, string(userID),
	).Scan(&count)
	return count, err
}

// ---- scan helpers ---------------------------------------------------------

// scannable is satisfied by both pgx.Row (QueryRow) and *pgx.Rows (Query).
type scannable interface {
	Scan(dest ...any) error
}

func scanHikeLog(row scannable) (*domain.HikeLogEntry, error) {
	var h domain.HikeLogEntry
	var routeID, postID *string
	var status string
	if err := row.Scan(
		&h.ID, &h.UserID, &h.MountainID, &routeID, &h.ClimbDate,
		&h.EvidencePhotoKeys, &status, &postID, &h.CreatedAt, &h.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.HikeLogNotFound{}
		}
		return nil, err
	}
	h.Status = domain.HikeStatus(status)
	if routeID != nil {
		id := ids.ID(*routeID)
		h.RouteID = &id
	}
	if postID != nil {
		id := ids.ID(*postID)
		h.PublishedPostID = &id
	}
	return &h, nil
}

func nullableID(id *ids.ID) any {
	if id == nil {
		return nil
	}
	return string(*id)
}
