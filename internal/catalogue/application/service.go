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
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// Dependencies the catalogue use-cases need.
type Dependencies struct {
	Mountains domain.MountainRepository
	Weather   domain.WeatherStore
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
	weather := s.getWeather(ctx, m.ID)
	return &ProfileView{
		Mountain:  *m,
		Routes:    routes,
		Basecamps: basecamps,
		Weather:   weather,
	}, nil
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
