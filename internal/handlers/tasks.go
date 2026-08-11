package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"wow1/internal/store"
)

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", errors.New("Title is required.")
	}
	if len(title) > maxTitleLength {
		return "", errors.New("Title is too long (max 200 characters).")
	}
	return title, nil
}

func taskIDFromRequest(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil
}

// CreateTask handles POST /tasks: validates the title, creates the task,
// and re-renders the task-app partial (add-task form plus the full task
// list) so the new row is always correctly wrapped and in place.
func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	rawTitle := r.FormValue("title")

	title, err := validateTitle(rawTitle)
	if err != nil {
		tasks, listErr := h.Store.ListTasks(ctx, userID)
		if listErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		h.render(w, "content", IndexData{Tasks: tasks, FormTitle: rawTitle, FormError: err.Error()})
		return
	}

	if _, err := h.Store.CreateTask(ctx, userID, title); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tasks, err := h.Store.ListTasks(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.render(w, "content", IndexData{Tasks: tasks})
}

// ToggleTask handles POST /tasks/{id}/toggle: flips done and re-renders the
// row.
func (h *Handlers) ToggleTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, ok := taskIDFromRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	task, err := h.Store.ToggleTask(ctx, id, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, "task_row", TaskRowData{Task: *task})
}

// GetTaskRow handles GET /tasks/{id}: re-renders the normal (non-editing)
// row, used by the edit form's Cancel button.
func (h *Handlers) GetTaskRow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, ok := taskIDFromRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	task, err := h.Store.GetTask(ctx, id, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, "task_row", TaskRowData{Task: *task})
}

// EditTask handles GET /tasks/{id}/edit: swaps the row to an inline edit
// form. Done tasks cannot be edited, even via a direct request.
func (h *Handlers) EditTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, ok := taskIDFromRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	task, err := h.Store.GetTask(ctx, id, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if task.Done {
		h.render(w, "task_row", TaskRowData{Task: *task, Error: "Cannot edit a completed task."})
		return
	}

	h.render(w, "task_edit", TaskRowData{Task: *task})
}

// UpdateTask handles PUT /tasks/{id}: validates and saves the new title,
// rejecting edits to already-done tasks server-side regardless of what the
// UI allowed.
func (h *Handlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, ok := taskIDFromRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	existing, err := h.Store.GetTask(ctx, id, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if existing.Done {
		h.render(w, "task_row", TaskRowData{Task: *existing, Error: "Cannot edit a completed task."})
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	rawTitle := r.FormValue("title")

	title, err := validateTitle(rawTitle)
	if err != nil {
		h.render(w, "task_edit", TaskRowData{
			Task:  store.Task{ID: existing.ID, UserID: existing.UserID, Title: rawTitle, Done: existing.Done, CreatedAt: existing.CreatedAt},
			Error: err.Error(),
		})
		return
	}

	task, err := h.Store.UpdateTaskTitle(ctx, id, userID, title)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Task became done or was deleted concurrently.
			h.render(w, "task_row", TaskRowData{Task: *existing, Error: "Could not save: task is no longer editable."})
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.render(w, "task_row", TaskRowData{Task: *task})
}

// DeleteTask handles DELETE /tasks/{id}: removes the task and responds
// with an empty body so the outerHTML swap removes the row from the DOM.
func (h *Handlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)

	id, ok := taskIDFromRequest(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if err := h.Store.DeleteTask(ctx, id, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
