package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/partner/domain"
	"hikingfo/backend/internal/shared/ids"
)

// NoticeRepository is the pgx adapter for domain.NoticeRepository.
type NoticeRepository struct{ pool *pgxpool.Pool }

// NewNoticeRepository wires the adapter over a shared pool.
func NewNoticeRepository(pool *pgxpool.Pool) *NoticeRepository {
	return &NoticeRepository{pool: pool}
}

var _ domain.NoticeRepository = (*NoticeRepository)(nil)

func (r *NoticeRepository) Create(ctx context.Context, n *domain.PartnerNotice) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO partner_notices (id, user_id, mountain_id, trip_start, trip_end, note, status, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		string(n.ID), string(n.UserID), string(n.MountainID),
		n.TripStart, n.TripEnd, n.Note,
		string(n.Status), n.ExpiresAt, n.CreatedAt, n.UpdatedAt,
	)
	return err
}

func (r *NoticeRepository) FindByID(ctx context.Context, id ids.ID) (*domain.PartnerNotice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, mountain_id, trip_start, trip_end, note, status, expires_at, created_at, updated_at
		   FROM partner_notices WHERE id = $1`, string(id))
	return scanNotice(row)
}

func (r *NoticeRepository) ListByUser(ctx context.Context, userID ids.ID, filter domain.NoticeFilter) ([]domain.PartnerNotice, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	where := []string{"user_id = $1"}
	args := []any{string(userID)}
	argN := 2

	if filter.MountainID != nil {
		where = append(where, fmt.Sprintf("mountain_id = $%d", argN))
		args = append(args, string(*filter.MountainID))
		argN++
	}
	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d::partner_notice_status", argN))
		args = append(args, string(*filter.Status))
		argN++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQ := fmt.Sprintf("SELECT count(*) FROM partner_notices WHERE %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`SELECT id, user_id, mountain_id, trip_start, trip_end, note, status, expires_at, created_at, updated_at
		FROM partner_notices WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		whereClause, argN, argN+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.PartnerNotice
	for rows.Next() {
		n, err := scanNotice(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *n)
	}
	return out, total, rows.Err()
}

func (r *NoticeRepository) ListOpen(ctx context.Context, mountainID ids.ID, tripStart, tripEnd time.Time, limit, offset int) ([]domain.PartnerNotice, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// Auto-expire: only return open notices with non-expired trips that overlap.
	where := `status = 'open' AND expires_at > now() AND mountain_id = $1
	          AND NOT (trip_end < $2 OR trip_start > $3)`
	args := []any{string(mountainID), tripStart, tripEnd}

	var total int
	countQ := fmt.Sprintf("SELECT count(*) FROM partner_notices WHERE %s", where)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`SELECT id, user_id, mountain_id, trip_start, trip_end, note, status, expires_at, created_at, updated_at
		FROM partner_notices WHERE %s ORDER BY trip_start ASC LIMIT $4 OFFSET $5`, where)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.PartnerNotice
	for rows.Next() {
		n, err := scanNotice(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *n)
	}
	return out, total, rows.Err()
}

func (r *NoticeRepository) UpdateStatus(ctx context.Context, id ids.ID, status domain.NoticeStatus) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE partner_notices SET status = $1 WHERE id = $2`,
		string(status), string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NoticeNotFound{}
	}
	return nil
}

func (r *NoticeRepository) Withdraw(ctx context.Context, id, userID ids.ID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE partner_notices SET status = 'withdrawn' WHERE id = $1 AND user_id = $2 AND status = 'open'`,
		string(id), string(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NoticeNotFound{}
	}
	return nil
}

// ---- scan helpers ---------------------------------------------------------

// scannable is satisfied by both pgx.Row and *pgx.Rows.
type scannable interface {
	Scan(dest ...any) error
}

func scanNotice(row scannable) (*domain.PartnerNotice, error) {
	var n domain.PartnerNotice
	var status string
	if err := row.Scan(
		&n.ID, &n.UserID, &n.MountainID,
		&n.TripStart, &n.TripEnd, &n.Note,
		&status, &n.ExpiresAt, &n.CreatedAt, &n.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.NoticeNotFound{}
		}
		return nil, err
	}
	n.Status = domain.NoticeStatus(status)
	return &n, nil
}
