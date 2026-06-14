package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
)

func main() {
	port := 7540

	// Используем переменную окружения TODO_PORT
	envPort := os.Getenv("TODO_PORT")
	if envPort != "" {
		p, err := strconv.Atoi(envPort)
		if err == nil {
			port = p
		}
	}

	webDir := "./web"

	// Простейший файловый сервер: отдаёт всё из ./web по соответствующим путям
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server starting on %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
	}
}
