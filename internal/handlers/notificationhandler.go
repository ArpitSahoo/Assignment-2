package handlers

import (
	"assignment-2/internal/utility"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

var webhooks []utility.RegisterWebhook

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
	webhook.ID = strconv.Itoa(len(webhooks) + 1)

	webhooks = append(webhooks, webhook)

	log.Println("Webhook " + webhook.Url + " has been registered with ID " + webhook.ID)

	w.Header().Set("Content-Type", "application/json")
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

func getAllWebhooks(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
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
			w.Header().Set("Content-Type", "application/json")
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

	http.Error(w, "Webhook not found", http.StatusNotFound)
}
