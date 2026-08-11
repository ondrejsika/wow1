package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Task struct {
	ID        int64
	UserID    int64
	Title     string
	Done      bool
	CreatedAt time.Time
}

const listTasksSQL = `
SELECT id, user_id, title, done, created_at
FROM tasks
WHERE user_id = $1
ORDER BY created_at DESC, id DESC`

func (s *Store) ListTasks(ctx context.Context, userID int64) ([]Task, error) {
	rows, err := s.Pool.Query(ctx, listTasksSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Done, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

const getTaskSQL = `
SELECT id, user_id, title, done, created_at
FROM tasks
WHERE id = $1 AND user_id = $2`

// GetTask fetches a task by id, scoped to userID. Returns ErrNotFound if the
// task does not exist or does not belong to userID.
func (s *Store) GetTask(ctx context.Context, id, userID int64) (*Task, error) {
	var t Task
	err := s.Pool.QueryRow(ctx, getTaskSQL, id, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Done, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

const createTaskSQL = `
INSERT INTO tasks (user_id, title)
VALUES ($1, $2)
RETURNING id, user_id, title, done, created_at`

func (s *Store) CreateTask(ctx context.Context, userID int64, title string) (*Task, error) {
	var t Task
	err := s.Pool.QueryRow(ctx, createTaskSQL, userID, title).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Done, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

const updateTaskTitleSQL = `
UPDATE tasks SET title = $1
WHERE id = $2 AND user_id = $3 AND done = false
RETURNING id, user_id, title, done, created_at`

// UpdateTaskTitle updates a task's title, scoped to userID, and only if the
// task is not done. Returns ErrNotFound if the task doesn't exist, isn't
// owned by userID, or is already done.
func (s *Store) UpdateTaskTitle(ctx context.Context, id, userID int64, title string) (*Task, error) {
	var t Task
	err := s.Pool.QueryRow(ctx, updateTaskTitleSQL, title, id, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Done, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

const toggleTaskSQL = `
UPDATE tasks SET done = NOT done
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, title, done, created_at`

func (s *Store) ToggleTask(ctx context.Context, id, userID int64) (*Task, error) {
	var t Task
	err := s.Pool.QueryRow(ctx, toggleTaskSQL, id, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Done, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

const deleteTaskSQL = `DELETE FROM tasks WHERE id = $1 AND user_id = $2`

func (s *Store) DeleteTask(ctx context.Context, id, userID int64) error {
	tag, err := s.Pool.Exec(ctx, deleteTaskSQL, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
