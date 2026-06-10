package handlers

import (
	"bechend-test/internal/config"
	"bechend-test/internal/storage"
	"bechend-test/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

func YandexLogin(w http.ResponseWriter, r *http.Request) {
	state := utils.GenerateRandomState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		MaxAge:   600,
	})
	authURL := fmt.Sprintf(
		"https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s&state=%s",
		config.YandexClientID,
		url.QueryEscape(config.YandexRedirectURI),
		state,
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func YandexCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "No code received", http.StatusBadRequest)
		return
	}
	cookieState, err := r.Cookie("oauth_state")
	if err != nil || cookieState.Value != state {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}
	// Обмен code на token
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", config.YandexClientID)
	data.Set("client_secret", config.YandexClientSecret)

	resp, err := http.PostForm("https://oauth.yandex.ru/token", data)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("Ответ /token: статус %s, тело: %s", resp.Status, string(bodyBytes))
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Printf("Ошибка парсинга tokenResp: %v", err)
		http.Error(w, "Failed to parse token", http.StatusInternalServerError)
		return
	}
	log.Printf("Access token получен: %s", tokenResp.AccessToken)

	userInfoURL := fmt.Sprintf("https://login.yandex.ru/info?format=json&oauth_token=%s", tokenResp.AccessToken)
	userResp, err := http.Get(userInfoURL)
	if err != nil {
		log.Printf("Ошибка GET /info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()
	body, _ := io.ReadAll(userResp.Body)
	log.Printf("Ответ /info: %s", string(body))

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

	userID, err := storage.SaveOrGetUserByYandex(yandexUser.ID, yandexUser.Email, yandexUser.FirstName+" "+yandexUser.LastName)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "user_id",
		Value:    fmt.Sprintf("%d", userID),
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400,
	})
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: "", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}
