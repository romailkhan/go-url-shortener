package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/database"
	"url-shortener/internal/model"
	"url-shortener/internal/repository"
	"url-shortener/internal/server"
	"url-shortener/internal/service"
)

func main() {
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dsn, err := config.GetDatabaseURL()
	if err != nil {
		log.Fatalf("database config: %v", err)
	}

	db, err := database.Connect(dsn)
	if err != nil {
		log.Fatalf("database connect: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("database pool: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := database.Migrate(db, &model.Link{}); err != nil {
		log.Fatalf("database migrate: %v", err)
	}

	rdb, err := cache.Connect(config.RedisAddr(), config.RedisPassword())
	if err != nil {
		log.Fatalf("redis connect: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	repo := repository.NewLink(db)
	shortener := service.NewShortener(repo, rdb)

	flushCtx, stopFlush := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopFlush()
	go shortener.RunClickFlush(flushCtx)

	srv, err := server.New(shortener)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	log.Fatal(srv.Listen())
}
