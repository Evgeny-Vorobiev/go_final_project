package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// DB — глобальное соединение с базой данных SQLite.
// Инициализируется функцией Init() из пакета db.
var DB *sql.DB

// Task представляет собой задачу планировщика задач.
// Поля соответствуют структуре таблицы scheduler.
type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// Tasks возвращает список задач с учётом параметра поиска.
// Если search пуст — возвращаются все задачи (упорядочены по дате).
// Если search соответствует формату ДД.ММ.ГГГГ — выполняется точный поиск по этой дате (преобразованной в YYYYMMDD).
// Иначе — регистронезависимый поиск по подстроке в полях title и comment (через COLLATE NOCASE).
// limit ограничивает количество возвращаемых записей.
func Tasks(limit int, search string) ([]*Task, error) {
	var rows *sql.Rows
	var err error

	dateRegex := regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`)

	if search == "" {
		// Запрос всех задач с сортировкой по дате
		query := `SELECT id, title, date, comment, repeat FROM scheduler ORDER BY date LIMIT ?`
		rows, err = DB.Query(query, limit)
	} else if dateRegex.MatchString(search) {
		// Поиск по точной дате: преобразуем ДД.ММ.ГГГГ в YYYYMMDD
		t, err := time.Parse("02.01.2006", search)
		if err != nil {
			return nil, fmt.Errorf("некорректная дата: %w", err)
		}
		dateStr := t.Format("20060102")
		query := `SELECT id, title, date, comment, repeat FROM scheduler WHERE date = ? ORDER BY date LIMIT ?`
		rows, err = DB.Query(query, dateStr, limit)
	} else {
		// Поиск по подстроке в заголовке или комментарии (регистронезависимый, COLLATE NOCASE)
		pattern := "%" + search + "%"
		query := `
			SELECT id, title, date, comment, repeat
			FROM scheduler
			WHERE title LIKE ? COLLATE NOCASE OR comment LIKE ? COLLATE NOCASE
			ORDER BY date
			LIMIT ?`
		rows, err = DB.Query(query, pattern, pattern, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Task
	for rows.Next() {
		var id int64
		var title, date, comment, repeat string

		err := rows.Scan(&id, &title, &date, &comment, &repeat)
		if err != nil {
			return nil, err
		}

		result = append(result, &Task{
			ID:      strconv.FormatInt(id, 10),
			Title:   title,
			Date:    date,
			Comment: comment,
			Repeat:  repeat,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// AddTask добавляет новую задачу в таблицу scheduler.
// Возвращает ID созданной записи или ошибку.
func AddTask(task *Task) (int64, error) {
	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
	res, err := DB.Exec(query, task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetTasksWithSearch возвращает список задач, отфильтрованных по параметру search.
// Если search пуст — возвращает все задачи, отсортированные по дате и ID.
// Если search соответствует формату YYYYMMDD — точный поиск по дате.
// Иначе — поиск по подстроке в title и comment.
// limit задаёт максимальное количество записей (по умолчанию 50, если limit <= 0).
func GetTasksWithSearch(search string, limit int) ([]*Task, error) {
	if limit <= 0 {
		limit = 50
	}

	var query string
	var params []interface{}

	if search == "" {
		query = `SELECT id, title, date, comment, repeat
                 FROM scheduler
                 ORDER BY date ASC, id ASC
                 LIMIT ?`
		params = []interface{}{limit}
	} else {
		// Проверяем, является ли search датой в формате YYYYMMDD
		if _, err := time.Parse("20060102", search); err == nil {
			query = `SELECT id, title, date, comment, repeat
                     FROM scheduler
                     WHERE date = ?
                     ORDER BY date ASC, id ASC
                     LIMIT ?`
			params = []interface{}{search, limit}
		} else {
			// Поиск по подстроке в title и comment (регистрозависимый, без COLLATE NOCASE)
			likeSearch := fmt.Sprintf("%%%s%%", search)
			query = `SELECT id, title, date, comment, repeat
                     FROM scheduler
                     WHERE title LIKE ? OR comment LIKE ?
                     ORDER BY date ASC, id ASC
                     LIMIT ?`
			params = []interface{}{likeSearch, likeSearch, limit}
		}
	}

	rows, err := DB.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки задач: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		if err := rows.Scan(&t.ID, &t.Title, &t.Date, &t.Comment, &t.Repeat); err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		tasks = append(tasks, t)
	}

	if tasks == nil {
		tasks = []*Task{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке результатов: %w", rows.Err())
	}

	return tasks, nil
}

// GetTask возвращает задачу по её идентификатору.
// Если задача не найдена, возвращает ошибку.
func GetTask(id string) (*Task, error) {
	query := `SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`
	row := DB.QueryRow(query, id)

	task := &Task{}
	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return nil, fmt.Errorf("задача не найдена")
	}
	return task, nil
}

// UpdateTask обновляет поля существующей задачи.
// Если задача с указанным id не найдена, возвращает ошибку.
func UpdateTask(task *Task) error {
	query := `UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`
	res, err := DB.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("задача не найдена")
	}
	return nil
}

// DeleteTask удаляет задачу по её ID.
// Если задача не найдена, возвращает ошибку.
func DeleteTask(id string) error {
	query := `DELETE FROM scheduler WHERE id = ?`
	res, err := DB.Exec(query, id)
	if err != nil {
		return err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("задача не найдена")
	}
	return nil
}

// UpdateDate обновляет только поле date у задачи с указанным ID.
// Используется для переноса периодической задачи на следующую дату.
func UpdateDate(newDate string, id string) error {
	query := `UPDATE scheduler SET date = ? WHERE id = ?`
	_, err := DB.Exec(query, newDate, id)
	return err
}
