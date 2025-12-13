package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/shinyyama/hackathon-backend/internal/config"
	"github.com/shinyyama/hackathon-backend/internal/db"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/server"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	srv := server.New(nil)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	errCh := make(chan error, 1)

	go func() {
		log.Printf("starting server on %s", addr)
		errCh <- srv.Start(addr)
	}()

	go func() {
		cfg, err := config.Load()
		if err != nil {
			log.Printf("config load error: %v", err)
			return
		}
		conn, err := db.Connect(cfg)
		if err != nil {
			log.Printf("db connect error: %v", err)
			return
		}
		if setter, ok := interface{}(srv).(interface{ SetDB(*gorm.DB) }); ok {
			setter.SetDB(conn)
		} else {
			log.Printf("server does not support SetDB; skipping DB injection")
		}
		if err := conn.AutoMigrate(&model.Item{}); err != nil {
			log.Printf("auto migrate error: %v", err)
		}
	}()

	if err := <-errCh; err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
