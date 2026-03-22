package main

import (
	"log"

	"url-shortener/internal/config"
	"url-shortener/internal/server"
)

func main() {
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv, err := server.New()
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	log.Fatal(srv.Listen())
}
