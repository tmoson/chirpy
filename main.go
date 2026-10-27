package main

import (
	"log"
)

func main() {
	chirpy := defaultChirpyServer()
	log.Fatal(chirpy.runServer())
}
