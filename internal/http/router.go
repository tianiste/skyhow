package http

import (
	"net/http"

	"skyhow/internal/http/handlers"

	"github.com/gin-gonic/gin"
)

type RouterDeps struct {
	DiscordAuth *handlers.DiscordAuthHandler
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
	}

	r.GET("/me", func(c *gin.Context) {
		userID, err := c.Cookie("sb_session")
		if err != nil || userID == "" {
			c.JSON(http.StatusOK, gin.H{
				"authenticated": false,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"authenticated": true,
			"user_id":       userID,
		})
	})

	return r
}
