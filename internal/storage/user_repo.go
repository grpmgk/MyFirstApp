package storage

import (
	"log"
)

// SaveOrGetUserByYandex создаёт или находит пользователя по yandexID и email
func SaveOrGetUserByYandex(yandexID, email, name string) (int, error) {
	log.Printf("===> SaveOrGetUserByYandex: yandexID='%s', email='%s', name='%s'", yandexID, email, name)
	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE yandex_id = $1", yandexID).Scan(&userID)
	if err == nil {
		log.Printf("Найден по yandex_id: %d", userID)
		return userID, nil
	}
	err = DB.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err == nil {
		log.Printf("Найден по email: %d, обновляем yandex_id", userID)
		_, err = DB.Exec("UPDATE users SET yandex_id = $1 WHERE id = $2", yandexID, userID)
		return userID, err
	}
	log.Printf("Создаём нового пользователя")
	err = DB.QueryRow("INSERT INTO users (yandex_id, email, name) VALUES ($1, $2, $3) RETURNING id", yandexID, email, name).Scan(&userID)
	if err != nil {
		log.Printf("Ошибка вставки: %v", err)
		return 0, err
	}
	log.Printf("Создан ID: %d", userID)
	return userID, nil
}
