// Admin use-cases for the catalogue context (contracts §9). Every mutation
// goes through MountainWriter, which snapshots the prior row into
// mountain_revisions inside the same transaction.
package application

import (
	"context"
	"strings"

	uuidGo "github.com/google/uuid"

	"hikingfo/backend/internal/catalogue/domain"
	"hikingfo/backend/internal/platform/storage"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	"hikingfo/backend/internal/shared/langtext"
)

// AdminList returns every mountain regardless of publish status.
func (s *Service) AdminList(ctx context.Context) ([]domain.Mountain, error) {
	items, err := s.d.AdminWriter.ListAll(ctx)
	if err != nil {
		return nil, kerr.WrapInternal("could not list mountains", err)
	}
	s.signPhotos(items)
	return items, nil
}

// AdminGet loads one mountain by ID (drafts included).
func (s *Service) AdminGet(ctx context.Context, id ids.ID) (*domain.Mountain, error) {
	m, err := s.d.Mountains.FindByID(ctx, id)
	if err != nil {
		return nil, kerr.NotFound("mountain not found")
	}
	s.signPhoto(m)
	return m, nil
}

// AdminProfile loads the full edit view (mountain + routes + basecamps) by ID,
// drafts included, so the admin edit page needs one round-trip (002 T013).
func (s *Service) AdminProfile(ctx context.Context, id ids.ID) (*ProfileView, error) {
	m, err := s.d.Mountains.FindByID(ctx, id)
	if err != nil {
		return nil, kerr.NotFound("mountain not found")
	}
	routes, err := s.d.Mountains.Routes(ctx, id)
	if err != nil {
		return nil, kerr.WrapInternal("could not load routes", err)
	}
	basecamps, err := s.d.Mountains.Basecamps(ctx, id)
	if err != nil {
		return nil, kerr.WrapInternal("could not load basecamps", err)
	}
	s.signPhoto(m)
	return &ProfileView{Mountain: *m, Routes: routes, Basecamps: basecamps, Photos: s.galleryFor(ctx, id)}, nil
}

// CreateMountainInput is the request body for POST /admin/mountains.
type CreateMountainInput struct {
	Slug        string                     `json:"slug" binding:"required"`
	Name        map[string]string          `json:"name" binding:"required"`
	Aliases     []string                   `json:"aliases"`
	Region      string                     `json:"region" binding:"required"`
	Province    string                     `json:"province"`
	Location    map[string]string          `json:"location"`
	Latitude    float64                    `json:"latitude"`
	Longitude   float64                    `json:"longitude"`
	PeakName    map[string]string          `json:"peak_name"`
	PeakHeightM int                        `json:"peak_height_m"`
	Difficulty  int                        `json:"difficulty" binding:"required,min=1,max=5"`
	Status      string                     `json:"status"`
	DataMeta    map[string]domain.FieldMeta `json:"data_meta"`
}

// toLangText converts a string map to a bilingual Text (ID required, EN fallback).
func toLangText(m map[string]string) (langtext.Text, error) {
	if m == nil || m["id"] == "" {
		return langtext.Text{}, kerr.Validation("Indonesian text (\"id\") is required")
	}
	return langtext.From(m["id"], m["en"]), nil
}

