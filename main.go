package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var db *sql.DB

func main() {
	// Читаем переменные окружения (их пропишем в docker-compose.yml)
	host := os.Getenv("DB_HOST") // имя сервиса в compose = "db"
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := "host=" + host + " port=" + port + " user=" + user +
		" password=" + password + " dbname=" + dbname + " sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer db.Close()

	// Создаём таблицу, если её нет (упрощённо)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL
	)`)

	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rows, err := db.Query("SELECT id, name FROM users")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var users []User
			for rows.Next() {
				var u User
				rows.Scan(&u.ID, &u.Name)
				users = append(users, u)
			}
			json.NewEncoder(w).Encode(users)
		} else if r.Method == http.MethodPost {
			var u User
			json.NewDecoder(r.Body).Decode(&u)
			var id int
			db.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id", u.Name).Scan(&id)
			u.ID = id
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(u)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	log.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
