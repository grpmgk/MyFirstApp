package handlers

import (
	"bechend-test/internal/models"
	"bechend-test/internal/storage"
	"encoding/json"
	"net/http"
)

func Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var user models.User
	err = storage.DB.QueryRow("SELECT id, name, email FROM users WHERE id = $1", cookie.Value).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.Redirect(w, r, "/", http.StatusFound)
}
