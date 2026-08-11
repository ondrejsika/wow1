package handlers

import (
	"errors"
	"net/http"
	"strings"

	"wow1/internal/store"
)

func validateListName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("List name is required.")
	}
	if len(name) > maxTitleLength {
		return "", errors.New("List name is too long (max 200 characters).")
	}
	return name, nil
}

// CreateTaskList handles POST /lists: validates the name, creates the
// list, and re-renders the lists overview partial.
func (h *Handlers) CreateTaskList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	rawName := r.FormValue("name")

	name, err := validateListName(rawName)
	if err != nil {
		lists, listErr := h.Store.ListTaskLists(ctx, userID)
		if listErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		h.render(w, "content", IndexData{Lists: lists, FormName: rawName, FormError: err.Error()})
		return
	}

	if _, err := h.Store.CreateTaskList(ctx, userID, name); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	lists, err := h.Store.ListTaskLists(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.render(w, "content", IndexData{Lists: lists})
}

// DeleteTaskList handles DELETE /lists/{id}: removes the list, cascading
// to its tasks, and responds with an empty body so the outerHTML swap
// removes the row from the DOM.
func (h *Handlers) DeleteTaskList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, ok := idFromPath(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	if err := h.Store.DeleteTaskList(ctx, id, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