// defaultEmpty maps a nil map to an empty one so optional bilingual fields
// don't fail validation.
func defaultEmpty(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// toLangTextOpt converts an optional string map; empty → zero Text.
func toLangTextOpt(m map[string]string) langtext.Text {
	if m == nil || m["id"] == "" {
		return langtext.Text{}
	}
	return langtext.From(m["id"], m["en"])
}

// AdminCreate creates a curated mountain.
func (s *Service) AdminCreate(ctx context.Context, in CreateMountainInput) (*domain.Mountain, error) {
	name, err := toLangText(in.Name)
	if err != nil {
		return nil, err
	}
	loc, err := toLangText(defaultEmpty(in.Location))
	if err != nil {
		return nil, err
	}
	peak, err := toLangText(defaultEmpty(in.PeakName))
	if err != nil {
		return nil, err
	}
	m := &domain.Mountain{
		Slug:        in.Slug,
		Name:        name,
		Aliases:     in.Aliases,
		Region:      domain.Region(in.Region),
		Province:    in.Province,
		Location:    loc,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
		PeakName:    peak,
		PeakHeightM: in.PeakHeightM,
		Difficulty:  in.Difficulty,
		Status:      domain.PublishStatus(in.Status),
		DataMeta:    in.DataMeta,
	}
	if m.Status == "" {
		m.Status = domain.StatusDraft
	}
	if err := s.d.AdminWriter.Create(ctx, m); err != nil {
		if _, ok := err.(domain.SlugConflict); ok {
			return nil, kerr.Conflict("slug already in use")
		}
		return nil, kerr.WrapInternal("could not create mountain", err)
	}
	return m, nil
}

// UpdateMountainInput is the request body for PATCH /admin/mountains/{id}.
type UpdateMountainInput struct {
	Slug        string                      `json:"slug"`
	Name        map[string]string           `json:"name"`
	Aliases     []string                    `json:"aliases"`
	Region      string                      `json:"region"`
	Province    string                      `json:"province"`
	Location    map[string]string           `json:"location"`
	Latitude    *float64                    `json:"latitude"`
	Longitude   *float64                    `json:"longitude"`
	PeakName    map[string]string           `json:"peak_name"`
	PeakHeightM *int                        `json:"peak_height_m"`
	Difficulty  *int                        `json:"difficulty"`
	Status      string                      `json:"status"`
	DataMeta    map[string]domain.FieldMeta `json:"data_meta"`
	Reason      string                      `json:"reason"`
}

// AdminUpdate patches a mountain; the writer snapshots the prior row first.
func (s *Service) AdminUpdate(ctx context.Context, id ids.ID, in UpdateMountainInput, adminID ids.ID) (*domain.Mountain, error) {
	m, err := s.d.Mountains.FindByID(ctx, id)
	if err != nil {
		return nil, kerr.NotFound("mountain not found")
	}
	if in.Slug != "" {
		m.Slug = in.Slug
	}
	if in.Name != nil {
		t, err := toLangText(in.Name)
		if err != nil {
			return nil, err
		}
		m.Name = t
	}
	if in.Aliases != nil {
		m.Aliases = in.Aliases
	}
	if in.Region != "" {
		m.Region = domain.Region(in.Region)
	}
	if in.Province != "" {
		m.Province = in.Province
	}
	if in.Location != nil {
		t, err := toLangText(in.Location)
		if err != nil {
			return nil, err
		}
		m.Location = t
	}
	if in.Latitude != nil {
		m.Latitude = *in.Latitude
	}
	if in.Longitude != nil {
		m.Longitude = *in.Longitude
	}
	if in.PeakName != nil {
		t, err := toLangText(in.PeakName)
		if err != nil {
			return nil, err
		}
		m.PeakName = t
	}
	if in.PeakHeightM != nil {
		m.PeakHeightM = *in.PeakHeightM
	}
	if in.Difficulty != nil {
		if *in.Difficulty < 1 || *in.Difficulty > 5 {
			return nil, kerr.Validation("difficulty must be 1–5")
		}
		m.Difficulty = *in.Difficulty
	}
	if in.Status != "" {
		m.Status = domain.PublishStatus(in.Status)
	}
	if in.DataMeta != nil {
		m.DataMeta = in.DataMeta
	}
	if err := s.d.AdminWriter.Update(ctx, m, &adminID, in.Reason); err != nil {
		if _, ok := err.(domain.SlugConflict); ok {
			return nil, kerr.Conflict("slug already in use")
		}
		return nil, kerr.WrapInternal("could not update mountain", err)
	}
	return m, nil
}

// AdminDelete removes a mountain. The cover photo object is deleted
// best-effort AFTER the row commit — storage failure never fails the delete
// (FR-014, research R7). Edit-log rows survive: audit outlives the entity.
func (s *Service) AdminDelete(ctx context.Context, id ids.ID) error {
	photoKey := ""
	if m, err := s.d.Mountains.FindByID(ctx, id); err == nil {
		photoKey = m.PhotoKey
	}
	if err := s.d.AdminWriter.Delete(ctx, id); err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return kerr.NotFound("mountain not found")
		}
		return kerr.WrapInternal("could not delete mountain", err)
	}
	s.deleteBlob(photoKey)
	return nil
}

