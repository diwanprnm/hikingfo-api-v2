package infrastructure

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/catalogue/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// MountainWriter implements domain.MountainWriter over pgx. Every mutation
// runs in one transaction; Update/Rollback snapshot the prior row into
// mountain_revisions first (contracts §9 → versioning).
type MountainWriter struct{ pool *pgxpool.Pool }

// NewMountainWriter wires the admin adapter over a shared pool.
func NewMountainWriter(pool *pgxpool.Pool) *MountainWriter {
	return &MountainWriter{pool: pool}
}

var _ domain.MountainWriter = (*MountainWriter)(nil)

// ---- mountains -------------------------------------------------------------

const mountainCols = `slug, name, aliases, region, province, location,
	latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text`

func searchText(m *domain.Mountain) string {
	parts := []string{strings.ToLower(m.Name.ID), strings.ToLower(m.Name.EN)}
	for _, a := range m.Aliases {
		parts = append(parts, strings.ToLower(a))
	}
	return strings.Join(parts, " ")
}

func marshalText(t langtext.Text) []byte {
	b, _ := json.Marshal(t)
	return b
}

// nilStrings maps a nil slice to an empty slice (aliases NOT NULL).
func nilStrings(s []string) any {
	if s == nil {
		return []string{}
	}
	return s
}

func marshalMeta(meta map[string]domain.FieldMeta) []byte {
	if meta == nil {
		meta = map[string]domain.FieldMeta{}
	}
	b, _ := json.Marshal(meta)
	return b
}

func (w *MountainWriter) Create(ctx context.Context, m *domain.Mountain) error {
	if m.ID.IsNil() {
		m.ID = ids.New()
	}
	if m.Status == "" {
		m.Status = domain.StatusDraft
	}
	_, err := w.pool.Exec(ctx, `
		INSERT INTO mountains (id, slug, name, aliases, region, province, location,
			latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
		VALUES ($1,$2,$3::jsonb,$4,$5::mountain_region,$6,$7::jsonb,$8,$9,$10::jsonb,$11,$12,$13::mountain_status,$14::jsonb,$15)`,
		string(m.ID), m.Slug, marshalText(m.Name), nilStrings(m.Aliases), string(m.Region), m.Province,
		marshalText(m.Location), m.Latitude, m.Longitude, marshalText(m.PeakName),
		m.PeakHeightM, m.Difficulty, string(m.Status), marshalMeta(m.DataMeta), searchText(m))
	if err != nil {
		return mapWriteErr(err, "mountain")
	}
	return nil
}

// snapshot loads the mutable columns of a mountain as a JSON-ready map.
func snapshotMountain(ctx context.Context, tx pgx.Tx, id ids.ID) (map[string]any, error) {
	var name, loc, peak, meta []byte
	var slug, province, region, status string
	var lat, lng float64
	var height, diff int
	var aliases []string
	var photoKey *string
	err := tx.QueryRow(ctx, `
		SELECT slug, name, aliases, region::text, province, location,
		       latitude, longitude, peak_name, peak_height_m, difficulty,
		       status::text, data_meta, photo_key
		  FROM mountains WHERE id = $1`, string(id)).
		Scan(&slug, &name, &aliases, &region, &province, &loc,
			&lat, &lng, &peak, &height, &diff, &status, &meta, &photoKey)
	if err != nil {
		return nil, err
	}
	aliasesJSON, _ := json.Marshal(aliases)
	pk := ""
	if photoKey != nil {
		pk = *photoKey
	}
	snap := map[string]any{
		"slug":         slug,
		"name":         json.RawMessage(name),
		"aliases":      json.RawMessage(aliasesJSON),
		"region":       region,
		"province":     province,
		"location":     json.RawMessage(loc),
		"latitude":     lat,
		"longitude":    lng,
		"peak_name":    json.RawMessage(peak),
		"peak_height_m": height,
		"difficulty":   diff,
		"status":       status,
		"data_meta":    json.RawMessage(meta),
		"photo_key":    pk,
	}
	return snap, nil
}

