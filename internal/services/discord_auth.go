package services

import (
	"context"
	"errors"
	"time"

	"skyhow/internal/auth"
	"skyhow/internal/store"
)

type AuthService struct {
	Discord    *auth.DiscordOAuth
	Users      *store.UserStore
	Sessions   *store.SessionStore
	SessionTTL time.Duration
}

func NewAuthService(
	discord *auth.DiscordOAuth,
	users *store.UserStore,
	sessions *store.SessionStore,
	sessionTTL time.Duration,
) *AuthService {
	if sessionTTL <= 0 {
		sessionTTL = 14 * 24 * time.Hour
	}
	return &AuthService{
		Discord:    discord,
		Users:      users,
		Sessions:   sessions,
		SessionTTL: sessionTTL,
	}
}

func (s *AuthService) LoginWithDiscord(ctx context.Context, code string) (string, time.Time, error) {
	if code == "" {
		return "", time.Time{}, errors.New("missing oauth code")
	}
	if s.Discord == nil || s.Users == nil || s.Sessions == nil {
		return "", time.Time{}, errors.New("auth service not configured")
	}

	token, err := s.Discord.Exchange(ctx, code)
	if err != nil {
		return "", time.Time{}, errors.New("token exchange failed")
	}

	me, err := s.Discord.FetchMe(ctx, token)
	if err != nil {
		return "", time.Time{}, errors.New("failed to fetch discord user")
	}

	displayName := me.GlobalName
	if displayName == "" {
		displayName = me.Username
	}

	var avatarURL *string
	if me.Avatar != "" {
		u := "https://cdn.discordapp.com/avatars/" + me.ID + "/" + me.Avatar + ".png?size=128"
		avatarURL = &u
	}

	if me.Email == "" {
		return "", time.Time{}, errors.New("discord did not return email; ensure scope 'email' is enabled")
	}

	userID, err := s.Users.UpsertByEmail(ctx, me.Email, displayName, avatarURL, me.Verified)
	if err != nil {
		return "", time.Time{}, errors.New("failed to upsert user")
	}

	expiresAt := time.Now().Add(s.SessionTTL)
	sessionID, err := s.Sessions.Create(ctx, userID, expiresAt)
	if err != nil {
		return "", time.Time{}, errors.New("failed to create session")
	}

	return sessionID, expiresAt, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if s.Sessions == nil {
		return errors.New("auth service not configured")
	}
	return s.Sessions.Delete(ctx, sessionID)
}