// AdminRevisions lists a mountain's edit history, newest first.
func (s *Service) AdminRevisions(ctx context.Context, mountainID ids.ID) ([]domain.Revision, error) {
	revs, err := s.d.AdminWriter.Revisions(ctx, mountainID)
	if err != nil {
		return nil, kerr.WrapInternal("could not load revisions", err)
	}
	return revs, nil
}

// AdminRollback restores a prior revision snapshot.
func (s *Service) AdminRollback(ctx context.Context, mountainID, revisionID, adminID ids.ID) error {
	if err := s.d.AdminWriter.Rollback(ctx, mountainID, revisionID, &adminID); err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return kerr.NotFound("revision not found")
		}
		return kerr.WrapInternal("could not roll back", err)
	}
	return nil
}

// ---- routes & basecamps ----------------------------------------------------

// RouteInput is the body for admin route create/update.
type RouteInput struct {
	Name             map[string]string `json:"name" binding:"required"`
	DistanceKm       float64           `json:"distance_km"`
	DurationHours    float64           `json:"duration_hours"`
	ElevationGainM   int               `json:"elevation_gain_m"`
	EntryRequirement map[string]string `json:"entry_requirements"`
	SortOrder        int               `json:"sort_order"`
	Reason           string            `json:"reason"`
}

// BasecampInput is the body for admin basecamp create/update.
type BasecampInput struct {
	Name          map[string]string `json:"name" binding:"required"`
	Facilities    map[string]string `json:"facilities"`
	CostEstimate  map[string]string `json:"cost_estimate"`
	IsPermitPoint bool              `json:"is_permit_point"`
	Latitude      *float64          `json:"latitude"`
	Longitude     *float64          `json:"longitude"`
	SortOrder     int               `json:"sort_order"`
	Reason        string            `json:"reason"`
}

// validateRoute rejects impossible values naming the offending field (FR-012,
// SC-004: the message must say WHICH field).
func validateRoute(in RouteInput) *kerr.Error {
	if in.DistanceKm < 0 {
		return kerr.Validation("distance_km must be 0 or greater").WithField("distance_km", "must be >= 0")
	}
	if in.DurationHours <= 0 {
		return kerr.Validation("duration_hours must be greater than 0").WithField("duration_hours", "must be > 0")
	}
	if in.ElevationGainM < 0 {
		return kerr.Validation("elevation_gain_m must be 0 or greater").WithField("elevation_gain_m", "must be >= 0")
	}
	return nil
}

// validateBasecamp enforces both-or-none coordinates within ±90/±180
// (FR-010; spec edge case: a basecamp without coords is valid).
func validateBasecamp(in BasecampInput) *kerr.Error {
	if (in.Latitude == nil) != (in.Longitude == nil) {
		missing := "longitude"
		if in.Longitude != nil {
			missing = "latitude"
		}
		return kerr.Validation(missing+" is required when coordinates are given").WithField(missing, "coordinates must be both or neither")
	}
	if in.Latitude != nil {
		if *in.Latitude < -90 || *in.Latitude > 90 {
			return kerr.Validation("latitude must be between -90 and 90").WithField("latitude", "out of range")
		}
		if *in.Longitude < -180 || *in.Longitude > 180 {
			return kerr.Validation("longitude must be between -180 and 180").WithField("longitude", "out of range")
		}
	}
	return nil
}

// AdminCreateRoute adds a route to a mountain.
func (s *Service) AdminCreateRoute(ctx context.Context, mountainID, adminID ids.ID, in RouteInput) (*domain.Route, error) {
	name, err := toLangText(in.Name)
	if err != nil {
		return nil, err
	}
	if verr := validateRoute(in); verr != nil {
		return nil, verr
	}
	entry, _ := toLangText(defaultEmpty(in.EntryRequirement))
	r := &domain.Route{
		Name:             name,
		DistanceKm:       in.DistanceKm,
		DurationHours:    in.DurationHours,
		ElevationGainM:   in.ElevationGainM,
		EntryRequirement: entry,
		SortOrder:        in.SortOrder,
	}
	if err := s.d.AdminWriter.CreateRoute(ctx, mountainID, adminID, r, in.Reason); err != nil {
		return nil, kerr.WrapInternal("could not create route", err)
	}
	return r, nil
}

