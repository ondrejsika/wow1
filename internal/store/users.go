package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID          int64  `gorm:"primaryKey"`
	OIDCSubject string `gorm:"column:oidc_subject;uniqueIndex;not null"`
	Email       string `gorm:"not null"`
	Name        string `gorm:"not null"`
	CreatedAt   time.Time
}

// UpsertUser creates the user on first login, or updates their email/name
// on subsequent logins, keyed by the stable OIDC subject.
func (s *Store) UpsertUser(ctx context.Context, subject, email, name string) (*User, error) {
	u := User{OIDCSubject: subject, Email: email, Name: name}
	err := s.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "oidc_subject"}},
			DoUpdates: clause.AssignmentColumns([]string{"email", "name"}),
		}).
		Create(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	if err := s.DB.WithContext(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
