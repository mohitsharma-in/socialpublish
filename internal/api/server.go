package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/yourorg/socialpublish/internal/api/handler"
	"github.com/yourorg/socialpublish/internal/api/middleware"
	"github.com/yourorg/socialpublish/internal/queue"
	"github.com/yourorg/socialpublish/internal/storage"
	"github.com/yourorg/socialpublish/internal/store"
)

// Config holds all server configuration.
type Config struct {
	ListenAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Server wraps the HTTP server with dependencies.
type Server struct {
	cfg    Config
	router *chi.Mux
	http   *http.Server
}

// New assembles the server.
func New(cfg Config, stores store.Stores, q queue.Queue, obj storage.ObjectStorage) *Server {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.Logger)

	r.Get("/health", handler.Health)
	r.Get("/readyz", handler.Readyz(stores))

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.Authenticate(stores.APIKeys))
		r.Use(middleware.InjectTenant(stores.Workspaces))
		r.Use(middleware.RateLimit(stores.Workspaces))

		r.Route("/accounts", func(r chi.Router) {
			ah := handler.NewAccount(stores.Accounts, stores.Tokens)
			r.Get("/", ah.List)
			r.Post("/connect", ah.Connect)
			r.Get("/{accountID}", ah.Get)
			r.Delete("/{accountID}", ah.Delete)
			r.Get("/{accountID}/status", ah.Status)
		})

		r.Route("/media", func(r chi.Router) {
			mh := handler.NewMedia(stores.Media, obj, q)
			r.Post("/upload", mh.Upload)
			r.Get("/", mh.List)
			r.Get("/{mediaID}", mh.Get)
			r.Delete("/{mediaID}", mh.Delete)
			r.Post("/{mediaID}/thumbnail", mh.SetThumbnail)
		})

		r.Route("/posts", func(r chi.Router) {
			ph := handler.NewPost(stores.Posts, stores.Media, q)
			r.Post("/", ph.Create)
			r.Get("/", ph.List)
			r.Get("/{postID}", ph.Get)
			r.Patch("/{postID}", ph.Update)
			r.Delete("/{postID}", ph.Delete)
			r.Post("/{postID}/publish", ph.Publish)
			r.Post("/{postID}/cancel", ph.Cancel)
			r.Post("/{postID}/duplicate", ph.Duplicate)
		})

		r.Route("/schedule", func(r chi.Router) {
			sh := handler.NewSchedule(stores.Posts)
			r.Get("/", sh.Calendar)
			r.Get("/queue", sh.Queue)
			r.Get("/next-available", sh.NextAvailable)
		})

		r.Route("/analytics", func(r chi.Router) {
			ah := handler.NewAnalytics(stores.Analytics)
			r.Get("/posts/{postID}", ah.Post)
			r.Get("/accounts/{accountID}", ah.Account)
			r.Get("/summary", ah.Summary)
		})

		r.Route("/webhooks", func(r chi.Router) {
			wh := handler.NewWebhook(stores.Webhooks, stores.Tokens)
			r.Post("/", wh.Create)
			r.Get("/", wh.List)
			r.Delete("/{webhookID}", wh.Delete)
			r.Post("/{webhookID}/test", wh.Test)
		})
	})

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return context.Background()
		},
	}

	return &Server{cfg: cfg, router: r, http: srv}
}

// Run starts the server and drains on cancellation.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	if err := s.http.Shutdown(shutCtx); err != nil {
		return err
	}
	return nil
}

// Handler returns the server's HTTP handler for tests.
func (s *Server) Handler() http.Handler {
	return s.router
}
