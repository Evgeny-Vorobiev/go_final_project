package server

import (
	"fmt"
	"net/http"
)

// Run запускает HTTP-сервер на указанном порту и раздаёт статику из папки web
func Run(port string) error {
	fs := http.FileServer(http.Dir("web"))

	mux := http.NewServeMux()
	mux.Handle("/", fs)

	// Сюда позже будем добавлять API-роуты, например:
	// mux.HandleFunc("/api/parcels", api.ListParcels)

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("serving static files from 'web' on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}