// AdminUpdateRoute patches a route.
func (s *Service) AdminUpdateRoute(ctx context.Context, routeID, adminID ids.ID, in RouteInput) (*domain.Route, error) {
	name, err := toLangText(in.Name)
	if err != nil {
		return nil, err
	}
	if verr := validateRoute(in); verr != nil {
		return nil, verr
	}
	entry, _ := toLangText(defaultEmpty(in.EntryRequirement))
	r := &domain.Route{
		ID:               routeID,
		Name:             name,
		DistanceKm:       in.DistanceKm,
		DurationHours:    in.DurationHours,
		ElevationGainM:   in.ElevationGainM,
		EntryRequirement: entry,
		SortOrder:        in.SortOrder,
	}
	if err := s.d.AdminWriter.UpdateRoute(ctx, adminID, r, in.Reason); err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return nil, kerr.NotFound("route not found")
		}
		return nil, kerr.WrapInternal("could not update route", err)
	}
	return r, nil
}

// AdminDeleteRoute removes a route.
func (s *Service) AdminDeleteRoute(ctx context.Context, routeID, adminID ids.ID, reason string) error {
	if err := s.d.AdminWriter.DeleteRoute(ctx, routeID, adminID, reason); err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return kerr.NotFound("route not found")
		}
		return kerr.WrapInternal("could not delete route", err)
	}
	return nil
}

// AdminCreateBasecamp adds a basecamp to a mountain.
func (s *Service) AdminCreateBasecamp(ctx context.Context, mountainID, adminID ids.ID, in BasecampInput) (*domain.Basecamp, error) {
	name, err := toLangText(in.Name)
	if err != nil {
		return nil, err
	}
	if verr := validateBasecamp(in); verr != nil {
		return nil, verr
	}
	b := &domain.Basecamp{
		Name:          name,
		Facilities:    toLangTextOpt(in.Facilities),
		CostEstimate:  toLangTextOpt(in.CostEstimate),
		IsPermitPoint: in.IsPermitPoint,
		Latitude:      in.Latitude,
		Longitude:     in.Longitude,
		SortOrder:     in.SortOrder,
	}
	if err := s.d.AdminWriter.CreateBasecamp(ctx, mountainID, adminID, b, in.Reason); err != nil {
		return nil, kerr.WrapInternal("could not create basecamp", err)
	}
	return b, nil
}

// AdminUpdateBasecamp patches a basecamp.
func (s *Service) AdminUpdateBasecamp(ctx context.Context, basecampID, adminID ids.ID, in BasecampInput) (*domain.Basecamp, error) {
	name, err := toLangText(in.Name)
	if err != nil {
		return nil, err
	}
	if verr := validateBasecamp(in); verr != nil {
		return nil, verr
	}
	b := &domain.Basecamp{
		ID:            basecampID,
		Name:          name,
		Facilities:    toLangTextOpt(in.Facilities),
		CostEstimate:  toLangTextOpt(in.CostEstimate),
		IsPermitPoint: in.IsPermitPoint,
		Latitude:      in.Latitude,
		Longitude:     in.Longitude,
		SortOrder:     in.SortOrder,
	}
	if err := s.d.AdminWriter.UpdateBasecamp(ctx, adminID, b, in.Reason); err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return nil, kerr.NotFound("basecamp not found")
		}
		return nil, kerr.WrapInternal("could not update basecamp", err)
	}
	return b, nil
}

// AdminDeleteBasecamp removes a basecamp.
func (s *Service) AdminDeleteBasecamp(ctx context.Context, basecampID, adminID ids.ID, reason string) error {
	if err := s.d.AdminWriter.DeleteBasecamp(ctx, basecampID, adminID, reason); err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return kerr.NotFound("basecamp not found")
		}
		return kerr.WrapInternal("could not delete basecamp", err)
	}
	return nil
}

