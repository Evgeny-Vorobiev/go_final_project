package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// secretKey — HMAC-ключ для подписи JWT, вычисляется из хэша пароля при инициализации.
var secretKey string

// claims — структура полезной нагрузки JWT-токена.
type claims struct {
	jwt.RegisteredClaims
	PasswordHash string `json:"password_hash"`
}

// InitAuth вычисляет secretKey на основе значения переменной окружения TODO_PASSWORD.
// Если пароль не задан, secretKey остаётся пустым — аутентификация отключена.
func InitAuth() {
	if pass := os.Getenv("TODO_PASSWORD"); pass != "" {
		hash := sha256.Sum256([]byte(pass))
		secretKey = hex.EncodeToString(hash[:])
	}
}

// signinHandler обрабатывает POST /api/signin.
// Проверяет пароль из тела запроса (поле password) со значением TODO_PASSWORD.
// При совпадении формирует JWT с хэшем пароля и сроком действия 8 часов.
func signinHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "ошибка десериализации JSON")
		return
	}

	pass := os.Getenv("TODO_PASSWORD")
	if pass == "" {
		writeError(w, "аутентификация не настроена")
		return
	}

	if req.Password != pass {
		writeError(w, "Неверный пароль")
		return
	}

	hash := sha256.Sum256([]byte(pass))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
		},
		PasswordHash: hex.EncodeToString(hash[:]),
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		writeError(w, "ошибка формирования токена")
		return
	}

	writeJSON(w, map[string]string{"token": tokenString})
}

// auth — middleware для проверки JWT-аутентификации.
// Если переменная окружения TODO_PASSWORD пуста, пропускает запрос без проверки.
// Иначе извлекает JWT из куки "token" и проверяет:
//   - подпись (secretKey)
//   - срок действия
//   - соответствие хэша пароля в payload текущему значению TODO_PASSWORD
//
// При неудаче возвращает HTTP 401.
func auth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pass := os.Getenv("TODO_PASSWORD")
		if pass == "" {
			next(w, r)
			return
		}

		// Получаем JWT из куки
		cookie, err := r.Cookie("token")
		if err != nil || cookie.Value == "" {
			http.Error(w, "Authentification required", http.StatusUnauthorized)
			return
		}

		tokenStr := cookie.Value
		hash := sha256.Sum256([]byte(pass))
		expectedHash := hex.EncodeToString(hash[:])

		parsedToken, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			http.Error(w, "Authentification required", http.StatusUnauthorized)
			return
		}

		cl, ok := parsedToken.Claims.(*claims)
		if !ok || !parsedToken.Valid || cl.PasswordHash != expectedHash {
			http.Error(w, "Authentification required", http.StatusUnauthorized)
			return
		}

		next(w, r)
	})
}
