package handlers

import (
	"bechend-test/internal/models"
	"bechend-test/internal/storage"
	"encoding/json"
	"net/http"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := storage.DB.Query("SELECT id, name FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.Name)
		users = append(users, u)
	}
	json.NewEncoder(w).Encode(users)
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	var id int
	err := storage.DB.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id", u.Name).Scan(&id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.ID = id
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}
