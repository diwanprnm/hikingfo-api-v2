package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/platform/config"
)

// Server owns the Gin engine and the HTTP listener.
type Server struct {
	engine *gin.Engine
	addr   string
}

// New creates a Server with the platform middleware stack wired.
func New(cfg config.Config, pool *pgxpool.Pool, lookup SessionLookup) *Server {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware stack.
	r.Use(
		Recovery(),
		RequestID(),
		I18n(),
		SecurityHeaders(),
		DevCORS(),
	)

	// Session cookie name from config.
	cookieName := cfg.Session.Name
	if cookieName == "" {
		cookieName = "hikingfo_session"
	}

	// Attach session auth — degrades gracefully on public routes. SessionAuth
	// also sets the session-scoped CSRF cookie used by the CSRF middleware.
	r.Use(SessionAuth(cookieName, lookup))

	// CSRF protection on mutating requests (double-submit cookie pattern).
	// Skips auth endpoints that establish the session (register/login/google).
	r.Use(func(c *gin.Context) {
		switch c.FullPath() {
		case "/api/v1/auth/register", "/api/v1/auth/login", "/api/v1/auth/google":
			c.Next()
			return
		}
		CSRF("hikingfo_csrf")(c)
	})

	// Health endpoint (always public).
	r.GET("/api/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return &Server{
		engine: r,
		addr:   cfg.HTTP.Addr,
	}
}

// Engine exposes the Gin engine so the composition root can register context
// routes after New returns.
func (s *Server) Engine() *gin.Engine { return s.engine }

// Run starts the HTTP server and blocks until SIGINT/SIGTERM, then gracefully
// shuts down.
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", s.addr)
		if err := s.engine.Run(s.addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		time.Sleep(10 * time.Second)
		return nil
	}
}
