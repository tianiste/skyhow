package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

type DiscordOAuth struct {
	oauth *oauth2.Config
}

func NewDiscordOAuth(clientID, clientSecret, redirectURL string) (*DiscordOAuth, error) {
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("discord oauth config missing env vars")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"identify", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
	return &DiscordOAuth{oauth: cfg}, nil
}

func (d *DiscordOAuth) AuthURL(state string) string {
	return d.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (d *DiscordOAuth) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return d.oauth.Exchange(ctx, code)
}

type DiscordMe struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
	Discriminator string `json:"discriminator"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	Avatar        string `json:"avatar"`
}

func (d *DiscordOAuth) FetchMe(ctx context.Context, token *oauth2.Token) (DiscordMe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
	if err != nil {
		return DiscordMe{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return DiscordMe{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return DiscordMe{}, fmt.Errorf("discord /users/@me failed: %s", resp.Status)
	}

	var me DiscordMe
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return DiscordMe{}, err
	}
	return me, nil
}

func RandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SafeReturnTo(returnTo string) string {
	if returnTo == "" {
		return "/"
	}
	u, err := url.Parse(returnTo)
	if err != nil {
		return "/"
	}
	if u.IsAbs() || u.Host != "" {
		return "/"
	}
	if len(returnTo) > 2000 {
		return "/"
	}
	if returnTo[0] != '/' {
		return "/"
	}
	return returnTo
}
