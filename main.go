package main

import (
	"log"
	"os"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/db"
	"github.com/Evgeny-Vorobiev/go_final_project/pkg/server"
)

func main() {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "8080"
	}

	// Поддержка переменной окружения TODO_DBFILE
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "scheduler.db"
	}

	log.Printf("using database file: %s", dbFile)

	if err := db.Init(dbFile); err != nil {
		log.Fatalf("failed to init DB: %v", err)
	}

	log.Printf("starting server on port %s", port)
	if err := server.Run(port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
