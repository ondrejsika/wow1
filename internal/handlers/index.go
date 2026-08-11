package handlers

import (
	"errors"
	"net/http"

	"wow1/internal/store"
)

// Index handles GET /: renders the home page listing the current user's
// task lists.
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	user, err := h.Store.GetUserByID(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	lists, err := h.Store.ListTaskLists(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, "layout", IndexData{User: user, Lists: lists})
}

// ViewList handles GET /lists/{id}: renders the full page for a single
// task list and its tasks.
func (h *Handlers) ViewList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, ok := idFromPath(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	user, err := h.Store.GetUserByID(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	list, err := h.Store.GetTaskList(ctx, id, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tasks, err := h.Store.ListTasks(ctx, userID, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, "layout", IndexData{User: user, List: list, Tasks: tasks})
}
