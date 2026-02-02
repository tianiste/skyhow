package middleware

import (
	"net/http"

	"skyhow/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const sessionCookieName = "sb_session"

func AuthMiddleware(
	users *store.UserStore,
	sessions *store.SessionStore,
	cookieDomain string,
	cookieSecure bool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(sessionCookieName)
		if err != nil || sessionID == "" {
			c.Next()
			return
		}

		userID, err := sessions.GetUserIDBySessionID(c.Request.Context(), sessionID)
		if err != nil {
			clearSessionCookie(c, cookieDomain, cookieSecure)
			c.Next()
			return
		}
		if userID == "" {
			clearSessionCookie(c, cookieDomain, cookieSecure)
			c.Next()
			return
		}

		u, err := users.GetByID(c.Request.Context(), userID)
		if err != nil {
			if err == pgx.ErrNoRows {
				clearSessionCookie(c, cookieDomain, cookieSecure)
			}
			c.Next()
			return
		}
		if !u.IsActive {
			clearSessionCookie(c, cookieDomain, cookieSecure)
			c.Next()
			return
		}

		c.Set("user", u)

		c.Header("Cache-Control", "no-store")

		c.Next()

		_ = http.StatusOK
	}
}

func clearSessionCookie(c *gin.Context, cookieDomain string, cookieSecure bool) {
	c.SetCookie(sessionCookieName, "", -1, "/", cookieDomain, cookieSecure, true)
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get("user"); !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "not authenticated",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
