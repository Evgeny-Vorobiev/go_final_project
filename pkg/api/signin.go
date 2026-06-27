package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Evgeny-Vorobiev/go_final_project/pkg/auth"
	_ "github.com/Evgeny-Vorobiev/go_final_project/pkg/auth"
)

type SigninRequest struct {
	Password string `json:"password"`
}
type SigninResponse struct {
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

func SigninHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var req SigninRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, SigninResponse{Error: "invalid JSON"})
		return
	}

	expected := os.Getenv("TODO_PASSWORD")
	if expected == "" || req.Password != expected {
		sendJSON(w, SigninResponse{Error: "Invalid password"})
		return
	}

	token := auth.GenerateToken(req.Password)
	sendJSON(w, SigninResponse{Token: token})
}
