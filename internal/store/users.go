package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID            string
	DisplayName   string
	AvatarURL     *string
	Email         *string
	EmailVerified bool
	Role          string
	IsActive      bool
}

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) UpsertByEmail(ctx context.Context, email string, displayName string, avatarURL *string, emailVerified bool) (string, error) {
	if email == "" {
		return "", errors.New("email is required to upsert user")
	}

	var userID string
	err := s.db.QueryRow(ctx, `
		insert into public.users (display_name, avatar_url, email, email_verified)
		values ($1, $2, $3, $4)
		on conflict (lower(email)) where email is not null
		do update set
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			email_verified = excluded.email_verified,
			updated_at = now()
		returning id;
	`, displayName, avatarURL, email, emailVerified).Scan(&userID)

	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *UserStore) GetByID(ctx context.Context, userID string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		select id, display_name, avatar_url, email, email_verified, role, is_active
		from public.users
		where id = $1;
	`, userID).Scan(
		&u.ID,
		&u.DisplayName,
		&u.AvatarURL,
		&u.Email,
		&u.EmailVerified,
		&u.Role,
		&u.IsActive,
	)
	return u, err
}
