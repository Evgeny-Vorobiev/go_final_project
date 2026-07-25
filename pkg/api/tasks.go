package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/db"
)

// TasksResp — структура ответа на запрос списка задач.
type TasksResp struct {
	Tasks []*db.Task `json:"tasks"`
}

// tasksHandler обрабатывает GET /api/tasks?search=...&limit=...
// Возвращает список задач, отфильтрованных по search (подстрока в названии/комментарии или дата в формате ДД.ММ.ГГГГ),
// с ограничением limit (по умолчанию 50).
func tasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	search := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")

	var limit int
	var err error
	if limitStr == "" {
		limit = 50
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			log.Printf("Неверный параметр limit: %s", limitStr)
			http.Error(w, "неверный параметр limit", http.StatusBadRequest)
			return
		}
	}

	// Проверка длины search (безопасность)
	if len(search) > 255 {
		http.Error(w, "Параметр search слишком длинный", http.StatusBadRequest)
		return
	}

	// Если search является датой в формате ДД.ММ.ГГГГ, преобразуем в YYYYMMDD для поиска по БД
	if isDate, _ := isValidDate(search); isDate {
		searchDate, _ := time.Parse("02.01.2006", search)
		search = searchDate.Format("20060102")
	}

	tasks, err := db.GetTasksWithSearch(search, limit)
	if err != nil {
		log.Printf("Ошибка при получении задач: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := TasksResp{Tasks: tasks}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Ошибка сериализации JSON: %v", err)
		http.Error(w, "ошибка сериализации JSON", http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

// isValidDate проверяет, является ли строка датой в формате ДД.ММ.ГГГГ.
func isValidDate(date string) (bool, error) {
	_, err := time.Parse("02.01.2006", date)
	if err != nil {
		return false, fmt.Errorf("неверный формат даты: %w", err)
	}
	return true, nil
}
