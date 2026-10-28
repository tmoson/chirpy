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
	chirpy := defaultChirpyServer(db, platform)
	log.Fatal(chirpy.runServer())
}
