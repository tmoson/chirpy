package main

import (
	"log"
	"os"

	"database/sql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		platform = "prod"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("Need signing secret for tokens")
	}
	chirpy := defaultChirpyServer(db, platform, jwtSecret)
	log.Fatal(chirpy.runServer())
}