func (w *MountainWriter) Update(ctx context.Context, m *domain.Mountain, revisedBy *ids.ID, reason string) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below

	snap, err := snapshotMountain(ctx, tx, m.ID)
	if err != nil {
		return mapWriteErr(err, "mountain")
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO mountain_revisions (id, mountain_id, snapshot, revised_by, reason)
		VALUES ($1,$2,$3::jsonb,$4,$5)`,
		ids.New(), string(m.ID), mustJSON(snap), revisedBy, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE mountains SET slug=$2, name=$3::jsonb, aliases=$4, region=$5::mountain_region,
		       province=$6, location=$7::jsonb, latitude=$8, longitude=$9, peak_name=$10::jsonb,
		       peak_height_m=$11, difficulty=$12, status=$13::mountain_status,
		       data_meta=$14::jsonb, search_text=$15, updated_at=now()
		 WHERE id=$1`,
		string(m.ID), m.Slug, marshalText(m.Name), nilStrings(m.Aliases), string(m.Region), m.Province,
		marshalText(m.Location), m.Latitude, m.Longitude, marshalText(m.PeakName),
		m.PeakHeightM, m.Difficulty, string(m.Status), marshalMeta(m.DataMeta), searchText(m)); err != nil {
		return mapWriteErr(err, "mountain")
	}
	return tx.Commit(ctx)
}

func (w *MountainWriter) Delete(ctx context.Context, id ids.ID) error {
	tag, err := w.pool.Exec(ctx, `DELETE FROM mountains WHERE id = $1`, string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.MountainNotFound{}
	}
	return nil
}

