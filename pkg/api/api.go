package api

import "net/http"

// Init регистрирует обработчики для всех конечных точек REST API.
// Для защищённых эндпоинтов (/api/task, /api/task/done, /api/tasks) применяется middleware auth.
// Эндпоинт /api/signin доступен без аутентификации.
func Init() {
	InitAuth() // инициализация JWT-секрета на основе пароля из переменной окружения

	http.HandleFunc("/api/nextdate", handleNextDate)
	http.HandleFunc("/api/task", auth(taskHandler))         // POST, GET, PUT, DELETE — защищено
	http.HandleFunc("/api/task/done", auth(handleDoneTask)) // отметка о выполнении — защищено
	http.HandleFunc("/api/tasks", auth(tasksHandler))       // список задач — защищено
	http.HandleFunc("/api/signin", signinHandler)           // вход без аутентификации
}

// taskHandler — единый обработчик для /api/task, который делегирует выполнение
// в зависимости от HTTP-метода.
func taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getTaskHandler(w, r)
	case http.MethodPost:
		addTaskHandler(w, r)
	case http.MethodPut:
		updateTaskHandler(w, r)
	case http.MethodDelete:
		handleDeleteTask(w, r)
	default:
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
	}
}
