package models

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	YandexID string `json:"yandex_id,omitempty"`
	VkID     string `json:"vk_id,omitempty"`
}
