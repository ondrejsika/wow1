package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type TaskList struct {
	ID        int64
	UserID    int64
	Name      string
	CreatedAt time.Time
}

const listTaskListsSQL = `
SELECT id, user_id, name, created_at
FROM task_lists
WHERE user_id = $1
ORDER BY created_at DESC, id DESC`

func (s *Store) ListTaskLists(ctx context.Context, userID int64) ([]TaskList, error) {
	rows, err := s.Pool.Query(ctx, listTaskListsSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []TaskList
	for rows.Next() {
		var l TaskList
		if err := rows.Scan(&l.ID, &l.UserID, &l.Name, &l.CreatedAt); err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}
	return lists, rows.Err()
}

const getTaskListSQL = `
SELECT id, user_id, name, created_at
FROM task_lists
WHERE id = $1 AND user_id = $2`

// GetTaskList fetches a task list by id, scoped to userID. Returns
// ErrNotFound if the list does not exist or does not belong to userID.
func (s *Store) GetTaskList(ctx context.Context, id, userID int64) (*TaskList, error) {
	var l TaskList
	err := s.Pool.QueryRow(ctx, getTaskListSQL, id, userID).Scan(
		&l.ID, &l.UserID, &l.Name, &l.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

const createTaskListSQL = `
INSERT INTO task_lists (user_id, name)
VALUES ($1, $2)
RETURNING id, user_id, name, created_at`

func (s *Store) CreateTaskList(ctx context.Context, userID int64, name string) (*TaskList, error) {
	var l TaskList
	err := s.Pool.QueryRow(ctx, createTaskListSQL, userID, name).Scan(
		&l.ID, &l.UserID, &l.Name, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

const deleteTaskListSQL = `DELETE FROM task_lists WHERE id = $1 AND user_id = $2`

// DeleteTaskList deletes a task list, scoped to userID. Its tasks are
// removed via ON DELETE CASCADE.
func (s *Store) DeleteTaskList(ctx context.Context, id, userID int64) error {
	tag, err := s.Pool.Exec(ctx, deleteTaskListSQL, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
