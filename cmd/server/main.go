package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/api"
	"github.com/Evgeny-Vorobiev/go_final_project/pkg/db"
)

func main() {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "scheduler.db"
	}

	if err := db.Init(dbFile); err != nil {
		log.Fatalf("failed to init DB: %v", err)
	}

	api.Init()

	addr := ":" + port
	log.Printf("server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
