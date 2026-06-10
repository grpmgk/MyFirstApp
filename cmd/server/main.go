package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"bechend-test/internal/handlers"
	"bechend-test/internal/storage"
)

func main() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	if err := storage.InitDB(connStr); err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer storage.DB.Close()

	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetUsers(w, r)
		} else if r.Method == http.MethodPost {
			handlers.CreateUser(w, r)
		}
	})

	http.HandleFunc("/", handlers.Home)
	http.HandleFunc("/auth/yandex/login", handlers.YandexLogin)
	http.HandleFunc("/auth/yandex/callback", handlers.YandexCallback)
	http.HandleFunc("/api/me", handlers.Me)
	http.HandleFunc("/auth/logout", handlers.Logout)

	log.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
