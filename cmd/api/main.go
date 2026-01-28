package main

import (
	"context"
	"log"
	"os"
	"time"

	"skyhow/internal/auth"
	httpapi "skyhow/internal/http"
	"skyhow/internal/http/handlers"
	"skyhow/internal/services"
	"skyhow/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	db, err := pgxpool.New(
		context.Background(),
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userStore := store.NewUserStore(db)

	discordOAuth, err := auth.NewDiscordOAuth(
		os.Getenv("DISCORD_CLIENT_ID"),
		os.Getenv("DISCORD_CLIENT_SECRET"),
		os.Getenv("DISCORD_REDIRECT_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}
	sessionStore := store.NewSessionStore(db)
	authSvc := services.NewAuthService(
		discordOAuth,
		userStore,
		sessionStore,
		14*24*time.Hour,
	)

	discordHandler := handlers.NewDiscordAuthHandler(
		discordOAuth,
		userStore,
		sessionStore,
		authSvc,
		os.Getenv("COOKIE_SECURE") == "true",
		os.Getenv("COOKIE_DOMAIN"),
	)
	router := httpapi.NewRouter(httpapi.RouterDeps{
		DiscordAuth:  discordHandler,
		Users:        userStore,
		Sessions:     sessionStore,
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
		CookieDomain: os.Getenv("COOKIE_DOMAIN"),
	})

	log.Println("listening on :8080")
	log.Fatal(router.Run(":8080"))
}
