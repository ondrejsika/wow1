package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type TaskList struct {
	ID        int64  `gorm:"primaryKey"`
	UserID    int64  `gorm:"not null;index"`
	Name      string `gorm:"not null"`
	CreatedAt time.Time
}

func (s *Store) ListTaskLists(ctx context.Context, userID int64) ([]TaskList, error) {
	var lists []TaskList
	err := s.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Find(&lists).Error
	return lists, err
}

// GetTaskList fetches a task list by id, scoped to userID. Returns
// ErrNotFound if the list does not exist or does not belong to userID.
func (s *Store) GetTaskList(ctx context.Context, id, userID int64) (*TaskList, error) {
	var l TaskList
	err := s.DB.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&l).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (s *Store) CreateTaskList(ctx context.Context, userID int64, name string) (*TaskList, error) {
	l := TaskList{UserID: userID, Name: name}
	if err := s.DB.WithContext(ctx).Create(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// DeleteTaskList deletes a task list, scoped to userID, along with its
// tasks.
func (s *Store) DeleteTaskList(ctx context.Context, id, userID int64) error {
	res := s.DB.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&TaskList{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return s.DB.WithContext(ctx).Where("list_id = ?", id).Delete(&Task{}).Error
}
