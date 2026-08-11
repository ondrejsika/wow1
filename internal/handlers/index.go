package handlers

import "net/http"

// Index renders the single-page TODO app: the add-task form and the
// current user's tasks, newest first.
func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	user, err := h.Store.GetUserByID(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tasks, err := h.Store.ListTasks(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, "layout", IndexData{User: user, Tasks: tasks})
}
