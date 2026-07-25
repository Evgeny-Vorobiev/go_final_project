package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/db"
)

const dateFormat = "20060102" // формат даты: YYYYMMDD

// writeJSON отправляет клиенту JSON-ответ с заголовком Content-Type: application/json.
func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writeError отправляет клиенту JSON-сообщение об ошибке в поле error.
func writeError(w http.ResponseWriter, msg string) {
	writeJSON(w, map[string]string{"error": msg})
}

// truncateToDate обнуляет часы, минуты, секунды и наносекунды, оставляя только календарную дату.
func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// addTaskHandler обрабатывает POST /api/task — добавление новой задачи.
// Проверяет заголовок, корректирует дату: если дата не указана — ставится сегодня,
// если дата в прошлом и нет правила повторения — переносится на сегодня,
// если дата в прошлом и есть правило — вычисляется следующая дата по правилу.
func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task db.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, "ошибка десериализации JSON")
		return
	}

	if task.Title == "" {
		writeError(w, "не указан заголовок задачи")
		return
	}

	now := time.Now()
	todayStr := truncateToDate(now).Format(dateFormat)

	// Если дата не задана, используем сегодняшнюю
	if task.Date == "" {
		task.Date = todayStr
	} else {
		// Пытаемся распарсить дату
		t, err := time.Parse(dateFormat, task.Date)
		if err != nil {
			writeError(w, "дата указана в неверном формате")
			return
		}
		t = truncateToDate(t)

		// Если указанная дата уже прошла (по календарю), корректируем
		if t.Before(truncateToDate(now)) {
			if task.Repeat == "" {
				// Без повтора — переносим на сегодня
				task.Date = todayStr
			} else {
				// С повтором — вычисляем следующую дату по правилу
				next, err := NextDate(now, task.Date, task.Repeat)
				if err != nil || next == "" {
					writeError(w, "недопустимое правило повторения")
					return
				}
				task.Date = next
			}
		}
		// Если дата >= сегодня, оставляем как есть (пользователь явно указал будущую дату)
	}

	// Дополнительная валидация правила повторения (даже если дата не корректировалась)
	if task.Repeat != "" {
		result, err := NextDate(now, task.Date, task.Repeat)
		if err != nil || result == "" {
			writeError(w, "недопустимое правило повторения")
			return
		}
	}

	id, err := db.AddTask(&task)
	if err != nil {
		writeError(w, "ошибка добавления задачи в базу данных")
		return
	}

	writeJSON(w, map[string]string{"id": strconv.FormatInt(id, 10)})
}

// getTaskHandler обрабатывает GET /api/task?id=<id> — получение задачи по идентификатору.
func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, "Не указан идентификатор")
		return
	}

	task, err := db.GetTask(id)
	if err != nil {
		writeError(w, "Задача не найдена")
		return
	}

	writeJSON(w, task)
}

// updateTaskHandler обрабатывает PUT /api/task — обновление существующей задачи.
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task struct {
		ID      string `json:"id"`
		Date    string `json:"date"`
		Title   string `json:"title"`
		Comment string `json:"comment"`
		Repeat  string `json:"repeat"`
	}

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, "ошибка десериализации JSON")
		return
	}

	if task.ID == "" {
		writeError(w, "Не указан идентификатор")
		return
	}
	if task.Title == "" {
		writeError(w, "не указан заголовок задачи")
		return
	}
	if task.Date == "" {
		writeError(w, "не указана дата")
		return
	}
	if _, err := time.Parse(dateFormat, task.Date); err != nil {
		writeError(w, "дата указана в неверном формате")
		return
	}

	// Валидация правила повторения (если указано)
	if task.Repeat != "" {
		now := time.Now()
		result, err := NextDate(now, task.Date, task.Repeat)
		if err != nil || result == "" {
			writeError(w, "недопустимое правило повторения")
			return
		}
	}

	if err := db.UpdateTask(&db.Task{
		ID:      task.ID,
		Date:    task.Date,
		Title:   task.Title,
		Comment: task.Comment,
		Repeat:  task.Repeat,
	}); err != nil {
		writeError(w, "Задача не найдена")
		return
	}

	writeJSON(w, map[string]string{})
}

// handleDoneTask обрабатывает POST /api/task/done?id=<id> — отметка задачи как выполненной.
// Для одноразовых задач (repeat == "") удаляет задачу.
// Для периодических задач вычисляет следующую дату через NextDate и обновляет поле date.
func handleDoneTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, "Не указан идентификатор")
		return
	}

	task, err := db.GetTask(id)
	if err != nil {
		writeError(w, "Задача не найдена")
		return
	}

	if task.Repeat == "" {
		// Одноразовая задача — удаляем
		if err := db.DeleteTask(id); err != nil {
			writeError(w, "Ошибка при удалении задачи")
			return
		}
	} else {
		// Периодическая задача — переносим на следующую дату
		now := time.Now()
		nextDate, err := NextDate(now, task.Date, task.Repeat)
		if err != nil || nextDate == "" {
			writeError(w, "недопустимое правило повторения")
			return
		}
		if err := db.UpdateDate(nextDate, id); err != nil {
			writeError(w, "Ошибка при обновлении задачи")
			return
		}
	}

	writeJSON(w, map[string]string{})
}

// handleDeleteTask обрабатывает DELETE /api/task?id=<id> — удаление задачи.
func handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, "Не указан идентификатор")
		return
	}

	// Проверка существования задачи перед удалением (чтобы вернуть понятную ошибку)
	if _, err := db.GetTask(id); err != nil {
		writeError(w, "Задача не найдена")
		return
	}

	if err := db.DeleteTask(id); err != nil {
		writeError(w, "Ошибка при удалении задачи")
		return
	}

	writeJSON(w, map[string]string{})
}
