package handlers

import (
	"assignment-2/internal/utility"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

var webhooks []utility.RegisterWebhook

// WebhookHandler handles requests to the /notifications endpoint,
// allowing clients to register new webhooks and retrieve all registered webhooks.
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		postRequest(w, r)
	case http.MethodGet:
		getAllWebhooks(w)
	default:
		http.Error(w, "Method "+r.Method+" not supported for "+utility.NotificationPath, http.StatusMethodNotAllowed)
	}
}

// WebhookIDHandler handles requests to the /notifications/{id} endpoint,
// allowing clients to retrieve or delete a specific webhook by its ID.
func WebhookIDHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	switch r.Method {
	case http.MethodGet:
		getWebhookByID(w, id)
	case http.MethodDelete:
		deleteWebhook(w, id)
	default:
		http.Error(w, "Method "+r.Method+" not supported for "+utility.NotificationPath, http.StatusMethodNotAllowed)
	}
}

func postRequest(w http.ResponseWriter, r *http.Request) {
	webhook := utility.RegisterWebhook{}
	err := json.NewDecoder(r.Body).Decode(&webhook)
	if err != nil {
		log.Println("Error during decoding of webhook registration information. Error: ", err)
		http.Error(w, "Something went wrong during information processing. Error: "+err.Error()+
			"\nPlease check your input JSON.", http.StatusBadRequest)
		return
	}

	if validatedFields(w, webhook) {
		return
	}

	webhook.ID = strconv.Itoa(len(webhooks) + 1)

	webhooks = append(webhooks, webhook)

	log.Println("Webhook " + webhook.Url + " has been registered with ID " + webhook.ID)

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(map[string]string{
		"id": webhook.ID,
	})
	if err != nil {
		log.Println("JSON encoding failed. Error:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func validatedFields(w http.ResponseWriter, webhook utility.RegisterWebhook) bool {
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

	if webhook.Event == "THRESHOLD" {
		if webhook.Threshold == nil {
			log.Println("Error threshold block is required for THRESHOLD event")
			http.Error(w, "Error threshold block is required for THRESHOLD event",
				http.StatusBadRequest)
			return true
		}

		validFields := map[string]bool{
			"pm25": true, "pm10": true,
			"temperature": true, "precipitation": true,
		}
		if !validFields[webhook.Threshold.Field] {
			log.Println("Error threshold field must be: pm25, pm10, temperature, or precipitation")
			http.Error(w, "error threshold field must be: pm25, pm10, temperature, or precipitation",
				http.StatusBadRequest)
			return true
		}

		validOperators := map[string]bool{">": true, "<": true}
		if !validOperators[webhook.Threshold.Operator] {
			log.Println("Error threshold operator must be > or <")
			http.Error(w, "error: threshold.operator must be > or <",
				http.StatusBadRequest)
			return true
		}
	}
	return false
}

func getAllWebhooks(w http.ResponseWriter) {
	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(webhooks)
	if err != nil {
		log.Println("Error during encoding of webhook registration information. Error: ", err)
		http.Error(w, "Something went wrong during webhook information processing: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func getWebhookByID(w http.ResponseWriter, id string) {
	for index, webhook := range webhooks {
		if webhook.ID == id {
			w.Header().Set(utility.ContentType, utility.ApplicationJSON)
			w.WriteHeader(http.StatusOK)

			err := json.NewEncoder(w).Encode(webhooks[index])
			if err != nil {
				log.Println("Error during encoding of webhook registration information. Error: ", err)
				http.Error(w, "Something went wrong during webhook information processing: "+err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	log.Println("Webhook with ID " + id + " is not found")
	http.Error(w, "Webhook id is not found", http.StatusNotFound)
}

func deleteWebhook(w http.ResponseWriter, id string) {
	for index, webhook := range webhooks {
		if webhook.ID == id {
			webhooks = append(webhooks[:index], webhooks[index+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	log.Println("Webhook with ID " + id + " is not found")
	http.Error(w, "Webhook not found", http.StatusNotFound)
}
