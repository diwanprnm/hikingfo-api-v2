package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/notification/domain"
	"hikingfo/backend/internal/shared/ids"
)

// NotificationRepository is the pgx adapter for domain.Repository.
type NotificationRepository struct{ pool *pgxpool.Pool }

// NewNotificationRepository wires the adapter over a shared pool.
func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

var _ domain.Repository = (*NotificationRepository)(nil)

func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	payload, err := json.Marshal(n.Payload)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO notifications (id, user_id, type, payload, created_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5)`,
		string(n.ID), string(n.UserID), string(n.Type), payload, n.CreatedAt,
	)
	return err
}

func (r *NotificationRepository) ListByUser(ctx context.Context, userID ids.ID, limit, offset int) ([]domain.Notification, int, error) {
	// Total count.
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid`, string(userID),
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, payload, read_at, created_at
		   FROM notifications
		  WHERE user_id = $1::uuid
		  ORDER BY created_at DESC
		  LIMIT $2 OFFSET $3`,
		string(userID), limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.Notification
	for rows.Next() {
		var n domain.Notification
		var id, uid string
		var ntype string
		var payload []byte
		if err := rows.Scan(&id, &uid, &ntype, &payload, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		n.ID = ids.ID(id)
		n.UserID = ids.ID(uid)
		n.Type = domain.Type(ntype)
		n.Payload = make(map[string]any)
		_ = json.Unmarshal(payload, &n.Payload)
		out = append(out, n)
	}
	return out, total, rows.Err()
}

func (r *NotificationRepository) MarkRead(ctx context.Context, userID, notificationID ids.ID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = now()
		  WHERE id = $1::uuid AND user_id = $2::uuid AND read_at IS NULL`,
		string(notificationID), string(userID),
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound{}
	}
	return nil
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, userID ids.ID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications
		  WHERE user_id = $1::uuid AND read_at IS NULL`, string(userID),
	).Scan(&count)
	return count, err
}

// errNotFound is returned when the target notification does not exist or is
// not owned by the user.
type errNotFound struct{}

func (errNotFound) Error() string { return "notification: not found" }

// Silence unused import.
var _ = errors.New