// ---- photo -----------------------------------------------------------------

// validUploadKey checks a key shaped like the ones POST /uploads issues:
// exactly "uploads/<uuid>.<ext>" with a whitelisted extension (contracts
// §Photo). The store is not consulted — the key is opaque and only used as a
// blob reference (research R1).
func validUploadKey(key string) bool {
	if !strings.HasPrefix(key, "uploads/") {
		return false
	}
	rest := strings.TrimPrefix(key, "uploads/")
	if strings.ContainsRune(rest, '/') {
		return false // no nested paths
	}
	dot := strings.LastIndexByte(rest, '.')
	if dot <= 0 || dot == len(rest)-1 {
		return false
	}
	if _, err := uuidGo.Parse(rest[:dot]); err != nil {
		return false
	}
	switch rest[dot+1:] {
	case "jpg", "jpeg", "png", "webp":
		return true
	}
	return false
}

// AdminSetPhoto attaches or replaces the mountain's cover photo (FR-001/002).
// Returns the updated mountain with a fresh presigned photo_url. The prior
// object is deleted best-effort AFTER the row commit (research R7).
func (s *Service) AdminSetPhoto(ctx context.Context, mountainID, adminID ids.ID, key, reason string) (*domain.Mountain, error) {
	if !validUploadKey(key) {
		return nil, kerr.Validation("key must be an upload issued by POST /uploads").WithField("key", "invalid")
	}
	prior, err := s.d.AdminWriter.SetPhoto(ctx, mountainID, adminID, key, reason, false)
	if err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return nil, kerr.NotFound("mountain not found")
		}
		return nil, kerr.WrapInternal("could not set photo", err)
	}
	s.deleteBlob(prior)
	return s.AdminGet(ctx, mountainID)
}

// AdminRemovePhoto returns the mountain to the no-photo fallback state (FR-002).
func (s *Service) AdminRemovePhoto(ctx context.Context, mountainID, adminID ids.ID, reason string) error {
	prior, err := s.d.AdminWriter.SetPhoto(ctx, mountainID, adminID, "", reason, true)
	if err != nil {
		if _, ok := err.(domain.MountainNotFound); ok {
			return kerr.NotFound("mountain not found")
		}
		return kerr.WrapInternal("could not remove photo", err)
	}
	s.deleteBlob(prior)
	return nil
}

// AdminEditLog returns a mountain's attribution trail, newest first (FR-013).
// Admin surface only — never reachable from a public endpoint (contracts §9).
func (s *Service) AdminEditLog(ctx context.Context, mountainID ids.ID) ([]domain.EditLogEntry, error) {
	items, err := s.d.AdminWriter.EditLog(ctx, mountainID)
	if err != nil {
		return nil, kerr.WrapInternal("could not load edit log", err)
	}
	return items, nil
}

// ---- photo URL signing -----------------------------------------------------

// signPhoto derives the presigned read URL for one mountain (002 T006).
// Empty key → PhotoURL "" (json omitempty → absent, treated as null by the
// frontend's fallback tile). No store (tests/minio-less deploys) → no URL.
func (s *Service) signPhoto(m *domain.Mountain) {
	if m == nil || m.PhotoKey == "" || s.d.Blobs == nil {
		return
	}
	url, err := s.d.Blobs.PresignURL(context.Background(), storage.Key(m.PhotoKey), storage.Download, photoURLTTL)
	if err != nil {
		return // degrade to the fallback tile; never fail the read
	}
	m.PhotoURL = string(url)
}

func (s *Service) signPhotos(items []domain.Mountain) {
	for i := range items {
		s.signPhoto(&items[i])
	}
}

// deleteBlob best-effort removes an object key (no-op when empty or no store).
func (s *Service) deleteBlob(key string) {
	if key == "" || s.d.Blobs == nil {
		return
	}
	_ = s.d.Blobs.Delete(context.Background(), storage.Key(key))
}
