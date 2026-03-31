package handlers

// TODO: start with /registration endpoint and set up firebase before continuing

/*
type Handler struct {
	Client *firestore.Client
} */

/*
func (h *Handler) getDashboardConfig(w http.ResponseWriter, r *http.Request) {
	log.Printf("Recieved %s request", r.Method)

	id := r.PathValue("id")
	ctx := r.Context()

	w.Header().Set("Content-Type", "application/json")

	doc, err := h.Client.Collection(utility.CollectionName).Doc(id).Get(ctx)
	if err != nil {
		log.Printf("Error getting document %s: %v", id, err)
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(doc.Data())
}

func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.ToLower(strings.TrimSpace(r.PathValue("id")))

	if id == "" {
		http.Error(w, "missing dashboard ID", http.StatusBadRequest)
		return
	}

	cfg, err := h.getDashboardConfig(w, r)
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusNotFound)
		return
	}

} */
