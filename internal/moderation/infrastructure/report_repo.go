// Package infrastructure holds the pgx adapters for the moderation context.
package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/moderation/domain"
	"hikingfo/backend/internal/shared/ids"
)

// ReportRepository implements moderation/domain.Repository over pgx.
type ReportRepository struct{ pool *pgxpool.Pool }

// NewReportRepository wires the adapter over a shared pool.
func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

var _ domain.Repository = (*ReportRepository)(nil)

func (r *ReportRepository) Create(ctx context.Context, rep *domain.Report) error {
	var reporter any
	if rep.ReporterID != nil {
		reporter = string(*rep.ReporterID)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reports (id, reporter_id, target_type, target_id, field_ref, reason, detail, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		string(rep.ID), reporter, string(rep.TargetType), string(rep.TargetID),
		rep.FieldRef, string(rep.Reason), rep.Detail, string(rep.Status))
	return err
}

func (r *ReportRepository) List(ctx context.Context, status domain.Status, limit int) ([]domain.Report, error) {
	q := `SELECT id, reporter_id, target_type::text, target_id, field_ref, reason::text,
	             detail, status::text, resolution, created_at, resolved_at
	        FROM reports`
	args := []any{}
	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, string(status))
	}
	q += ` ORDER BY created_at LIMIT $` + itoa(len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Report
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rep)
	}
	return out, rows.Err()
}

func (r *ReportRepository) FindByID(ctx context.Context, id ids.ID) (*domain.Report, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, reporter_id, target_type::text, target_id, field_ref, reason::text,
		       detail, status::text, resolution, created_at, resolved_at
		  FROM reports WHERE id = $1`, string(id))
	return scanReport(row)
}

func (r *ReportRepository) UpdateStatus(ctx context.Context, id ids.ID, status domain.Status, resolution string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE reports SET status = $2, resolution = $3, resolved_at = now() WHERE id = $1`,
		string(id), string(status), resolution)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ReportRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status::text, count(*) FROM reports GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, rows.Err()
}

func scanReport(row scannable) (*domain.Report, error) {
	var rep domain.Report
	var reporter, fieldRef, detail, resolution *string
	var reason, status, targetType string
	var resolvedAt *time.Time
	if err := row.Scan(
		&rep.ID, &reporter, &targetType, &rep.TargetID, &fieldRef, &reason,
		&detail, &status, &resolution, &rep.CreatedAt, &resolvedAt,
	); err != nil {
		return nil, err
	}
	if reporter != nil {
		id := ids.ID(*reporter)
		rep.ReporterID = &id
	}
	rep.TargetType = domain.TargetType(targetType)
	rep.Reason = domain.Reason(reason)
	rep.Status = domain.Status(status)
	rep.FieldRef = deref(fieldRef)
	rep.Detail = deref(detail)
	rep.Resolution = deref(resolution)
	rep.ResolvedAt = resolvedAt
	return &rep, nil
}

// scannable is satisfied by both pgx.Row and *pgx.Rows.
type scannable interface {
	Scan(dest ...any) error
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// BlockRepository implements moderation/domain.BlockRepository over the
// blocks table (0001_foundation.sql).
type BlockRepository struct{ pool *pgxpool.Pool }

// NewBlockRepository wires the adapter over a shared pool.
func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{pool: pool}
}

var _ domain.BlockRepository = (*BlockRepository)(nil)

func (b *BlockRepository) Insert(ctx context.Context, blocker, blocked ids.ID) error {
	_, err := b.pool.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING`,
		string(blocker), string(blocked))
	return err
}

func (b *BlockRepository) Delete(ctx context.Context, blocker, blocked ids.ID) error {
	_, err := b.pool.Exec(ctx,
		`DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`,
		string(blocker), string(blocked))
	return err
}

func (b *BlockRepository) Exists(ctx context.Context, blocker, blocked ids.ID) (bool, error) {
	var ok bool
	err := b.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker_id = $1 AND blocked_id = $2)`,
		string(blocker), string(blocked)).Scan(&ok)
	return ok, err
}

// owners implements the composition-root owner lookups over raw SQL (the
// moderation context needs "who owns this content" without importing journey).
type owners struct{ pool *pgxpool.Pool }

// NewOwners wires the owner-lookup adapter.
func NewOwners(pool *pgxpool.Pool) *owners { return &owners{pool: pool} }

// JourneyOwner returns the author of a journey post.
func (o *owners) JourneyOwner(ctx context.Context, postID ids.ID) (ids.ID, bool) {
	var uid string
	err := o.pool.QueryRow(ctx,
		`SELECT user_id FROM journey_posts WHERE id = $1`, string(postID)).Scan(&uid)
	if err != nil {
		return ids.Nil, false
	}
	return ids.ID(uid), true
}

// HikeOwner returns the hiker of a hike log entry.
func (o *owners) HikeOwner(ctx context.Context, hikeID ids.ID) (ids.ID, bool) {
	var uid string
	err := o.pool.QueryRow(ctx,
		`SELECT user_id FROM hike_log_entries WHERE id = $1`, string(hikeID)).Scan(&uid)
	if err != nil {
		return ids.Nil, false
	}
	return ids.ID(uid), true
}
