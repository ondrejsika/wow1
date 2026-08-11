// Package handlers implements the HTTP handlers for the wow1 TODO app.
package handlers

import (
	"context"
	"html/template"
	"net/http"

	"wow1/internal/session"
	"wow1/internal/store"
)

const maxTitleLength = 200

type Handlers struct {
	Templates *template.Template
	Store     *store.Store
	Sessions  *session.Manager
}

func New(tmpl *template.Template, st *store.Store, sessions *session.Manager) *Handlers {
	return &Handlers{Templates: tmpl, Store: st, Sessions: sessions}
}

type ctxKey int

const userIDKey ctxKey = 0

// RequireAuth loads the session user id into the request context, or
// redirects to /login if there is none.
func (h *Handlers) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := h.Sessions.UserID(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

func userIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

// IndexData is the root template data for the full page render.
type IndexData struct {
	User  *store.User
	Tasks []store.Task
}

// AddTaskFormData is the data for the standalone add-task form partial.
type AddTaskFormData struct {
	Title string
	Error string
}

// TaskRowData is the data for a single task row partial.
type TaskRowData struct {
	Task  store.Task
	Error string
	// OOB, if set, marks the row for htmx out-of-band swapping using this
	// swap style (e.g. "afterbegin:#task-list").
	OOB string
}

func (h *Handlers) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
