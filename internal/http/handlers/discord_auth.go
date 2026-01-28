package handlers

import (
	"net/http"
	"time"

	"skyhow/internal/auth"
	"skyhow/internal/services"
	"skyhow/internal/store"

	"github.com/gin-gonic/gin"
)

type DiscordAuthHandler struct {
	Discord     *auth.DiscordOAuth
	Users       *store.UserStore
	Sessions    *store.SessionStore
	AuthService *services.AuthService // Added: service dependency

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
	authSvc *services.AuthService,
	cookieSecure bool,
	cookieDomain string,
) *DiscordAuthHandler {
	return &DiscordAuthHandler{
		Discord:      discord,
		Users:        users,
		Sessions:     sessions,
		AuthService:  authSvc,
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

	if h.AuthService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth service not configured"})
		return
	}

	sessionID, expiresAt, err := h.AuthService.LoginWithDiscord(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie(
		sessionCookieName,
		sessionID,
		int(time.Until(expiresAt).Seconds()),
		"/",
		h.CookieDomain,
		h.CookieSecure,
		true,
	)

	returnTo, _ := c.Cookie(returnToCookie)
	c.SetCookie(returnToCookie, "", -1, "/", h.CookieDomain, h.CookieSecure, true)
	if returnTo == "" {
		returnTo = "/"
	}
	c.Redirect(http.StatusFound, returnTo)
}

func (h *DiscordAuthHandler) Logout(c *gin.Context) {
	sessionID, _ := c.Cookie(sessionCookieName)

	if h.AuthService != nil {
		_ = h.AuthService.Logout(c.Request.Context(), sessionID)
	} else if h.Sessions != nil && sessionID != "" {
		_ = h.Sessions.Delete(c.Request.Context(), sessionID)
	}

	c.SetCookie(sessionCookieName, "", -1, "/", h.CookieDomain, h.CookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
