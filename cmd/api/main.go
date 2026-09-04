// Command api is the composition root for the hikingfo backend.
//
// It wires config, logging, the Postgres pool, goose migrations, the identity
// session-lookup adapter, and the HTTP server (Gin) at boot.  Per-context
// routes are registered on the engine after http.New returns.
package main

import (
	"context"
	"log/slog"
	"os"

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
	jrnInfra "hikingfo/backend/internal/journey/infrastructure"
	jrnHTTP "hikingfo/backend/internal/journey/interfaces"
	notifApp "hikingfo/backend/internal/notification/application"
	notifInfra "hikingfo/backend/internal/notification/infrastructure"
	notifHTTP "hikingfo/backend/internal/notification/interfaces"
	prtApp "hikingfo/backend/internal/partner/application"
	prtInfra "hikingfo/backend/internal/partner/infrastructure"
	prtHTTP "hikingfo/backend/internal/partner/interfaces"
	"hikingfo/backend/internal/platform/config"
	"hikingfo/backend/internal/platform/db"
	httpsrv "hikingfo/backend/internal/platform/http"
	"hikingfo/backend/internal/platform/logging"
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

	// 3. Run migrations.
	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		return err
	}

	// 4. Session-lookup adapter (identity → platform boundary).
	lookup := identityInfra.NewSessionLookupAdapter(pool)

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
		WeatherBaseURL:  cfg.Weather.OpenMeteoBaseURL,
		WeatherTTL:      cfg.Weather.TTL,
	})
	catHandler := catHTTP.NewHandler(catSvc)
	catHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), catHandler)

	// 9. Wire journey context (US2).
	jrnSvc := jrnApp.New(jrnApp.Dependencies{
		Hikes: jrnInfra.NewHikeLogRepository(pool),
		Posts: jrnInfra.NewJourneyPostRepository(pool),
	})
	jrnHandler := jrnHTTP.NewHandler(jrnSvc)
	jrnHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), jrnHandler)

	// 10. Wire achievement context (US3) — reads journey facts via a declared port.
	achSvc := achApp.New(achApp.Dependencies{
		Badges:   achInfra.NewBadgeConfigRepo(pool),
		Journeys: achInfra.NewJourneyQuery(pool),
	})
	achHandler := achHTTP.NewHandler(achSvc)
	achHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), achHandler)

	// 11. Wire partner context (US5) — reads identity via declared ports.
	prtSvc := prtApp.New(prtApp.Dependencies{
		Notices:  prtInfra.NewNoticeRepository(pool),
		Requests: prtInfra.NewRequestRepository(pool),
		Profiles: profileReader{svc: identitySvc},
		Revealer: contactRevealer{svc: identitySvc},
	})
	prtHandler := prtHTTP.NewHandler(prtSvc)
	prtHTTP.RegisterRoutes(srv.Engine().Group("/api/v1"), prtHandler)

	// 12. Run until SIGINT/SIGTERM.
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
