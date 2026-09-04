package infrastructure

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/shared/ids"
)

// SessionLookupAdapter adapts the identity infrastructure (pgx repos) into
// the platform http.SessionLookup interface.  The composition root creates
// one and passes it to http.New.
type SessionLookupAdapter struct {
	pool *pgxpool.Pool
}

func NewSessionLookupAdapter(pool *pgxpool.Pool) *SessionLookupAdapter {
	return &SessionLookupAdapter{pool: pool}
}

// FindUserByTokenHash loads the user for an active, non-revoked session.
// Returns (nil, nil) when the token is absent, expired, revoked, or the user
// is not active.
func (a *SessionLookupAdapter) FindUserByTokenHash(ctx context.Context, tokenHash string) (*http.SessionUser, error) {
	var (
		userID string
		role   string
		status string
	)
	err := a.pool.QueryRow(ctx,
		`SELECT u.id::text, u.role, u.status
		   FROM sessions s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.token_hash = $1
		    AND s.revoked_at IS NULL
		    AND s.expires_at > now()`,
		tokenHash,
	).Scan(&userID, &role, &status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &http.SessionUser{
		ID:     ids.ID(userID),
		Role:   role,
		Status: status,
	}, nil
}

var _ http.SessionLookup = (*SessionLookupAdapter)(nil)
