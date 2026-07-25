package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // драйвер SQLite для database/sql
)

// schema определяет структуру таблицы scheduler и индекс по дате.
const schema = `
CREATE TABLE IF NOT EXISTS scheduler (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date CHAR(8) NOT NULL DEFAULT '',
    title VARCHAR(255) NOT NULL DEFAULT '',
    comment TEXT NOT NULL DEFAULT '',
    repeat VARCHAR(128) NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_scheduler_date ON scheduler (date);
`

// Init открывает или создаёт файл базы SQLite, выполняет schema и сохраняет соединение в глобальную переменную DB.
// Путь к файлу БД может быть переопределён через переменную окружения TODO_DBFILE.
func Init(dbFile string) error {
	if envPath := os.Getenv("TODO_DBFILE"); envPath != "" {
		dbFile = envPath
	}

	absPath, err := filepath.Abs(dbFile)
	if err != nil {
		return fmt.Errorf("не удалось получить абсолютный путь к БД: %w", err)
	}

	db, err := sql.Open("sqlite", absPath)
	if err != nil {
		return fmt.Errorf("ошибка открытия соединения: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("ошибка пинга БД: %w", err)
	}

	if _, execErr := db.Exec(schema); execErr != nil {
		db.Close()
		return fmt.Errorf("ошибка выполнения схемы БД: %w", execErr)
	}

	DB = db
	return nil
}

// Close закрывает глобальное соединение с БД.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
