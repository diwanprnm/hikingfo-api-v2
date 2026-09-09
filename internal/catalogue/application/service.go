// Package application holds the catalogue bounded context's use-cases.
// Pure Go — depends only on catalogue/domain and the shared kernel.
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"hikingfo/backend/internal/catalogue/domain"
	"hikingfo/backend/internal/platform/storage"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// photoURLTTL is how long a derived presigned photo_url stays valid
// (002 research R2: 24 h — long enough for a cached list page to still render).
const photoURLTTL = 24 * time.Hour

// Dependencies the catalogue use-cases need.
type Dependencies struct {
	Mountains domain.MountainRepository
	Weather   domain.WeatherStore
	// Gallery is the 004 curated-photo port (nil-tolerant: profile reads
	// return an empty photos array without it).
	Gallery domain.GalleryRepository
	// AdminWriter persists curated mountains (nil in public-only deployments).
	AdminWriter domain.MountainWriter
	// Blobs presigns photo read URLs and deletes prior photo objects.
	// Nil-tolerant: reads degrade to no photo_url, deletions become no-ops
	// (tests and minio-less deploys).
	Blobs storage.BlobStore
	// WeatherBaseURL is the Open-Meteo API base URL.
	WeatherBaseURL string
	// WeatherTTL is how long a cached forecast stays live.
	WeatherTTL time.Duration
	// Now returns the current time (injectable for tests).
	Now func() time.Time
}

// Service exposes the catalogue use-cases to the interfaces layer.
type Service struct{ d Dependencies }

// New wires a Service.
func New(d Dependencies) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.WeatherTTL <= 0 {
		d.WeatherTTL = 30 * time.Minute
	}
	if d.WeatherBaseURL == "" {
		d.WeatherBaseURL = "https://api.open-meteo.com/v1"
	}
	return &Service{d: d}
}

// ---- Search & Browse ------------------------------------------------------

