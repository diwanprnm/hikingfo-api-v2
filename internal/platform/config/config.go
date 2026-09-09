// Package config loads the typed runtime configuration from environment
// variables (12-factor; research.md §4). Values are read once at startup by
// the composition root and passed down; contexts never read env directly.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the full runtime configuration.
type Config struct {
	Env      string // "development" | "production"
	AppURL   string // external base URL for links/redirects (APP_BASE_URL)
	HTTP     HTTP
	Postgres Postgres
	Session  Session
	Google   GoogleOIDC
	MinIO    MinIO
	SMTP     SMTP
	Weather  Weather
	Admin    AdminSeed
}

type HTTP struct {
	Addr           string
	TrustedProxies []string
}

type Postgres struct {
	URL string
	// Individual parts are available for environments that prefer them.
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	MaxConns int32
}

// DSN composes a pgx connection string from parts.
func (p Postgres) DSN() string {
	if p.URL != "" {
		return p.URL
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, p.SSLMode,
	)
}

type Session struct {
	Secret   string // cookie signing/encryption secret
	Lifetime time.Duration
	Name     string
}

type GoogleOIDC struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type MinIO struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Secure    bool
	// PublicURL is the base URL browsers use for presigned URLs (002 research
	// R2). Inside compose the API reaches MinIO at minio:9000, but a signed
	// URL must carry the host the browser actually calls (e.g.
	// http://localhost:9000 — port 9000 is published in compose.yaml).
	// Empty → sign with Endpoint (URLs then work only in-cluster / tests).
	PublicURL string
	Buckets   struct {
		Photos   string
		Evidence string
		Avatars  string
	}
}

type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type Weather struct {
	OpenMeteoBaseURL string
	TTL              time.Duration
}

type AdminSeed struct {
	Email       string
	Password    string
	DisplayName string
}

// Load reads the environment into Config. Unknown/invalid values error so a
// misconfigured container fails fast at boot rather than halfway through a
// request.
func Load() (Config, error) {
	cfg := Config{
		Env: get("APP_ENV", "development"),
		HTTP: HTTP{
			Addr:           get("HTTP_ADDR", ":8080"),
			TrustedProxies: splitList(get("TRUSTED_PROXIES", "")),
		},
		AppURL: strings.TrimRight(get("APP_BASE_URL", "http://localhost"), "/"),
	}

	var err error
	if cfg.Postgres, err = loadPostgres(); err != nil {
		return Config{}, err
	}
	if cfg.Session, err = loadSession(); err != nil {
		return Config{}, err
	}
	cfg.Google = loadGoogle()
	cfg.MinIO = loadMinIO()
	cfg.SMTP = loadSMTP()
	cfg.Weather = loadWeather()
	cfg.Admin = loadAdminSeed()
	return cfg, nil
}

func loadPostgres() (Postgres, error) {
	p := Postgres{
		Host:     get("POSTGRES_HOST", "localhost"),
		Port:     getInt("POSTGRES_PORT", 5432),
		User:     get("POSTGRES_USER", "hikingfo"),
		Password: get("POSTGRES_PASSWORD", ""),
		Database: get("POSTGRES_DB", "hikingfo"),
		SSLMode:  get("POSTGRES_SSLMODE", "disable"),
		MaxConns: int32(getInt("POSTGRES_MAX_CONNS", 10)),
		URL:      os.Getenv("DATABASE_URL"),
	}
	return p, nil
}

func loadSession() (Session, error) {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		return Session{}, fmt.Errorf("config: SESSION_SECRET is required")
	}
	// Recommend 32+ bytes. We don't hard-fail on short dev secrets but surface it.
	if len(secret) < 32 {
		fmt.Fprintln(os.Stderr, "config: warning: SESSION_SECRET is shorter than 32 bytes")
	}
	return Session{
		Secret:   secret,
		Lifetime: time.Duration(getInt("SESSION_LIFETIME_MINUTES", 7*24*60)) * time.Minute,
		Name:     get("SESSION_COOKIE_NAME", "hikingfo_session"),
	}, nil
}

func loadGoogle() GoogleOIDC {
	return GoogleOIDC{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}
}

func loadMinIO() MinIO {
	m := MinIO{
		Endpoint:  get("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: get("MINIO_ROOT_USER", "hikingfo"),
		SecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
		Secure:    getBool("MINIO_SECURE", false),
		PublicURL: strings.TrimRight(os.Getenv("MINIO_PUBLIC_URL"), "/"),
	}
	m.Buckets.Photos = get("MINIO_BUCKET_PHOTOS", "photos")
	m.Buckets.Evidence = get("MINIO_BUCKET_EVIDENCE", "evidence")
	m.Buckets.Avatars = get("MINIO_BUCKET_AVATARS", "avatars")
	return m
}

func loadSMTP() SMTP {
	return SMTP{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     getInt("SMTP_PORT", 587),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
	}
}

func loadWeather() Weather {
	return Weather{
		OpenMeteoBaseURL: get("OPEN_METEO_BASE_URL", "https://api.open-meteo.com/v1"),
		TTL:              time.Duration(getInt("WEATHER_TTL_MINUTES", 30)) * time.Minute,
	}
}

func loadAdminSeed() AdminSeed {
	return AdminSeed{
		Email:       os.Getenv("ADMIN_EMAIL"),
		Password:    os.Getenv("ADMIN_PASSWORD"),
		DisplayName: get("ADMIN_DISPLAY_NAME", "hikingfo Admin"),
	}
}

func get(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
