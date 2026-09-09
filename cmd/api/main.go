// Command api is the composition root for the hikingfo backend.
//
// It wires config, logging, the Postgres pool, goose migrations, the identity
// session-lookup adapter, and the HTTP server (Gin) at boot.  Per-context
// routes are registered on the engine after http.New returns.
package main

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"hikingfo/backend/internal/identity/application"
	identityDomain "hikingfo/backend/internal/identity/domain"
	identityInfra "hikingfo/backend/internal/identity/infrastructure"
	identityHTTP "hikingfo/backend/internal/identity/interfaces"
	catApp "hikingfo/backend/internal/catalogue/application"
	catInfra "hikingfo/backend/internal/catalogue/infrastructure"
	catHTTP "hikingfo/backend/internal/catalogue/interfaces"
	achApp "hikingfo/backend/internal/achievement/application"
	achInfra "hikingfo/backend/internal/achievement/infrastructure"
	achHTTP "hikingfo/backend/internal/achievement/interfaces"
	jrnApp "hikingfo/backend/internal/journey/application"
	jrnDomain "hikingfo/backend/internal/journey/domain"
	jrnInfra "hikingfo/backend/internal/journey/infrastructure"
	jrnHTTP "hikingfo/backend/internal/journey/interfaces"
	modApp "hikingfo/backend/internal/moderation/application"
	modInfra "hikingfo/backend/internal/moderation/infrastructure"
	modHTTP "hikingfo/backend/internal/moderation/interfaces"
	notifApp "hikingfo/backend/internal/notification/application"
	notifDomain "hikingfo/backend/internal/notification/domain"
	notifInfra "hikingfo/backend/internal/notification/infrastructure"
	notifHTTP "hikingfo/backend/internal/notification/interfaces"
	prtApp "hikingfo/backend/internal/partner/application"
	prtInfra "hikingfo/backend/internal/partner/infrastructure"
	prtHTTP "hikingfo/backend/internal/partner/interfaces"
	"hikingfo/backend/internal/platform/config"
	"hikingfo/backend/internal/platform/db"
	httpsrv "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/platform/logging"
	"hikingfo/backend/internal/platform/storage"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Config + logger.
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	_ = logging.Setup(logging.Env(cfg.Env), os.Getenv("LOG_LEVEL"), os.Stderr)

	// 2. Postgres pool.
	ctx := context.Background()
	pool, err := db.Open(ctx, db.Options{
		DSN:      cfg.Postgres.DSN(),
		MaxConns: cfg.Postgres.MaxConns,
	})
	if err != nil {
		return err
	}
	defer pool.Close()

	// 3. Run migrations (schema, then dev/demo seed data when present).
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	if err := db.Migrate(ctx, pool, migrationsDir); err != nil {
		return err
	}
	if err := db.Migrate(ctx, pool, filepath.Join(migrationsDir, "seed")); err != nil {
		return err
	}

	// 4. Session-lookup adapter (identity → platform boundary).
	lookup := identityInfra.NewSessionLookupAdapter(pool)

	// 5a. Blob store (002 T002): MinIO for photos/uploads. Startup stays
	// graceful when unreachable or unconfigured — handlers degrade to the
	// placeholder upload / unsigned-photo modes (tests, minio-less deploys).
	var blobs storage.BlobStore
	if cfg.MinIO.SecretKey != "" {
		photoStore, err := storage.NewMinIOStore(storage.MinIOConfig{
			Endpoint:       cfg.MinIO.Endpoint,
			AccessKey:      cfg.MinIO.AccessKey,
			SecretKey:      cfg.MinIO.SecretKey,
			Secure:         cfg.MinIO.Secure,
			PublicEndpoint: publicHost(cfg.MinIO.PublicURL),
			PublicSecure:   strings.HasPrefix(cfg.MinIO.PublicURL, "https://"),
		}, cfg.MinIO.Buckets.Photos)
		if err != nil {
			slog.Warn("minio: invalid config, uploads disabled", "error", err)
		} else if err := photoStore.EnsureBucket(ctx); err != nil {
			slog.Warn("minio: unreachable, uploads in placeholder mode", "error", err)
		} else {
			blobs = photoStore
		}
	}

	// 5. HTTP server (Gin + middleware stack).
	srv := httpsrv.New(cfg, pool, lookup)

	// 6. Wire identity context.
	identitySvc := application.New(application.Dependencies{
		Users:      identityInfra.NewUserRepository(pool),
		Contacts:   identityInfra.NewContactRepository(pool),
		Sessions:   identityInfra.NewSessionRepository(pool),
		Tokens: identityHTTP.NewTokenCompose(
			identityInfra.NewEmailVerificationStore(pool),
			identityInfra.NewPasswordResetStore(pool),
		),
		Passwords:  identityInfra.NewArgon2Hasher(),
		Mailer:     identityInfra.NewMailer(toSMTPConfig(cfg.SMTP)),
		Now:        nil, // uses time.Now
		SessionTTL: cfg.Session.Lifetime,
		AppURL:     cfg.AppURL,
	})
	identityHandler := identityHTTP.NewHandler(identitySvc, identityHTTP.Config{
		CookieName: cfg.Session.Name,
		Experience: identityDomain.LevelProviderFunc(deriveLevel), // achievement bucket rules
	})
	identityHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), identityHandler)

	// 7. Wire notification context.
	notifSvc := notifApp.New(notifInfra.NewNotificationRepository(pool))
	notifHandler := notifHTTP.NewHandler(notifSvc)
	notifHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), notifHandler)

	// 8. Wire catalogue context.
	catSvc := catApp.New(catApp.Dependencies{
		Mountains:       catInfra.NewMountainRepository(pool),
		Weather:         catInfra.NewWeatherRepository(pool),
		Gallery:         catInfra.NewGalleryRepository(pool),
		AdminWriter:     catInfra.NewMountainWriter(pool),
		Blobs:           blobs,
		WeatherBaseURL:  cfg.Weather.OpenMeteoBaseURL,
		WeatherTTL:      cfg.Weather.TTL,
	})
	catHandler := catHTTP.NewHandler(catSvc)
	catHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), catHandler)
	catHTTP.RegisterAdminRoutes(srv.Engine().Group("/api/v1"), catHandler)

	// 9. Wire journey context (US2). OnHikeRecorded evaluates badges and
	// records badge_earned notifications (T067) — the achievement context is
	// read-time derived, so the hook only announces newly earned milestones.
	// achSvc/notifSvc are declared before use in the closure below.
	achSvc := achApp.New(achApp.Dependencies{
		Badges:   achInfra.NewBadgeConfigRepo(pool),
		Journeys: achInfra.NewJourneyQuery(pool),
	})
	jrnSvc := jrnApp.New(jrnApp.Dependencies{
		Hikes: jrnInfra.NewHikeLogRepository(pool),
		Posts: jrnInfra.NewJourneyPostRepository(pool),
		SlugLookup: func(ctx context.Context, slug string) (ids.ID, error) {
			m, err := catInfra.NewMountainRepository(pool).FindBySlug(ctx, slug)
			if err != nil {
				return ids.Nil, kerr.NotFound("mountain not found")
			}
			return m.ID, nil
		},
		OnHikeRecorded: func(ctx context.Context, userID ids.ID) error {
			return evaluateAndNotifyBadges(ctx, achSvc, notifSvc, userID)
		},
	})
	jrnHandler := jrnHTTP.NewHandler(jrnSvc, blobs)
	jrnHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), jrnHandler)

	// 10. Achievement context (US3) — routes only; the service was built above.
	achHandler := achHTTP.NewHandler(achSvc)
	achHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), achHandler)
	achHTTP.RegisterAdminRoutes(srv.Engine().Group("/api/v1"), achHandler)

	// 11. Wire partner context (US5) — reads identity via declared ports and
	// notifies request lifecycle events (T089 backend half).
	prtSvc := prtApp.New(prtApp.Dependencies{
		Notices:  prtInfra.NewNoticeRepository(pool),
		Requests: prtInfra.NewRequestRepository(pool),
		Profiles: profileReader{svc: identitySvc},
		Revealer: contactRevealer{svc: identitySvc},
		OnRequestSent: func(ctx context.Context, recipient, sender ids.ID, payload map[string]any) {
			_ = notifSvc.Record(ctx, recipient, notifDomain.TypePartnerRequestReceived, payload)
		},
		OnRequestAccepted: func(ctx context.Context, sender, recipient ids.ID, payload map[string]any) {
			_ = notifSvc.Record(ctx, sender, notifDomain.TypePartnerRequestAccepted, payload)
		},
	})
	prtHandler := prtHTTP.NewHandler(prtSvc)
	prtHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), prtHandler)

	// 12. Wire moderation context (US4 admin queue) — resolution side effects
	// call into journey/notification via closures declared here.
	modRepo := modInfra.NewReportRepository(pool)
	modOwners := modInfra.NewOwners(pool)
	modSvc := modApp.New(modRepo, modApp.SideEffects{
		HidePost: func(ctx context.Context, postID ids.ID) error {
			return jrnInfra.NewJourneyPostRepository(pool).SetModerationStatus(
				ctx, postID, jrnDomain.ModerationHidden)
		},
		RemoveEvidence: func(ctx context.Context, hikeID ids.ID) error {
			return jrnInfra.NewHikeLogRepository(pool).SetStatus(
				ctx, hikeID, jrnDomain.HikeStatusRemoved)
		},
		Notify: func(ctx context.Context, userID ids.ID, payload map[string]any) error {
			return notifSvc.Record(ctx, userID, notifDomain.TypeModerationOutcome, payload)
		},
	})
	modSvc.JourneyOwner = modOwners.JourneyOwner
	modSvc.HikeOwner = modOwners.HikeOwner
	modSvc.Blocks = modInfra.NewBlockRepository(pool)
	modHandler := modHTTP.NewHandler(modSvc)
	modHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), modHandler)
	modHTTP.RegisterAdminRoutes(srv.Engine().Group("/api/v1"), modHandler)

	// 12a. US5 pre-match safety: partner request reports → moderation queue.
	prtHandler.SetModeration(modSvc)

	// 12a. Wire catalogue field reports → moderation queue (FR-003,
	// mountain_field target — moderation service exists by now).
	catHandler.SetFieldReport(func(ctx context.Context, mountainID ids.ID, field, reason, detail string) error {
		_, err := modSvc.Create(ctx, nil, "mountain_field", string(mountainID), field, reason, detail)
		return err
	})

	// 12b. Wire GET /admin/stats: catalogue completeness + report aging.
	identityHandler.SetStats(func(ctx context.Context) (map[string]any, error) {
		mountains, err := catInfra.NewMountainWriter(pool).ListAll(ctx)
		if err != nil {
			return nil, err
		}
		complete := 0
		for _, m := range mountains {
			if m.Location.ID != "" && len(m.DataMeta) > 0 {
				complete++
			}
		}
		reportCounts, err := modSvc.Counts(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"mountains": map[string]any{
				"total":     len(mountains),
				"complete":  complete,
			},
			"reports": reportCounts,
		}, nil
	})

	// 13. Run until SIGINT/SIGTERM.
	return srv.Run()
}

