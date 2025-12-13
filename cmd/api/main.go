package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/shinyyama/hackathon-backend/internal/config"
	"github.com/shinyyama/hackathon-backend/internal/db"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/server"
)

// Starts the HTTP server immediately and runs DB migration asynchronously.
func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	conn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	srv := server.New(conn)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	errCh := make(chan error, 1)

	// Start HTTP server immediately (foreground via channel wait).
	go func() {
		log.Printf("starting server on %s", addr)
		errCh <- srv.Start(addr)
	}()

	// Run AutoMigrate asynchronously so it does not block startup.
	go func() {
		if err := conn.AutoMigrate(&model.Item{}); err != nil {
			log.Printf("auto migrate error: %v", err)
		} else {
			log.Printf("auto migrate completed")
		}
	}()

	// Block main goroutine until server stops.
	if err := <-errCh; err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
