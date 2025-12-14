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

	sha := os.Getenv("GIT_SHA")
	buildTime := os.Getenv("BUILD_TIME")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	errCh := make(chan error, 1)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}
	conn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	if err := conn.AutoMigrate(&model.Item{}, &model.Conversation{}, &model.Message{}, &model.ConversationState{}, &model.Purchase{}); err != nil {
		log.Fatalf("auto migrate error: %v", err)
	}

	srv := server.New(conn, sha, buildTime)

	go func() {
		log.Printf("starting server on %s", addr)
		errCh <- srv.Start(addr)
	}()

	if err := <-errCh; err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
