package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ry023/semdiff-sample-app/api/internal/database"
	"github.com/ry023/semdiff-sample-app/api/internal/server"
)

func main() {
	port := getenv("PORT", "8080")
	dbPath := getenv("DB_PATH", "./data/todo.db")
	cookieSecure := getenv("COOKIE_SECURE", "false") == "true"

	if err := os.MkdirAll(directoryFor(dbPath), 0o755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	store := database.NewStore(db)
	handler := server.New(store, cookieSecure)
	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	}()

	log.Printf("todo API listening on http://localhost:%s", port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func directoryFor(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}
