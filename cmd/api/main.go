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

	if err := conn.AutoMigrate(&model.Item{}); err != nil {
		log.Fatalf("auto migrate error: %v", err)
	}

	srv := server.New(conn)
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)
	if err := srv.Start(addr); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