// SearchResult is a page of mountains.
type SearchResult struct {
	Items    []domain.Mountain `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// SearchAndFilter returns a paginated list of published mountains.
func (s *Service) SearchAndFilter(ctx context.Context, f domain.SearchFilter) (*SearchResult, error) {
	items, total, err := s.d.Mountains.SearchAndFilter(ctx, f)
	if err != nil {
		return nil, kerr.WrapInternal("could not search mountains", err)
	}
	s.signPhotos(items) // 002 §read-model: photo_url derived on every public list
	return &SearchResult{
		Items:    items,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
	}, nil
}

// Regions returns the distinct regions that have published mountains.
func (s *Service) Regions(ctx context.Context) ([]domain.Region, error) {
	return s.d.Mountains.Regions(ctx)
}

// ---- Mountain Profile -----------------------------------------------------

// ProfileView assembles all field blocks + provenance + weather for a mountain.
type ProfileView struct {
	Mountain  domain.Mountain          `json:"mountain"`
	Routes    []domain.Route           `json:"routes"`
	Basecamps []domain.Basecamp        `json:"basecamps"`
	Photos    []domain.MountainPhoto   `json:"photos"` // 004: always an array (FR-003)
	Weather   *domain.WeatherSnapshot  `json:"weather,omitempty"`
}

// GetProfile loads the full mountain profile by slug.
func (s *Service) GetProfile(ctx context.Context, slug string) (*ProfileView, error) {
	m, err := s.d.Mountains.FindBySlug(ctx, slug)
	if err != nil {
		return nil, kerr.NotFound("mountain not found")
	}
	routes, err := s.d.Mountains.Routes(ctx, m.ID)
	if err != nil {
		return nil, kerr.WrapInternal("could not load routes", err)
	}
	basecamps, err := s.d.Mountains.Basecamps(ctx, m.ID)
	if err != nil {
		return nil, kerr.WrapInternal("could not load basecamps", err)
	}
	photos := s.galleryFor(ctx, m.ID)
	weather := s.getWeather(ctx, m.ID)
	s.signPhoto(m) // 002 §read-model: presigned photo_url (24 h) or null
	return &ProfileView{
		Mountain:  *m,
		Routes:    routes,
		Basecamps: basecamps,
		Photos:    photos,
		Weather:   weather,
	}, nil
}

// galleryFor loads a mountain's curated photos with presigned read URLs.
// Nil repo → empty array (FR-003 always an array); signing is best-effort.
func (s *Service) galleryFor(ctx context.Context, mountainID ids.ID) []domain.MountainPhoto {
	if s.d.Gallery == nil {
		return []domain.MountainPhoto{}
	}
	photos, err := s.d.Gallery.ListByMountain(ctx, mountainID)
	if err != nil {
		return []domain.MountainPhoto{}
	}
	if photos == nil {
		photos = []domain.MountainPhoto{}
	}
	for i := range photos {
		s.signGalleryPhoto(&photos[i])
	}
	return photos
}

// signGalleryPhoto derives a presigned read URL for one gallery photo (004
// R3 — same TTL/pattern as the cover photo).
func (s *Service) signGalleryPhoto(p *domain.MountainPhoto) {
	if p == nil || p.PhotoKey == "" || s.d.Blobs == nil {
		return
	}
	url, err := s.d.Blobs.PresignURL(context.Background(), storage.Key(p.PhotoKey), storage.Download, photoURLTTL)
	if err != nil {
		return // degrade; the frontend fallback tile handles a missing URL
	}
	p.PhotoURL = string(url)
}

// AdminAddPhoto attaches a curated photo to a mountain and logs provenance
// (004 FR-013/FR-014, constitution II). The photo_key must come from
// POST /uploads; the blob itself already exists (uploaded via the shared
// pipeline), so nothing is stored here beyond the reference row.
func (s *Service) AdminAddPhoto(ctx context.Context, mountainID, adminID ids.ID, key string) (*domain.MountainPhoto, error) {
	if !validUploadKey(key) {
		return nil, kerr.Validation("key must be an upload issued by POST /uploads").WithField("key", "invalid")
	}
	if s.d.Gallery == nil {
		return nil, kerr.Internal("gallery not wired")
	}
	if _, err := s.d.Mountains.FindByID(ctx, mountainID); err != nil {
		return nil, kerr.NotFound("mountain not found")
	}
	p := &domain.MountainPhoto{ID: ids.New(), MountainID: mountainID, PhotoKey: key}
	if err := s.d.Gallery.Insert(ctx, p); err != nil {
		return nil, kerr.WrapInternal("could not add photo", err)
	}
	if s.d.AdminWriter != nil {
		_ = s.d.AdminWriter.AppendEditLog(ctx, domain.EditLogEntry{
			AdminID: adminID, EntityType: domain.EditEntityMountain, EntityID: p.ID,
			MountainID: mountainID, Action: domain.EditActionPhotoSet,
			Detail: map[string]any{"key": key},
		})
	}
	s.signGalleryPhoto(p)
	return p, nil
}

// AdminDeletePhoto removes a curated photo (author-of-record path). 404 when
// the photo is not on this mountain; provenance logged (004 FR-014).
func (s *Service) AdminDeletePhoto(ctx context.Context, photoID, mountainID, adminID ids.ID) error {
	if s.d.Gallery == nil {
		return kerr.Internal("gallery not wired")
	}
	// ponytail: key resolved by listing (gallery volumes are low per plan.md);
	// upgrade path is a FindByID on the port if it ever matters.
	key := ""
	if photos, err := s.d.Gallery.ListByMountain(ctx, mountainID); err == nil {
		for _, p := range photos {
			if p.ID == photoID {
				key = p.PhotoKey
				break
			}
		}
	}
	if err := s.d.Gallery.Delete(ctx, photoID, mountainID); err != nil {
		if _, ok := err.(domain.PhotoNotFound); ok {
			return kerr.NotFound("photo not found")
		}
		return kerr.WrapInternal("could not delete photo", err)
	}
	if s.d.AdminWriter != nil {
		_ = s.d.AdminWriter.AppendEditLog(ctx, domain.EditLogEntry{
			AdminID: adminID, EntityType: domain.EditEntityMountain, EntityID: photoID,
			MountainID: mountainID, Action: domain.EditActionPhotoRemove,
			Detail: map[string]any{"key": key},
		})
	}
	return nil
}

// ---- Weather --------------------------------------------------------------

// getWeather returns the cached weather, attempting a refresh if stale.
func (s *Service) getWeather(ctx context.Context, mountainID ids.ID) *domain.WeatherSnapshot {
	w, err := s.d.Weather.Get(ctx, mountainID)
	if err != nil || w == nil {
		return nil
	}
	// If stale, attempt a background refresh but still return the cached data.
	if s.d.Now().After(w.ExpiresAt) {
		w.IsLive = false
		go s.refreshWeather(mountainID)
	}
	return w
}

// GetWeather returns weather for a mountain by slug, refreshing if stale.
func (s *Service) GetWeather(ctx context.Context, slug string) (*domain.WeatherSnapshot, error) {
	m, err := s.d.Mountains.FindBySlug(ctx, slug)
	if err != nil {
		return nil, kerr.NotFound("mountain not found")
	}
	w := s.getWeather(ctx, m.ID)
	if w == nil {
		// No cached data — fetch now (blocking).
		w, err = s.fetchAndCacheWeather(ctx, m)
		if err != nil {
			return nil, kerr.WrapInternal("could not fetch weather", err)
		}
	}
	return w, nil
}

// RefreshWeather fetches the latest forecast from Open-Meteo and caches it.
func (s *Service) refreshWeather(mountainID ids.ID) {
	// Background refresh — best effort, errors logged.
	_ = s.fetchAndCacheWeatherByID(context.Background(), mountainID)
}

func (s *Service) fetchAndCacheWeather(ctx context.Context, m *domain.Mountain) (*domain.WeatherSnapshot, error) {
	url := fmt.Sprintf("%s/forecast?latitude=%f&longitude=%f&hourly=temperature_2m,precipitation_probability,weathercode&daily=temperature_2m_max,temperature_2m_min,precipitation_sum,weathercode&timezone=Asia/Jakarta",
		s.d.WeatherBaseURL, m.Latitude, m.Longitude)

	resp, err := http.Get(url) //nolint:gosec // Open-Meteo is a public API
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("open-meteo: %d: %s", resp.StatusCode, body)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	now := s.d.Now()
	w := &domain.WeatherSnapshot{
		MountainID: string(m.ID),
		CapturedAt: now,
		IsLive:     true,
		ExpiresAt:  now.Add(s.d.WeatherTTL),
		Payload:    data,
	}
	_ = s.d.Weather.Upsert(ctx, w)
	return w, nil
}

func (s *Service) fetchAndCacheWeatherByID(ctx context.Context, mountainID ids.ID) error {
	m, err := s.d.Mountains.FindByID(ctx, mountainID)
	if err != nil {
		return err
	}
	_, err = s.fetchAndCacheWeather(ctx, m)
	return err
}

// idsFromString is a helper to convert a string to ids.ID.
func idsFromString(s string) ids.ID { return ids.ID(s) }

// Silence unused import warnings.
var _ = io.ReadAll
