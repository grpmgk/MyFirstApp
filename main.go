package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	_ "github.com/lib/pq"
)

const (
	yandexClientID     = "9d38e25d2cf64d6897786576f95efe9f"
	yandexClientSecret = "3244501b154b47c69763e47b753cb3d2"
	yandexRedirectURI  = "http://localhost:8080/auth/yandex/callback"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	YandexID string `json:"yandex_id,omitempty"`
	VkID     string `json:"vk_id,omitempty"`
}

var db *sql.DB

func main() {
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
		yandex_id TEXT UNIQUE,
		email TEXT,
		name TEXT,
		created_at TIMESTAMP DEFAULT NOW()
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
			if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			var id int
			err := db.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id", u.Name).Scan(&id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			u.ID = id
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(u)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/auth/yandex/login", func(w http.ResponseWriter, r *http.Request) {
		// Генерируем случайный параметр state для защиты от CSRF
		state := generateRandomState()

		// Сохраняем state в куку (чтобы потом проверить)
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			HttpOnly: true,
			MaxAge:   600, // 10 минут
		})

		// Формируем URL для редиректа на Яндекс
		authURL := fmt.Sprintf(
			"https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s&state=%s",
			yandexClientID,
			url.QueryEscape(yandexRedirectURI),
			state,
		)

		// Перенаправляем пользователя на Яндекс
		http.Redirect(w, r, authURL, http.StatusFound)
	})

	http.HandleFunc("/auth/yandex/callback", func(w http.ResponseWriter, r *http.Request) {
		// Получаем code из параметров запроса
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		// Проверяем state (защита от CSRF)
		cookieState, err := r.Cookie("oauth_state")
		if err != nil || cookieState.Value != state {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		// Обмениваем code на токен
		// Формируем тело запроса (application/x-www-form-urlencoded)
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", code)
		data.Set("client_id", yandexClientID)
		data.Set("client_secret", yandexClientSecret)

		// Отправляем POST с телом
		resp, err := http.PostForm("https://oauth.yandex.ru/token", data)
		if err != nil {
			http.Error(w, "Token exchange failed", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Ответ Яндекса на /token: статус %s, тело: %s", resp.Status, string(bodyBytes))
		// нужно снова восстановить тело для декодирования
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		// Парсим ответ
		var tokenResp struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			log.Printf("Ошибка парсинга tokenResp: %v", err)
			http.Error(w, "Failed to parse token", http.StatusInternalServerError)
			return
		}
		log.Printf("Access token получен: %s", tokenResp.AccessToken) // <- добавить

		// Получаем информацию о пользователе
		userInfoURL := fmt.Sprintf("https://login.yandex.ru/info?format=json&oauth_token=%s", tokenResp.AccessToken)
		log.Printf("Запрос к /info: %s", userInfoURL) // <- добавить

		userResp, err := http.Get(userInfoURL)
		if err != nil {
			log.Printf("Ошибка GET /info: %v", err)
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}
		defer userResp.Body.Close()

		body, _ := io.ReadAll(userResp.Body)        // <- добавить
		log.Printf("Ответ /info: %s", string(body)) // <- добавить

		// Парсим информацию
		var yandexUser struct {
			ID        string `json:"id"`
			Email     string `json:"default_email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		}
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&yandexUser); err != nil {
			log.Printf("Ошибка парсинга JSON: %v", err)
			http.Error(w, "Failed to parse user info", http.StatusInternalServerError)
			return
		}

		log.Printf("Данные от Яндекса: ID=%s, Email=%s, Имя=%s %s", yandexUser.ID, yandexUser.Email, yandexUser.FirstName, yandexUser.LastName)

		// Сохраняем пользователя в БД
		userID, err := saveOrGetUserByYandex(yandexUser.ID, yandexUser.Email, yandexUser.FirstName+" "+yandexUser.LastName)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Устанавливаем сессию (простая кука с user_id)
		http.SetCookie(w, &http.Cookie{
			Name:     "user_id",
			Value:    fmt.Sprintf("%d", userID),
			HttpOnly: true,
			Path:     "/",
			MaxAge:   86400,
		})

		// Удаляем куку с state
		http.SetCookie(w, &http.Cookie{
			Name:   "oauth_state",
			Value:  "",
			MaxAge: -1,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	})

	http.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_id")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var user User
		err = db.QueryRow("SELECT id, name, email FROM users WHERE id = $1", cookie.Value).Scan(&user.ID, &user.Name, &user.Email)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	http.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "user_id",
			Value:  "",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/", http.StatusFound)
	})

	log.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func generateRandomState() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "fallback_state"
	}
	return base64.URLEncoding.EncodeToString(b)
}

func saveOrGetUserByYandex(yandexID, email, name string) (int, error) {
	log.Printf("===> saveOrGetUserByYandex: yandexID='%s', email='%s', name='%s'", yandexID, email, name)
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE yandex_id = $1", yandexID).Scan(&userID)
	if err == nil {
		log.Printf("Найден по yandex_id: %d", userID)
		return userID, nil
	}
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err == nil {
		log.Printf("Найден по email: %d, обновляем yandex_id", userID)
		_, err = db.Exec("UPDATE users SET yandex_id = $1 WHERE id = $2", yandexID, userID)
		return userID, err
	}
	log.Printf("Создаём нового пользователя")
	err = db.QueryRow("INSERT INTO users (yandex_id, email, name) VALUES ($1, $2, $3) RETURNING id", yandexID, email, name).Scan(&userID)
	if err != nil {
		log.Printf("Ошибка вставки: %v", err)
		return 0, err
	}
	log.Printf("Создан ID: %d", userID)
	return userID, nil
}
