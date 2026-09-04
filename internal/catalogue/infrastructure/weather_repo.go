package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/catalogue/domain"
	"hikingfo/backend/internal/shared/ids"
)

// WeatherRepository is the pgx adapter for domain.WeatherStore.
type WeatherRepository struct{ pool *pgxpool.Pool }

// NewWeatherRepository wires the adapter over a shared pool.
func NewWeatherRepository(pool *pgxpool.Pool) *WeatherRepository {
	return &WeatherRepository{pool: pool}
}

var _ domain.WeatherStore = (*WeatherRepository)(nil)

func (r *WeatherRepository) Get(ctx context.Context, mountainID ids.ID) (*domain.WeatherSnapshot, error) {
	var s domain.WeatherSnapshot
	var payload []byte
	err := r.pool.QueryRow(ctx,
		`SELECT mountain_id, payload, captured_at, is_live, expires_at
		   FROM mountain_weather
		  WHERE mountain_id = $1`, string(mountainID),
	).Scan(&s.MountainID, &payload, &s.CapturedAt, &s.IsLive, &s.ExpiresAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s.Payload = make(map[string]any)
	_ = json.Unmarshal(payload, &s.Payload)
	return &s, nil
}

func (r *WeatherRepository) Upsert(ctx context.Context, s *domain.WeatherSnapshot) error {
	payload, err := json.Marshal(s.Payload)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO mountain_weather (mountain_id, payload, captured_at, is_live, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (mountain_id) DO UPDATE
		   SET payload = EXCLUDED.payload, captured_at = EXCLUDED.captured_at,
		       is_live = EXCLUDED.is_live, expires_at = EXCLUDED.expires_at`,
		s.MountainID, payload, s.CapturedAt, s.IsLive, s.ExpiresAt,
	)
	return err
}
