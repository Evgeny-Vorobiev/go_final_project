package api

import (
	"net/http"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/auth"
)

func Init() {
	// Статика
	http.HandleFunc("/", serveStatic)

	// API
	http.HandleFunc("/api/nextdate", nextDateHandler)
	http.HandleFunc("/api/signin", SigninHandler)

	// /api/task — POST (создать), PUT (обновить)
	http.HandleFunc("/api/task", authMiddleware(taskHandler))

	// /api/tasks — GET (список + поиск)
	http.HandleFunc("/api/tasks", authMiddleware(tasksHandler))

	// /api/task/get?id= — GET (получить одну задачу)
	http.HandleFunc("/api/task/get", authMiddleware(getTaskWrapper))

	// /api/task/done — POST (выполнить: delete/move)
	http.HandleFunc("/api/task/done", authMiddleware(doneHandler))

	// /api/task/delete?id= — POST (удалить)
	http.HandleFunc("/api/task/delete", authMiddleware(deleteWrapper))
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return auth.Auth(next)
}

func getTaskWrapper(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	TaskGetHandler(w, r)
}

func deleteWrapper(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	TaskDeleteHandler(w, r)
}

func doneHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	TaskDoneHandler(w, r)
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func nextDateHandler(w http.ResponseWriter, r *http.Request) {
	// логика в nextdate.go
}

func taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		TaskPostHandler(w, r)
	case http.MethodPut:
		TaskPutHandler(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func tasksHandler(w http.ResponseWriter, r *http.Request) {
	TasksHandler(w, r)
}
