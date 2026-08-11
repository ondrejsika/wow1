package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID        int64  `gorm:"primaryKey"`
	UserID    int64  `gorm:"not null;index"`
	ListID    int64  `gorm:"not null;index"`
	Title     string `gorm:"not null"`
	Done      bool   `gorm:"not null;default:false"`
	CreatedAt time.Time
}

func (s *Store) ListTasks(ctx context.Context, userID, listID int64) ([]Task, error) {
	var tasks []Task
	err := s.DB.WithContext(ctx).
		Where("list_id = ? AND user_id = ?", listID, userID).
		Order("created_at DESC, id DESC").
		Find(&tasks).Error
	return tasks, err
}

// GetTask fetches a task by id, scoped to userID. Returns ErrNotFound if the
// task does not exist or does not belong to userID.
func (s *Store) GetTask(ctx context.Context, id, userID int64) (*Task, error) {
	var t Task
	err := s.DB.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// CreateTask creates a task in listID, scoped to userID. Returns
// ErrNotFound if the list does not exist or does not belong to userID.
func (s *Store) CreateTask(ctx context.Context, userID, listID int64, title string) (*Task, error) {
	if _, err := s.GetTaskList(ctx, listID, userID); err != nil {
		return nil, err
	}
	t := Task{UserID: userID, ListID: listID, Title: title}
	if err := s.DB.WithContext(ctx).Create(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTaskTitle updates a task's title, scoped to userID, and only if the
// task is not done. Returns ErrNotFound if the task doesn't exist, isn't
// owned by userID, or is already done.
func (s *Store) UpdateTaskTitle(ctx context.Context, id, userID int64, title string) (*Task, error) {
	res := s.DB.WithContext(ctx).Model(&Task{}).
		Where("id = ? AND user_id = ? AND done = ?", id, userID, false).
		Update("title", title)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return s.GetTask(ctx, id, userID)
}

func (s *Store) ToggleTask(ctx context.Context, id, userID int64) (*Task, error) {
	task, err := s.GetTask(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Model(task).Update("done", !task.Done).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Store) DeleteTask(ctx context.Context, id, userID int64) error {
	res := s.DB.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&Task{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
