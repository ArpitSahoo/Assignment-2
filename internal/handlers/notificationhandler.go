package handlers

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// WebhookHandler handles requests to the /notifications endpoint,
// allowing clients to register new webhooks and retrieve all registered webhooks.
func (h *Handler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.registerWebhook(w, r)
	case http.MethodGet:
		h.getAllWebhooks(w, r)
	default:
		http.Error(w, "Method "+r.Method+" not supported for "+utility.NotificationPath, http.StatusMethodNotAllowed)
	}
}

// WebhookIDHandler handles requests to the /notifications/{id} endpoint,
// allowing clients to retrieve or delete a specific webhook by its ID.
func (h *Handler) WebhookIDHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	switch r.Method {
	case http.MethodGet:
		h.getWebhookByID(w, r, id)
	case http.MethodDelete:
		h.deleteWebhook(w, r, id)
	default:
		http.Error(w, "Method "+r.Method+" not supported for "+utility.NotificationPath, http.StatusMethodNotAllowed)
	}
}

// registerWebhook processes POST requests to create a new webhook.
// It validates the input, stores the webhook in Firestore, and returns the ID of the created webhook in the response.
func (h *Handler) registerWebhook(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(r.Body)

	var webhookReg models.RegisterWebhook
	if err := json.NewDecoder(r.Body).Decode(&webhookReg); err != nil {
		log.Println("Error decoding webhook registration request: ", err)
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if validateWebhook(w, webhookReg) {
		return
	}

	ref, _, err := h.Client.Collection(utility.WebhooksCollection).Add(r.Context(), map[string]any{
		"url":       webhookReg.Url,
		"country":   webhookReg.Country,
		"event":     webhookReg.Event,
		"threshold": webhookReg.Threshold,
	})

	if err != nil {
		log.Printf("Failed to add webhook: %v", err)
		http.Error(w, "Failed to add webhook", http.StatusInternalServerError)
		return
	}

	log.Println("Webhook created with ID: ", ref.ID)

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(models.WebhookResponse{
		ID: ref.ID,
	}); err != nil {
		log.Printf("Failed to encode webhook response: %v", err)
	}

}

// validateWebhook checks the validity of a webhook registration request.
// It ensures the URL is provided and the event type is valid.
// For THRESHOLD events, delegates threshold validation to validateThreshold.
// Returns true if validation fails, false if all checks pass.
func validateWebhook(w http.ResponseWriter, webhook models.RegisterWebhook) bool {
	if webhook.Url == "" {
		log.Println("Webhook URL is required")
		http.Error(w, "Webhook URL is required", http.StatusBadRequest)
		return true
	}

	validEvents := map[string]bool{
		"REGISTER": true, "CHANGE": true,
		"DELETE": true, "INVOKE": true, "THRESHOLD": true,
	}

	if !validEvents[webhook.Event] {
		log.Println("Error event must be either REGISTER, CHANGE, DELETE, INVOKE, THRESHOLD")
		http.Error(w, "Error event must be either REGISTER, CHANGE, DELETE, INVOKE, THRESHOLD",
			http.StatusBadRequest)
		return true
	}

	if webhook.Event != "THRESHOLD" && webhook.Threshold != nil {
		log.Println("Threshold block is only allowed for THRESHOLD event")
		http.Error(w, "Threshold block is only allowed for THRESHOLD event", http.StatusBadRequest)
		return true
	}

	if webhook.Event == "THRESHOLD" {
		return validateThreshold(w, webhook.Threshold)
	}

	return false
}

// validateThreshold validates the threshold block of a webhook registration request.
// Returns true if validation fails, false if it passes.
func validateThreshold(w http.ResponseWriter, threshold *models.Threshold) bool {
	if threshold == nil {
		log.Println("Error threshold block is required for THRESHOLD event")
		http.Error(w, "Error threshold block is required for THRESHOLD event", http.StatusBadRequest)
		return true
	}

	validFields := map[string]bool{
		"pm25": true, "pm10": true,
		"temperature": true, "precipitation": true,
	}
	if !validFields[threshold.Field] {
		log.Println("Error threshold field must be: pm25, pm10, temperature, or precipitation")
		http.Error(w, "error threshold field must be: pm25, pm10, temperature, or precipitation",
			http.StatusBadRequest)
		return true
	}

	validOperators := map[string]bool{">": true, "<": true, ">=": true, "<=": true, "=": true}
	if !validOperators[threshold.Operator] {
		log.Println("Error threshold operator must be >, <, >=, <= or =")
		http.Error(w, "error: threshold.operator must be >, <, >=, <= or =",
			http.StatusBadRequest)
		return true
	}

	if threshold.UpperOperator != "" {
		if !validOperators[threshold.UpperOperator] {
			log.Println("Error threshold upperOperator must be >, <, >=, <= or =")
			http.Error(w, "error: threshold.upperOperator must be >, <, >=, <= or =",
				http.StatusBadRequest)
			return true
		}
	}

	return false
}

// getAllWebhooks retrieves all registered webhooks from Firestore and returns them as a JSON array in the response.
// If there is an error during retrieval, it returns an appropriate error response and logs it.
func (h *Handler) getAllWebhooks(w http.ResponseWriter, r *http.Request) {
	docs, err := h.Client.Collection(utility.WebhooksCollection).Documents(r.Context()).GetAll()
	if err != nil {
		log.Printf("Error retrieving webhooks: %v", err)
		http.Error(w, "Error retrieving webhooks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var webhooks []models.RegisterWebhook
	for _, doc := range docs {
		var webhook models.RegisterWebhook
		if err := doc.DataTo(&webhook); err != nil {
			log.Printf("Error converting document to webhook: %v", err)
			http.Error(w, "Error processing webhook data: "+err.Error(), http.StatusInternalServerError)
			continue
		}
		webhook.ID = doc.Ref.ID
		webhooks = append(webhooks, webhook)
	}

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(webhooks); err != nil {
		log.Println("Error encoding webhooks response: ", err)
		http.Error(w, "Something went wrong during webhooks information processing: "+err.Error(), http.StatusInternalServerError)
	}
}

// getWebhookByID retrieves a specific webhook by its ID from Firestore and returns it as a JSON object in the response.
// If the webhook is not found or if there is an error during retrieval, it returns an appropriate error response and logs it.
func (h *Handler) getWebhookByID(w http.ResponseWriter, r *http.Request, id string) {
	doc, err := h.Client.Collection(utility.WebhooksCollection).Doc(id).Get(r.Context())
	if err != nil {
		log.Printf("Error retrieving webhook with ID %s: %v", id, err)
		http.Error(w, "Webhook not found: "+err.Error(), http.StatusNotFound)
		return
	}

	var webhook models.RegisterWebhook
	if err := doc.DataTo(&webhook); err != nil {
		log.Printf("Error converting document to webhook: %v", err)
		http.Error(w, "Error processing webhook data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	webhook.ID = doc.Ref.ID

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(webhook); err != nil {
		log.Println("Error encoding webhook: ", err)
		http.Error(w, "Something went wrong during webhook information processing: "+err.Error(), http.StatusInternalServerError)
	}
}

// deleteWebhook removes a specific webhook by its ID from Firestore.
// It first checks if the webhook exists, and if it does, it deletes it and returns a 204 No Content status.
// If the webhook is not found or if there is an error during deletion, it returns an appropriate error response and logs it.
func (h *Handler) deleteWebhook(w http.ResponseWriter, r *http.Request, id string) {
	_, err := h.Client.Collection(utility.WebhooksCollection).Doc(id).Get(r.Context())
	if err != nil {
		log.Println("Webhook with ID " + id + " not found")
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	_, err = h.Client.Collection(utility.WebhooksCollection).Doc(id).Delete(r.Context())
	if err != nil {
		log.Printf("Error deleting webhook: %v", err)
		http.Error(w, "Failed to delete webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	log.Println("Webhook with ID ", id, " deleted successfully")
}
