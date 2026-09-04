package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/achievement/domain"
	"hikingfo/backend/internal/shared/ids"
)

// JourneyQuery is the pgx adapter for domain.JourneyQueryPort.
// It reads directly from hike_log_entries (cross-context READ is allowed).
type JourneyQuery struct{ pool *pgxpool.Pool }

// NewJourneyQuery wires the adapter over a shared pool.
func NewJourneyQuery(pool *pgxpool.Pool) *JourneyQuery {
	return &JourneyQuery{pool: pool}
}

var _ domain.JourneyQueryPort = (*JourneyQuery)(nil)

func (q *JourneyQuery) CountVerifiedDistinctMountains(ctx context.Context, userID ids.ID) (int, error) {
	var count int
	err := q.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT mountain_id)
		   FROM hike_log_entries
		  WHERE user_id = $1
		    AND status = 'verified'`, string(userID)).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
