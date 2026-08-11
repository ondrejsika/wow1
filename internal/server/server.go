// Package server wires together config, storage, auth, and routing, and
// runs the wow1 HTTP server until interrupted.
package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"wow1/internal/config"
	"wow1/internal/handlers"
	"wow1/internal/oidcauth"
	"wow1/internal/session"
	"wow1/internal/store"
)

// Run starts the wow1 web server and blocks until it shuts down (on
// SIGINT/SIGTERM) or fails to start.
func Run(cfg *config.Config, templatesFS, staticFS, migrationsFS embed.FS) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := store.Migrate(cfg.DatabaseURL, migrationsFS); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer st.Close()

	tmpl, err := parseTemplates(templatesFS)
	if err != nil {
		return fmt.Errorf("templates: %w", err)
	}

	sessions := session.NewManager(cfg.SessionSecret)

	auth, err := oidcauth.New(ctx, cfg.OIDCIssuerURL, cfg.OIDCClientID, cfg.OIDCClientSecret, cfg.OIDCRedirectURL, st, sessions)
	if err != nil {
		return fmt.Errorf("oidc: %w", err)
	}

	h := handlers.New(tmpl, st, sessions)

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("static fs: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))
	mux.HandleFunc("GET /login", auth.Login)
	mux.HandleFunc("GET /auth/callback", auth.Callback)
	mux.HandleFunc("GET /logout", oidcauth.Logout(sessions))

	mux.HandleFunc("GET /{$}", h.RequireAuth(h.Index))
	mux.HandleFunc("POST /tasks", h.RequireAuth(h.CreateTask))
	mux.HandleFunc("GET /tasks/{id}", h.RequireAuth(h.GetTaskRow))
	mux.HandleFunc("PUT /tasks/{id}", h.RequireAuth(h.UpdateTask))
	mux.HandleFunc("DELETE /tasks/{id}", h.RequireAuth(h.DeleteTask))
	mux.HandleFunc("POST /tasks/{id}/toggle", h.RequireAuth(h.ToggleTask))
	mux.HandleFunc("GET /tasks/{id}/edit", h.RequireAuth(h.EditTask))

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("wow1 listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

func parseTemplates(templatesFS embed.FS) (*template.Template, error) {
	funcs := template.FuncMap{
		"form": func(title, errMsg string) handlers.AddTaskFormData {
			return handlers.AddTaskFormData{Title: title, Error: errMsg}
		},
		"row": func(t store.Task) handlers.TaskRowData {
			return handlers.TaskRowData{Task: t}
		},
	}
	return template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html")
}