func (w *MountainWriter) ListAll(ctx context.Context) ([]domain.Mountain, error) {
	rows, err := w.pool.Query(ctx, `
		SELECT m.id, m.slug, m.name, m.aliases, m.region, m.province,
		       m.location, m.latitude, m.longitude, m.peak_name, m.peak_height_m,
		       m.difficulty, m.status, m.data_meta, m.photo_key, m.created_at, m.updated_at
		  FROM mountains m ORDER BY m.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Mountain
	for rows.Next() {
		m, err := scanMountain(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (w *MountainWriter) Revisions(ctx context.Context, mountainID ids.ID) ([]domain.Revision, error) {
	rows, err := w.pool.Query(ctx, `
		SELECT id, mountain_id, snapshot, revised_by, reason, created_at
		  FROM mountain_revisions WHERE mountain_id = $1
		 ORDER BY created_at DESC`, string(mountainID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Revision
	for rows.Next() {
		var rev domain.Revision
		var mid string
		var snap []byte
		var by *string
		if err := rows.Scan(&rev.ID, &mid, &snap, &by, &rev.Reason, &rev.CreatedAt); err != nil {
			return nil, err
		}
		rev.MountainID = ids.ID(mid)
		if by != nil {
			b := ids.ID(*by)
			rev.RevisedBy = &b
		}
		if err := json.Unmarshal(snap, &rev.Snapshot); err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}

func (w *MountainWriter) Rollback(ctx context.Context, mountainID, revisionID ids.ID, revisedBy *ids.ID) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below

	var snapRaw []byte
	err = tx.QueryRow(ctx,
		`SELECT snapshot FROM mountain_revisions WHERE id = $1 AND mountain_id = $2`,
		string(revisionID), string(mountainID)).Scan(&snapRaw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.MountainNotFound{}
		}
		return err
	}
	var snap map[string]any
	if err := json.Unmarshal(snapRaw, &snap); err != nil {
		return err
	}

	cur, err := snapshotMountain(ctx, tx, mountainID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO mountain_revisions (id, mountain_id, snapshot, revised_by, reason)
		VALUES ($1,$2,$3::jsonb,$4,$5)`,
		ids.New(), string(mountainID), mustJSON(cur), revisedBy, "pre-rollback snapshot"); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE mountains SET slug=$2, name=$3::jsonb, aliases=$4, region=$5::mountain_region,
		       province=$6, location=$7::jsonb, latitude=$8, longitude=$9, peak_name=$10::jsonb,
		       peak_height_m=$11, difficulty=$12, status=$13::mountain_status,
		       data_meta=$14::jsonb, search_text=$15, photo_key=NULLIF($16,''), updated_at=now()
		 WHERE id=$1`,
		string(mountainID),
		snap["slug"], snap["name"], snap["aliases"], snap["region"], snap["province"],
		snap["location"], snap["latitude"], snap["longitude"], snap["peak_name"],
		snap["peak_height_m"], snap["difficulty"], snap["status"], snap["data_meta"],
		searchTextFromSnapshot(snap), snap["photo_key"]); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func searchTextFromSnapshot(snap map[string]any) string {
	parts := []string{}
	if raw, ok := snap["name"].(map[string]any); ok {
		for _, k := range []string{"id", "en"} {
			if v, ok := raw[k].(string); ok && v != "" {
				parts = append(parts, strings.ToLower(v))
			}
		}
	}
	if raw, ok := snap["aliases"].([]any); ok {
		for _, a := range raw {
			if v, ok := a.(string); ok && v != "" {
				parts = append(parts, strings.ToLower(v))
			}
		}
	}
	return strings.Join(parts, " ")
}

// ---- routes & basecamps ----------------------------------------------------
//
// Every sub-object mutation appends an admin_edit_log row IN THE SAME
// TRANSACTION as the data write (FR-013 attribution, research R5). The log's
// mountain_id is read from the owning row (delete) so the trail aggregates per
// mountain even after the sub-object is gone.

// execer is satisfied by both pgx.Tx and *pgxpool.Pool.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func appendEditLogTx(ctx context.Context, db execer, e domain.EditLogEntry) error {
	detail := e.Detail
	if detail == nil {
		detail = map[string]any{}
	}
	_, err := db.Exec(ctx, `
		INSERT INTO admin_edit_log (id, admin_id, entity_type, entity_id, mountain_id, action, reason, detail)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb)`,
		ids.New(), string(e.AdminID), e.EntityType, string(e.EntityID), string(e.MountainID), e.Action, e.Reason, mustJSON(detail))
	return err
}

func (w *MountainWriter) CreateRoute(ctx context.Context, mountainID, adminID ids.ID, r *domain.Route, reason string) error {
	if r.ID.IsNil() {
		r.ID = ids.New()
	}
	r.MountainID = mountainID
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below
	if _, err := tx.Exec(ctx, `
		INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours,
		                             elevation_gain_m, entry_requirements, sort_order)
		VALUES ($1,$2,$3::jsonb,$4,$5,$6,$7::jsonb,$8)`,
		string(r.ID), string(mountainID), marshalText(r.Name), r.DistanceKm,
		r.DurationHours, r.ElevationGainM, marshalText(r.EntryRequirement), r.SortOrder); err != nil {
		return err
	}
	if err := appendEditLogTx(ctx, tx, domain.EditLogEntry{
		AdminID: adminID, EntityType: domain.EditEntityRoute, EntityID: r.ID,
		MountainID: mountainID, Action: domain.EditActionCreate, Reason: reason,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// mountainOf loads the owning mountain_id of a route/basecamp row (no FK walk
// on the delete path — the row still exists at log time).
func (w *MountainWriter) mountainOfRoute(ctx context.Context, tx pgx.Tx, id ids.ID) (ids.ID, error) {
	var mid string
	err := tx.QueryRow(ctx, `SELECT mountain_id FROM mountain_routes WHERE id=$1`, string(id)).Scan(&mid)
	return ids.ID(mid), err
}

func (w *MountainWriter) mountainOfBasecamp(ctx context.Context, tx pgx.Tx, id ids.ID) (ids.ID, error) {
	var mid string
	err := tx.QueryRow(ctx, `SELECT mountain_id FROM mountain_basecamps WHERE id=$1`, string(id)).Scan(&mid)
	return ids.ID(mid), err
}

func (w *MountainWriter) UpdateRoute(ctx context.Context, adminID ids.ID, r *domain.Route, reason string) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below
	mid, err := w.mountainOfRoute(ctx, tx, r.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.MountainNotFound{}
		}
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE mountain_routes SET name=$2::jsonb, distance_km=$3, duration_hours=$4,
		       elevation_gain_m=$5, entry_requirements=$6::jsonb, sort_order=$7, updated_at=now()
		 WHERE id=$1`,
		string(r.ID), marshalText(r.Name), r.DistanceKm, r.DurationHours,
		r.ElevationGainM, marshalText(r.EntryRequirement), r.SortOrder)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.MountainNotFound{}
	}
	if err := appendEditLogTx(ctx, tx, domain.EditLogEntry{
		AdminID: adminID, EntityType: domain.EditEntityRoute, EntityID: r.ID,
		MountainID: mid, Action: domain.EditActionUpdate, Reason: reason,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *MountainWriter) DeleteRoute(ctx context.Context, id, adminID ids.ID, reason string) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below
	mid, err := w.mountainOfRoute(ctx, tx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.MountainNotFound{}
		}
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM mountain_routes WHERE id = $1`, string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.MountainNotFound{}
	}
	if err := appendEditLogTx(ctx, tx, domain.EditLogEntry{
		AdminID: adminID, EntityType: domain.EditEntityRoute, EntityID: id,
		MountainID: mid, Action: domain.EditActionDelete, Reason: reason,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *MountainWriter) CreateBasecamp(ctx context.Context, mountainID, adminID ids.ID, b *domain.Basecamp, reason string) error {
	if b.ID.IsNil() {
		b.ID = ids.New()
	}
	b.MountainID = mountainID
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below
	if _, err := tx.Exec(ctx, `
		INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate,
		                                is_permit_point, latitude, longitude, sort_order)
		VALUES ($1,$2,$3::jsonb,$4::jsonb,$5::jsonb,$6,$7,$8,$9)`,
		string(b.ID), string(mountainID), marshalText(b.Name), marshalText(b.Facilities),
		marshalText(b.CostEstimate), b.IsPermitPoint, b.Latitude, b.Longitude, b.SortOrder); err != nil {
		return err
	}
	if err := appendEditLogTx(ctx, tx, domain.EditLogEntry{
		AdminID: adminID, EntityType: domain.EditEntityBasecamp, EntityID: b.ID,
		MountainID: mountainID, Action: domain.EditActionCreate, Reason: reason,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *MountainWriter) UpdateBasecamp(ctx context.Context, adminID ids.ID, b *domain.Basecamp, reason string) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below
	mid, err := w.mountainOfBasecamp(ctx, tx, b.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.MountainNotFound{}
		}
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE mountain_basecamps SET name=$2::jsonb, facilities=$3::jsonb, cost_estimate=$4::jsonb,
		       is_permit_point=$5, latitude=$6, longitude=$7, sort_order=$8
		 WHERE id=$1`,
		string(b.ID), marshalText(b.Name), marshalText(b.Facilities),
		marshalText(b.CostEstimate), b.IsPermitPoint, b.Latitude, b.Longitude, b.SortOrder)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.MountainNotFound{}
	}
	if err := appendEditLogTx(ctx, tx, domain.EditLogEntry{
		AdminID: adminID, EntityType: domain.EditEntityBasecamp, EntityID: b.ID,
		MountainID: mid, Action: domain.EditActionUpdate, Reason: reason,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *MountainWriter) DeleteBasecamp(ctx context.Context, id, adminID ids.ID, reason string) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below
	mid, err := w.mountainOfBasecamp(ctx, tx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.MountainNotFound{}
		}
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM mountain_basecamps WHERE id = $1`, string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.MountainNotFound{}
	}
	if err := appendEditLogTx(ctx, tx, domain.EditLogEntry{
		AdminID: adminID, EntityType: domain.EditEntityBasecamp, EntityID: id,
		MountainID: mid, Action: domain.EditActionDelete, Reason: reason,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---- photo + edit log ------------------------------------------------------

// SetPhoto writes photo_key (empty key = remove), returns the prior key, and —
// like Update — snapshots the prior row so photo changes stay reversible
// (FR-005 rides the existing revision mechanism) plus an edit-log attribution.
func (w *MountainWriter) SetPhoto(ctx context.Context, mountainID, adminID ids.ID, key, reason string, deleted bool) (string, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit below

	var prior *string
	if err := tx.QueryRow(ctx, `SELECT photo_key FROM mountains WHERE id=$1`, string(mountainID)).Scan(&prior); err != nil {
		if err == pgx.ErrNoRows {
			return "", domain.MountainNotFound{}
		}
		return "", err
	}
	pk := ""
	if prior != nil {
		pk = *prior
	}

	snap, err := snapshotMountain(ctx, tx, mountainID)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO mountain_revisions (id, mountain_id, snapshot, revised_by, reason)
		VALUES ($1,$2,$3::jsonb,$4,$5)`,
		ids.New(), string(mountainID), mustJSON(snap), &adminID, photoReason(reason, deleted)); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE mountains SET photo_key=NULLIF($2,''), updated_at=now() WHERE id=$1`,
		string(mountainID), key); err != nil {
		return "", err
	}
	action := domain.EditActionPhotoSet
	if deleted {
		action = domain.EditActionPhotoRemove
	}
	if err := appendEditLogTx(ctx, tx, domain.EditLogEntry{
		AdminID: adminID, EntityType: domain.EditEntityMountain, EntityID: mountainID,
		MountainID: mountainID, Action: action, Reason: reason,
		Detail: map[string]any{"key": key, "prior_key": pk},
	}); err != nil {
		return "", err
	}
	return pk, tx.Commit(ctx)
}

func photoReason(reason string, deleted bool) string {
	if reason != "" {
		return reason
	}
	if deleted {
		return "photo removed"
	}
	return "photo set"
}

func (w *MountainWriter) AppendEditLog(ctx context.Context, e domain.EditLogEntry) error {
	return appendEditLogTx(ctx, w.pool, e)
}

func (w *MountainWriter) EditLog(ctx context.Context, mountainID ids.ID) ([]domain.EditLogEntry, error) {
	rows, err := w.pool.Query(ctx, `
		SELECT id, admin_id, entity_type, entity_id, mountain_id, action, reason, detail, created_at
		  FROM admin_edit_log WHERE mountain_id = $1
		 ORDER BY created_at DESC`, string(mountainID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.EditLogEntry
	for rows.Next() {
		var e domain.EditLogEntry
		var adminID, entID, mID string
		var detail []byte
		if err := rows.Scan(&e.ID, &adminID, &e.EntityType, &entID, &mID, &e.Action, &e.Reason, &detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.AdminID, e.EntityID, e.MountainID = ids.ID(adminID), ids.ID(entID), ids.ID(mID)
		e.Detail = map[string]any{}
		_ = json.Unmarshal(detail, &e.Detail)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---- helpers ---------------------------------------------------------------

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// mapWriteErr turns duplicate-slug violations into the domain conflict error.
func mapWriteErr(err error, _ string) error {
	if strings.Contains(err.Error(), "SQLSTATE 23505") || strings.Contains(err.Error(), "duplicate key") {
		return domain.SlugConflict{}
	}
	return err
}
