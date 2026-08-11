package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID          int64
	OIDCSubject string
	Email       string
	Name        string
	CreatedAt   time.Time
}

const upsertUserSQL = `
INSERT INTO users (oidc_subject, email, name)
VALUES ($1, $2, $3)
ON CONFLICT (oidc_subject) DO UPDATE
SET email = EXCLUDED.email, name = EXCLUDED.name
RETURNING id, oidc_subject, email, name, created_at`

// UpsertUser creates the user on first login, or updates their email/name
// on subsequent logins, keyed by the stable OIDC subject.
func (s *Store) UpsertUser(ctx context.Context, subject, email, name string) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, upsertUserSQL, subject, email, name).Scan(
		&u.ID, &u.OIDCSubject, &u.Email, &u.Name, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

const getUserByIDSQL = `
SELECT id, oidc_subject, email, name, created_at
FROM users
WHERE id = $1`

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, getUserByIDSQL, id).Scan(
		&u.ID, &u.OIDCSubject, &u.Email, &u.Name, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
