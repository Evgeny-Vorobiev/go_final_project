package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/db"
)

type Task struct {
	ID      int    `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// sendJSON теперь только здесь (или в api.go). Если хочешь, можно перенести в api.go — главное, чтобы был ОДИН раз.
func sendJSON(w http.ResponseWriter, v any) {
	b, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(b)
}

func TasksHandler(w http.ResponseWriter, r *http.Request) {
	limit := 50
	search := r.URL.Query().Get("search")

	rows, err := buildTasksQuery(search, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}

	sendJSON(w, tasks)
}

func buildTasksQuery(search string, limit int) (*sql.Rows, error) {
	query := "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?"
	args := []any{limit}

	if search != "" {
		if len(search) == 10 && search[2] == '.' && search[5] == '.' {
			parts := strings.Split(search, ".")
			if len(parts) == 3 {
				y, m, d := parts[2], parts[1], parts[0]
				dateStr := y + m + d
				query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? ORDER BY date LIMIT ?"
				args = []any{dateStr, limit}
				return db.GetDB().Query(query, args...)
			}
		}
		like := "%" + search + "%"
		query = `SELECT id, date, title, comment, repeat FROM scheduler
			WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?`
		args = []any{like, like, limit}
	}

	return db.GetDB().Query(query, args...)
}

func TaskGetHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		sendJSON(w, map[string]string{"error": "invalid id"})
		return
	}

	row := db.GetDB().QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id)
	t := &Task{}
	err = row.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, t)
}

func TaskPostHandler(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		sendJSON(w, map[string]string{"error": "invalid JSON"})
		return
	}

	_, err := time.Parse("20060102", task.Date)
	if err != nil {
		sendJSON(w, map[string]string{"error": "invalid date format, use YYYYMMDD"})
		return
	}

	res, err := db.GetDB().Exec(
		"INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
		task.Date, task.Title, task.Comment, task.Repeat,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	sendJSON(w, map[string]any{"id": id})
}

func TaskPutHandler(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		sendJSON(w, map[string]string{"error": "invalid JSON"})
		return
	}

	if task.ID <= 0 {
		sendJSON(w, map[string]string{"error": "missing or invalid id"})
		return
	}

	_, err := db.GetDB().Exec(
		"UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?",
		task.Date, task.Title, task.Comment, task.Repeat, task.ID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, map[string]string{"status": "ok"})
}

func TaskDoneHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID     int    `json:"id"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSON(w, map[string]string{"error": "invalid JSON"})
		return
	}

	if payload.ID <= 0 {
		sendJSON(w, map[string]string{"error": "missing or invalid id"})
		return
	}

	row := db.GetDB().QueryRow(
		"SELECT id, date, repeat FROM scheduler WHERE id=?", payload.ID,
	)
	var id int
	var dateStr, repeatRule string
	err := row.Scan(&id, &dateStr, &repeatRule)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	nextDateStr, err := NextDate(now, dateStr, repeatRule)
	if err != nil && payload.Action == "move" {
		sendJSON(w, map[string]string{"error": "cannot compute next date"})
		return
	}

	tx, err := db.GetDB().Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	if payload.Action == "delete" || repeatRule == "" {
		_, err = tx.Exec("DELETE FROM scheduler WHERE id=?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, err = tx.Exec("UPDATE scheduler SET date=? WHERE id=?", nextDateStr, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, map[string]string{"status": "ok"})
}

func TaskDeleteHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		sendJSON(w, map[string]string{"error": "invalid id"})
		return
	}

	res, err := db.GetDB().Exec("DELETE FROM scheduler WHERE id=?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.NotFound(w, r)
		return
	}

	sendJSON(w, map[string]string{"status": "ok"})
}
