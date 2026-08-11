// Package store provides the PostgreSQL-backed data access layer for wow1,
// via GORM. Schema is kept in sync with the models through AutoMigrate.
package store

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	DB *gorm.DB
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("unwrap database: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := db.AutoMigrate(&User{}, &TaskList{}, &Task{}); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}

	return &Store{DB: db}, nil
}

func (s *Store) Close() {
	if sqlDB, err := s.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
