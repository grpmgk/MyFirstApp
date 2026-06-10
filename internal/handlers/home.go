package handlers

import "net/http"

func Home(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "internal/web/index.html")
}
