package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"skyhow/internal/http/handlers"
	"skyhow/internal/http/middleware"
	"skyhow/internal/store"
)

type RouterDeps struct {
	DiscordAuth  *handlers.DiscordAuthHandler
	Guides       *handlers.GuideHandler
	Users        *store.UserStore
	Sessions     *store.SessionStore
	CookieSecure bool
	CookieDomain string
}

func NewRouter(deps RouterDeps) *gin.Engine {
	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Use(middleware.AuthMiddleware(
		deps.Users,
		deps.Sessions,
		deps.CookieDomain,
		deps.CookieSecure,
	))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	auth := r.Group("/auth")
	{
		auth.GET("/discord/start", deps.DiscordAuth.Start)
		auth.GET("/discord/callback", deps.DiscordAuth.Callback)
		auth.POST("/logout", deps.DiscordAuth.Logout)
	}

	api := r.Group("/api")

	guides := api.Group("/guides")
	{
		guides.GET("", deps.Guides.ListPublished)
		guides.GET("/:id", deps.Guides.Get)

		guides.POST("", middleware.RequireAuth(), deps.Guides.Create)
		guides.PUT("/:id", middleware.RequireAuth(), deps.Guides.Update)
		guides.POST("/:id/publish", middleware.RequireAuth(), deps.Guides.Publish)
		guides.POST("/:id/unpublish", middleware.RequireAuth(), deps.Guides.Unpublish)
		guides.DELETE("/:id", middleware.RequireAuth(), deps.Guides.Delete)
	}

	r.GET("/me", func(c *gin.Context) {
		uAny, ok := c.Get("user")
		if !ok || uAny == nil {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
			return
		}

		u := uAny.(store.User)

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
