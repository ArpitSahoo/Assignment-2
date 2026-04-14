package handlers

import (
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/api/iterator"
)

var startTime = time.Now()

// HandleStatus is a function that handles the switch case for the status endpoint,
// it only allows GET requests and calls the handleGetStatus
// function to handle the logic of retrieving the status of the external APIs and returning the response.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetStatus(w)
	default:
		http.Error(w, "method not allowed, please use GET", http.StatusMethodNotAllowed)
	}
}

// handleGetStatus is a handler function which checks the status of the external APIs
// it returns a JSON response with the status codes, version and uptime of the server.
func (h *Handler) handleGetStatus(w http.ResponseWriter) {
	results := getAPIStatuses()
	uptimeSeconds := int(time.Since(startTime).Seconds())
	webhookCount := h.getWebhookCount()

	resp := utility.StatusResponse{
		RestCountriesAPI: results["restCountries"],
		MeteoAPI:         results["meteo"],
		OpenAQ:           checkOpenAQStatus(),
		NominatimAPI:     results["nominatim"],
		CurrencyAPI:      results["currency"],
		NotificationDB:   h.probeFirestore(),
		Webhooks:         webhookCount,
		Version:          "v1",
		Uptime:           uptimeSeconds,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) getWebhookCount() int {
	webhookCount := 0
	docs, err := h.Client.Collection(utility.WebhooksCollection).Documents(context.Background()).GetAll()
	if err != nil {
		log.Printf("Could not fetch webhook count: %v", err)
	} else {
		webhookCount = len(docs)
	}
	return webhookCount
}

// getAPIStatuses checks the status of each external API concurrently
// and returns a map with the API names and their corresponding status codes.
func getAPIStatuses() map[string]int {
	type probe struct {
		name   string
		url    string
		apiKey string
	}

	probes := []probe{
		{name: "restCountries", url: utility.RestCountriesProbe},
		{name: "meteo", url: utility.MeteoProbe},
		{name: "openAQ", url: utility.OpenAQProbe, apiKey: os.Getenv("OPENAQ_API_KEY")},
		{name: "nominatim", url: utility.NominatimProbe},
		{name: "currency", url: utility.CurrencyProbe},
	}

	resultCh := make(chan utility.ApiResult, len(probes))

	for _, p := range probes {
		go func(p probe) {
			resultCh <- utility.ApiResult{Name: p.name, Status: checkAPIStatus(p.url, p.apiKey)}
		}(p)
	}

	results := make(map[string]int, len(probes))
	for range probes {
		r := <-resultCh
		results[r.Name] = r.Status
	}
	return results
}

// checkAPIStatus probes a URL with an optional API key and returns the HTTP status code.
// Returns 500 if the request fails.
func checkOpenAQStatus() int {
	url := utility.OpenAQProbe
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return http.StatusInternalServerError
	}

	apiKey := os.Getenv("OPENAQ_API_KEY")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to reach %s: %v", url, err)
		return http.StatusInternalServerError
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Failed to close response body : %v", err)
		}
	}(resp.Body)

	return resp.StatusCode
}

// checkAPIStatus checks the status of an API by making a GET request.
// It returns the status code of the response if successful.
// If the request fails, it returns 500 Service Unavailable.
func checkAPIStatus(url string, apiKey string) int {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", url, err)
		return http.StatusInternalServerError
	}

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to reach %s: %v", url, err)
		return http.StatusInternalServerError
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Printf("Failed to close response body : %v", err)
		}
	}()

	return resp.StatusCode
}

// probeFirestore checks Firestore connectivity by fetching a single document.
// Returns 200 if reachable, 500 otherwise.
func (h *Handler) probeFirestore() int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := h.Client.Collection(utility.RegistrationsCollection).Limit(1).Documents(ctx).Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}
