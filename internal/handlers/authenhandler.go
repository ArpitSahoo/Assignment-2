package handlers

import "net/http"

func (h *Handler) HandleAuthenticationReq(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		println("Authentication request received")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
