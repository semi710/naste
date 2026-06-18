package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/semi710/nastebin/internal/config"
	"github.com/semi710/nastebin/internal/handlers"
	"github.com/semi710/nastebin/internal/storage"
)

func main() {
	cfg := config.Load()

	store, err := storage.NewStore(cfg)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	h := handlers.NewHandler(cfg, store)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Apply middleware
	var handler http.Handler = mux
	handler = handlers.SecurityHeaders(handler)

	server := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        handler,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	go func() {
		log.Printf("naste-server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("server stopped")
}
