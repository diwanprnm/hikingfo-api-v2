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

// RequestRepository is the pgx adapter for domain.RequestRepository.
type RequestRepository struct{ pool *pgxpool.Pool }

// NewRequestRepository wires the adapter over a shared pool.
func NewRequestRepository(pool *pgxpool.Pool) *RequestRepository {
	return &RequestRepository{pool: pool}
}

var _ domain.RequestRepository = (*RequestRepository)(nil)

func (r *RequestRepository) Create(ctx context.Context, req *domain.PartnerRequest) error {
	var noticeID any
	if req.NoticeID != nil {
		noticeID = string(*req.NoticeID)
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO partner_requests (id, from_user_id, to_user_id, notice_id, mountain_id, trip_start, trip_end, message, status, matched_at, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		string(req.ID), string(req.FromUserID), string(req.ToUserID),
		noticeID, string(req.MountainID),
		req.TripStart, req.TripEnd, req.Message,
		string(req.Status), req.MatchedAt, req.CreatedAt, req.ExpiresAt,
	)
	return err
}

func (r *RequestRepository) FindByID(ctx context.Context, id ids.ID) (*domain.PartnerRequest, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, from_user_id, to_user_id, notice_id, mountain_id, trip_start, trip_end, message, status, matched_at, created_at, expires_at
		   FROM partner_requests WHERE id = $1`, string(id))
	return scanRequest(row)
}

func (r *RequestRepository) ListByUser(ctx context.Context, userID ids.ID, filter domain.RequestFilter) ([]domain.PartnerRequest, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	where := []string{"(from_user_id = $1 OR to_user_id = $1)"}
	args := []any{string(userID)}
	argN := 2

	if filter.Direction == "sent" {
		where = append(where, fmt.Sprintf("from_user_id = $%d", argN))
		args = append(args, string(userID))
		argN++
	} else if filter.Direction == "received" {
		where = append(where, fmt.Sprintf("to_user_id = $%d", argN))
		args = append(args, string(userID))
		argN++
	}

	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d::partner_request_status", argN))
		args = append(args, string(*filter.Status))
		argN++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQ := fmt.Sprintf("SELECT count(*) FROM partner_requests WHERE %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`SELECT id, from_user_id, to_user_id, notice_id, mountain_id, trip_start, trip_end, message, status, matched_at, created_at, expires_at
		FROM partner_requests WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		whereClause, argN, argN+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.PartnerRequest
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *req)
	}
	return out, total, rows.Err()
}

func (r *RequestRepository) UpdateStatus(ctx context.Context, id ids.ID, status domain.RequestStatus, matchedAt *time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE partner_requests SET status = $1, matched_at = $2 WHERE id = $3`,
		string(status), matchedAt, string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.RequestNotFound{}
	}
	return nil
}

func (r *RequestRepository) Withdraw(ctx context.Context, id, userID ids.ID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE partner_requests SET status = 'withdrawn' WHERE id = $1 AND from_user_id = $2 AND status = 'pending'`,
		string(id), string(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.RequestNotFound{}
	}
	return nil
}

func (r *RequestRepository) HasAcceptedMatch(ctx context.Context, userA, userB, mountainID ids.ID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM partner_requests
		  WHERE status = 'accepted'
		    AND mountain_id = $3
		    AND ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))`,
		string(userA), string(userB), string(mountainID)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---- scan helpers ---------------------------------------------------------

func scanRequest(row scannable) (*domain.PartnerRequest, error) {
	var req domain.PartnerRequest
	var status string
	var noticeID *string
	if err := row.Scan(
		&req.ID, &req.FromUserID, &req.ToUserID, &noticeID, &req.MountainID,
		&req.TripStart, &req.TripEnd, &req.Message,
		&status, &req.MatchedAt, &req.CreatedAt, &req.ExpiresAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.RequestNotFound{}
		}
		return nil, err
	}
	req.Status = domain.RequestStatus(status)
	if noticeID != nil {
		id := ids.ID(*noticeID)
		req.NoticeID = &id
	}
	return &req, nil
}

