package http

import (
	"net/http"

	"skyhow/internal/http/handlers"
	"skyhow/internal/store"

	"github.com/gin-gonic/gin"
)

type RouterDeps struct {
	DiscordAuth  *handlers.DiscordAuthHandler
	Users        *store.UserStore
	Sessions     *store.SessionStore
	CookieSecure bool
	CookieDomain string
}

func NewRouter(deps RouterDeps) *gin.Engine {
	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	auth := r.Group("/auth")
	{
		auth.GET("/discord/start", deps.DiscordAuth.Start)
		auth.GET("/discord/callback", deps.DiscordAuth.Callback)
		auth.POST("/logout", deps.DiscordAuth.Logout)
	}

	r.GET("/me", func(c *gin.Context) {
		sessionID, err := c.Cookie("sb_session")
		if err != nil || sessionID == "" {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}

		userID, err := deps.Sessions.GetUserIDBySessionID(c.Request.Context(), sessionID)
		if err != nil || userID == "" {
			c.SetCookie("sb_session", "", -1, "/", deps.CookieDomain, deps.CookieSecure, true)
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}

		u, err := deps.Users.GetByID(c.Request.Context(), userID)
		if err != nil || !u.IsActive {
			c.SetCookie("sb_session", "", -1, "/", deps.CookieDomain, deps.CookieSecure, true)
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"authenticated": true,
			"user": gin.H{
				"id":           u.ID,
				"display_name": u.DisplayName,
				"avatar_url":   u.AvatarURL,
				"role":         u.Role,
			},
		})
	})

	return r
}
