package main

import (
	"context"
	"log"
	"os"

	"skyhow/internal/auth"
	httpapi "skyhow/internal/http"
	"skyhow/internal/http/handlers"
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

	discordHandler := handlers.NewDiscordAuthHandler(
		discordOAuth,
		userStore,
		os.Getenv("COOKIE_SECURE") == "true",
		os.Getenv("COOKIE_DOMAIN"),
	)

	router := httpapi.NewRouter(httpapi.RouterDeps{
		DiscordAuth: discordHandler,
	})

	log.Println("listening on :8080")
	log.Fatal(router.Run(":8080"))
}
