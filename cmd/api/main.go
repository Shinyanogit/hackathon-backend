package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// Temporary minimal server for Cloud Run startup debugging.
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	log.Printf("starting minimal debug server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
