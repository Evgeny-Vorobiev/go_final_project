package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

var db *sql.DB

const schema = `
CREATE TABLE IF NOT EXISTS scheduler (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date CHAR(8) NOT NULL DEFAULT '',
    title VARCHAR(255) NOT NULL,
    comment TEXT,
    repeat VARCHAR(128)
);

CREATE INDEX IF NOT EXISTS idx_scheduler_date ON scheduler(date);
`

// Init инициализирует БД: проверяет файл, открывает, при необходимости создаёт схему
func Init(dbFile string) error {
	// Проверяем существование файла
	_, err := os.Stat(dbFile)
	install := err != nil

	var dsn string
	// modernc.org/sqlite использует путь к файлу как DSN
	dsn = fmt.Sprintf("file:%s?_foreign_keys=off&_busy_timeout=5000", dbFile)

	dbConn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Важно: делаем пинг, чтобы соединение реально установилось и мы могли проверить ошибки
	if err := dbConn.Ping(); err != nil {
		return fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	if install {
		// Если файла не было — выполняем создание таблицы и индекса
		if _, err := dbConn.Exec(schema); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	}

	db = dbConn
	return nil
}

// GetDB возвращает глобальный экземпляр БД (чтобы другие пакеты могли с ним работать)
func GetDB() *sql.DB {
	return db
}
