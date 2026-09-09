package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/catalogue/domain"
	"hikingfo/backend/internal/shared/ids"
)

// MountainRepository is the pgx adapter for domain.MountainRepository.
type MountainRepository struct{ pool *pgxpool.Pool }

// NewMountainRepository wires the adapter over a shared pool.
func NewMountainRepository(pool *pgxpool.Pool) *MountainRepository {
	return &MountainRepository{pool: pool}
}

var _ domain.MountainRepository = (*MountainRepository)(nil)

func (r *MountainRepository) SearchAndFilter(ctx context.Context, f domain.SearchFilter) ([]domain.Mountain, int, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	where := []string{"m.status = 'published'"}
	args := []any{}
	argN := 1

	if f.Query != "" {
		where = append(where, fmt.Sprintf("m.search_text ILIKE $%d", argN))
		args = append(args, "%"+strings.ToLower(f.Query)+"%")
		argN++
	}
	if f.Region != "" {
		where = append(where, fmt.Sprintf("m.region = $%d::mountain_region", argN))
		args = append(args, string(f.Region))
		argN++
	}
	if f.Province != "" {
		where = append(where, fmt.Sprintf("m.province ILIKE $%d", argN))
		args = append(args, "%"+f.Province+"%")
		argN++
	}
	if f.Difficulty > 0 {
		where = append(where, fmt.Sprintf("m.difficulty = $%d", argN))
		args = append(args, f.Difficulty)
		argN++
	}
	if f.MinHeight > 0 {
		where = append(where, fmt.Sprintf("m.peak_height_m >= $%d", argN))
		args = append(args, f.MinHeight)
		argN++
	}
	if f.MaxHeight > 0 {
		where = append(where, fmt.Sprintf("m.peak_height_m <= $%d", argN))
		args = append(args, f.MaxHeight)
		argN++
	}

	whereClause := strings.Join(where, " AND ")

	// Total count
	var total int
	countQ := fmt.Sprintf("SELECT count(*) FROM mountains m WHERE %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	q := fmt.Sprintf(`
		SELECT m.id, m.slug, m.name, m.aliases, m.region, m.province,
		       m.location, m.latitude, m.longitude, m.peak_name, m.peak_height_m,
		       m.difficulty, m.status, m.data_meta, m.photo_key, m.created_at, m.updated_at
		  FROM mountains m
		 WHERE %s
		 ORDER BY m.peak_height_m DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, argN, argN+1)
	args = append(args, f.PageSize, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.Mountain
	for rows.Next() {
		m, err := scanMountain(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *m)
	}
	return out, total, rows.Err()
}

func (r *MountainRepository) FindBySlug(ctx context.Context, slug string) (*domain.Mountain, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT m.id, m.slug, m.name, m.aliases, m.region, m.province,
		        m.location, m.latitude, m.longitude, m.peak_name, m.peak_height_m,
		        m.difficulty, m.status, m.data_meta, m.photo_key, m.created_at, m.updated_at
		   FROM mountains m
		  WHERE m.slug = $1`, slug)
	return scanMountain(row)
}

func (r *MountainRepository) FindByID(ctx context.Context, id ids.ID) (*domain.Mountain, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT m.id, m.slug, m.name, m.aliases, m.region, m.province,
		        m.location, m.latitude, m.longitude, m.peak_name, m.peak_height_m,
		        m.difficulty, m.status, m.data_meta, m.photo_key, m.created_at, m.updated_at
		   FROM mountains m
		  WHERE m.id = $1`, string(id))
	return scanMountain(row)
}

func (r *MountainRepository) Routes(ctx context.Context, mountainID ids.ID) ([]domain.Route, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, mountain_id, name, distance_km, duration_hours,
		        elevation_gain_m, entry_requirements, sort_order, created_at, updated_at
		   FROM mountain_routes
		  WHERE mountain_id = $1
		  ORDER BY sort_order`, string(mountainID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Route
	for rows.Next() {
		var rt domain.Route
		var mid string
		if err := rows.Scan(
			&rt.ID, &mid, &rt.Name, &rt.DistanceKm, &rt.DurationHours,
			&rt.ElevationGainM, &rt.EntryRequirement, &rt.SortOrder,
			&rt.CreatedAt, &rt.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rt.MountainID = ids.ID(mid)
		out = append(out, rt)
	}
	return out, rows.Err()
}

func (r *MountainRepository) Basecamps(ctx context.Context, mountainID ids.ID) ([]domain.Basecamp, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, mountain_id, name, facilities, cost_estimate,
		        is_permit_point, latitude, longitude, sort_order
		   FROM mountain_basecamps
		  WHERE mountain_id = $1
		  ORDER BY sort_order`, string(mountainID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Basecamp
	for rows.Next() {
		var b domain.Basecamp
		var mid string
		if err := rows.Scan(
			&b.ID, &mid, &b.Name, &b.Facilities, &b.CostEstimate,
			&b.IsPermitPoint, &b.Latitude, &b.Longitude, &b.SortOrder,
		); err != nil {
			return nil, err
		}
		b.MountainID = ids.ID(mid)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *MountainRepository) Regions(ctx context.Context) ([]domain.Region, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT region FROM mountains WHERE status = 'published' ORDER BY region`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Region
	for rows.Next() {
		var reg domain.Region
		if err := rows.Scan(&reg); err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, rows.Err()
}

// ---- scan helpers ---------------------------------------------------------

// scannable is satisfied by both pgx.Row (QueryRow) and *pgx.Rows (Query).
type scannable interface {
	Scan(dest ...any) error
}

func scanMountain(row scannable) (*domain.Mountain, error) {
	var m domain.Mountain
	var region string
	var status string
	var dataMeta []byte
	var photoKey *string // photo_key is nullable; "" when absent
	if err := row.Scan(
		&m.ID, &m.Slug, &m.Name, &m.Aliases, &region, &m.Province,
		&m.Location, &m.Latitude, &m.Longitude, &m.PeakName, &m.PeakHeightM,
		&m.Difficulty, &status, &dataMeta, &photoKey, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.MountainNotFound{}
		}
		return nil, err
	}
	if photoKey != nil {
		m.PhotoKey = *photoKey
	}
	m.Region = domain.Region(region)
	m.Status = domain.PublishStatus(status)
	m.DataMeta = make(map[string]domain.FieldMeta)
	_ = json.Unmarshal(dataMeta, &m.DataMeta)
	return &m, nil
}
