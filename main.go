package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/api"
	"github.com/Evgeny-Vorobiev/go_final_project/pkg/db"
)

// main инициализирует базу данных, регистрирует API-обработчики и запускает HTTP-сервер.
// Настройки порта и пути к БД берутся из переменных окружения TODO_PORT и TODO_DBFILE.
func main() {

	// Определяем порт сервера: по умолчанию 7540, можно переопределить через TODO_PORT
	port := "7540"
	if p := os.Getenv("TODO_PORT"); p != "" {
		port = p
	}

	// Определяем путь к файлу базы данных SQLite: по умолчанию scheduler.db
	dbFile := "scheduler.db"
	if f := os.Getenv("TODO_DBFILE"); f != "" {
		dbFile = f
	}

	// Инициализация подключения к БД и создание таблицы scheduler при необходимости
	if err := db.Init(dbFile); err != nil {
		log.Fatalf("ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	// Регистрация всех маршрутов REST API (включая middleware аутентификации)
	api.Init()

	// Обслуживание статических файлов фронтенда из директории web
	webDir, err := filepath.Abs("web")
	if err != nil {
		log.Fatalf("не удалось определить путь к web: %v", err)
	}
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	// Запуск сервера
	addr := fmt.Sprintf(":%s", port)
	log.Printf("сервер запущен на %s, БД: %s, web: %s", addr, dbFile, webDir)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("ошибка запуска сервера: %v", err)
	}
}
