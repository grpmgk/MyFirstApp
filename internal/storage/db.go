package storage

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(connStr string) error {
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        yandex_id TEXT UNIQUE,
        email TEXT,
        name TEXT,
        created_at TIMESTAMP DEFAULT NOW()
    )`)
	if err != nil {
		log.Printf("Ошибка создания таблицы: %v", err)
	}
	return nil
}