// deriveLevel maps a verified distinct-mountain count to the identity
// experience-level buckets. The rules live in the achievement context (which
// owns derived badge/level math); identity only declares the provider port, so
// the composition root supplies the concrete bucket function here.
func deriveLevel(count int) identityDomain.ExperienceLevel {
	switch {
	case count >= 35:
		return identityDomain.LevelAhli
	case count >= 15:
		return identityDomain.LevelLanjut
	case count >= 5:
		return identityDomain.LevelMenengah
	default:
		return identityDomain.LevelPemula
	}
}

// evaluateAndNotifyBadges compares the current earned badge set against the
// last-notified threshold stored per user (in notifications), and records a
// badge_earned notification for every newly crossed milestone. Idempotent per
// threshold: re-recording requires the count to reach a strictly higher tier.
// ponytail: last-notified state rides on prior badge_earned payloads (max
// threshold), upgrade to a dedicated table when lapses need per-tier tracking.
func evaluateAndNotifyBadges(ctx context.Context, achSvc *achApp.Service, notifSvc *notifApp.Service, userID ids.ID) error {
	view, err := achSvc.GetBadges(ctx, userID)
	if err != nil {
		return err
	}
	// Highest threshold the user has already been notified about.
	notifiedAt := 0
	items, _, err := notifSvc.List(ctx, userID, 1, 100)
	if err == nil {
		for _, n := range items {
			if n.Type != notifDomain.TypeBadgeEarned {
				continue
			}
			if v, ok := n.Payload["threshold"].(float64); ok && int(v) > notifiedAt {
				notifiedAt = int(v)
			}
		}
	}
	for _, b := range view.Badges {
		if !b.Earned || b.Threshold <= notifiedAt {
			continue
		}
		_ = notifSvc.Record(ctx, userID, notifDomain.TypeBadgeEarned, map[string]any{
			"badge_key":  b.Key,
			"threshold":  b.Threshold,
			"badge_name": b.Name.ID,
		})
	}
	return nil
}

// toSMTPConfig maps the platform config shape to the identity infrastructure shape.
func toSMTPConfig(c config.SMTP) identityInfra.SMTPConfig {
	return identityInfra.SMTPConfig{
		Host:     c.Host,
		Port:     c.Port,
		Username: c.Username,
		Password: c.Password,
		From:     c.From,
	}
}

// publicHost extracts host:port from a base URL for MinIO presign signing
// ("" when the URL is empty or unparseable).
func publicHost(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return u.Host
}
