package handlers

import (
	"log"
	"net/http"
	"time"

	"skyhow/internal/auth"
	"skyhow/internal/store"

	"github.com/gin-gonic/gin"
)

type DiscordAuthHandler struct {
	Discord      *auth.DiscordOAuth
	Users        *store.UserStore
	Sessions     *store.SessionStore
	CookieSecure bool
	CookieDomain string
}

const (
	oauthStateCookie  = "sb_oauth_state"
	returnToCookie    = "sb_return_to"
	sessionCookieName = "sb_session"
)

func NewDiscordAuthHandler(
	discord *auth.DiscordOAuth,
	users *store.UserStore,
	sessions *store.SessionStore,
	cookieSecure bool,
	cookieDomain string,
) *DiscordAuthHandler {
	return &DiscordAuthHandler{
		Discord:      discord,
		Users:        users,
		Sessions:     sessions,
		CookieSecure: cookieSecure,
		CookieDomain: cookieDomain,
	}
}

func (h *DiscordAuthHandler) Start(c *gin.Context) {
	state, err := auth.RandomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create state"})
		return
	}

	returnTo := auth.SafeReturnTo(c.Query("returnTo"))

	c.SetCookie(oauthStateCookie, state, 600, "/", h.CookieDomain, h.CookieSecure, true)
	c.SetCookie(returnToCookie, returnTo, 600, "/", h.CookieDomain, h.CookieSecure, true)

	c.Redirect(http.StatusFound, h.Discord.AuthURL(state))
}

func (h *DiscordAuthHandler) Callback(c *gin.Context) {
	expectedState, err := c.Cookie(oauthStateCookie)
	if err != nil || expectedState == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing oauth state cookie"})
		return
	}

	gotState := c.Query("state")
	if gotState == "" || gotState != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	c.SetCookie(oauthStateCookie, "", -1, "/", h.CookieDomain, h.CookieSecure, true)

	token, err := h.Discord.Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token exchange failed"})
		return
	}

	me, err := h.Discord.FetchMe(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch discord user"})
		return
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "discord did not return email; ensure scope 'email' is enabled"})
		return
	}

	userID, err := h.Users.UpsertByEmail(c.Request.Context(), me.Email, displayName, avatarURL, me.Verified)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert user"})
		return
	}

	exp := time.Now().Add(14 * 24 * time.Hour)

	if h.Sessions == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sessions store not configured"})
		return
	}

	sessionID, err := h.Sessions.Create(c.Request.Context(), userID, exp) // Added
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.SetCookie(sessionCookieName, sessionID, int(time.Until(exp).Seconds()), "/", h.CookieDomain, h.CookieSecure, true) // Changed

	returnTo, _ := c.Cookie(returnToCookie)
	c.SetCookie(returnToCookie, "", -1, "/", h.CookieDomain, h.CookieSecure, true)
	if returnTo == "" {
		returnTo = "/"
	}
	c.Redirect(http.StatusFound, returnTo)
}

func (h *DiscordAuthHandler) Logout(c *gin.Context) {
	sessionID, err := c.Cookie(sessionCookieName)
	if err != nil {
		log.Println("error", err)
		return
	}
	if err := h.Sessions.Delete(c.Request.Context(), sessionID); err != nil {
		log.Println("error on logout", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout error"})
		return
	}
	c.SetCookie(sessionCookieName, "", -1, "/", h.CookieDomain, h.CookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
