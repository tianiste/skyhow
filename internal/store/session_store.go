package store

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionStore struct {
	db *pgxpool.Pool
}

func NewSessionStore(db *pgxpool.Pool) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(ctx context.Context, userID string, expiresAt time.Time) (string, error) {
	var sessionID string
	err := s.db.QueryRow(ctx, `
		insert into public.sessions (user_id, expires_at)
		values ($1, $2)
		returning id;
	`, userID, expiresAt).Scan(&sessionID)
	return sessionID, err
}

func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	_, err := s.db.Exec(ctx, `delete from public.sessions where id = $1;`, sessionID)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("deleted", sessionID)
	return nil
}

func (s *SessionStore) GetUserIDBySessionID(ctx context.Context, sessionID string) (string, error) {
	var userID string
	err := s.db.QueryRow(ctx, `
		select user_id
		from public.sessions
		where id = $1
		  and expires_at > now();
	`, sessionID).Scan(&userID)
	return userID, err
}
