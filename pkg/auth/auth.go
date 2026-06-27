package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
)

func GenerateToken(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

func ValidateToken(token, password string) bool {
	return token == GenerateToken(password)
}

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pass := os.Getenv("TODO_PASSWORD")
		// Если пароль не задан — пропускаем без проверки
		if pass == "" {
			next(w, r)
			return
		}

		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		token := cookie.Value
		if !ValidateToken(token, pass) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	})
}
