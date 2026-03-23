package handlers

import (
	"assignment-2/cmd/utility/consts"
	"assignment-2/cmd/utility/structs"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

var (
	startTime  = time.Now()
	httpClient = &http.Client{
		Timeout: 3 * time.Second,
	}
)

// apiResult is a struct that holds the name and status of an API.
type apiResult struct {
	Name   string
	Status int
}

// HandleStatus is a function that handles the switch case for the status endpoint,
// it only allows GET requests and calls the handleGetStatus
// function to handle the logic of retrieving the status of the external APIs and returning the response.
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetStatus(w, r)
	default:
		http.Error(w, "method not allowed, please use GET", http.StatusMethodNotAllowed)
	}
}

// handleGetStatus is a handler function which checks the status of the external APIs
// it returns a JSON response with the status codes, version and uptime of the server.
func handleGetStatus(w http.ResponseWriter, r *http.Request) {
	results := getAPIStatuses()
	uptimeSeconds := int(time.Since(startTime).Seconds())

	resp := structs.StatusResponse{
		RestCountriesAPI: results["restCountries"],
		MeteoAPI:         results["meteo"],
		OpenAQ:           results["openAQ"],
		NominatimAPI:     results["nominatim"],
		CurrencyAPI:      results["currency"],
		Version:          "v1",
		Uptime:           uptimeSeconds,
	}

	jsonWriter(w, resp)
}

// getAPIStatuses checks the status of each external API concurrently
// and returns a map with the API names and their corresponding status codes.
func getAPIStatuses() map[string]int {
	apis := map[string]string{
		"restCountries": consts.RestCountriesAPI,
		"meteo":         consts.MeteoAPI,
		"openAQ":        consts.OpenAQAPI,
		"nominatim":     consts.NominatimAPI,
		"currency":      consts.CurrencyAPI,
	}

	resultCh := make(chan apiResult)

	for name, url := range apis {
		go func(name, url string) {
			resultCh <- apiResult{
				Name:   name,
				Status: checkAPIStatus(url),
			}
		}(name, url)
	}

	results := make(map[string]int)
	for range apis {
		result := <-resultCh
		results[result.Name] = result.Status
	}
	return results
}

// checkAPIStatus checks the status of an API by making a GET request.
// It returns the status code of the response if successful.
// If the request fails, it returns 503 Service Unavailable.
func checkAPIStatus(url string) int {
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Printf("Failed to reach %s: %v", url, err)
		return http.StatusServiceUnavailable
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Printf("Failed to close response body : %v", err)
		}
	}()

	return resp.StatusCode
}

// jsonWriter is a helper function that takes an http.ResponseWriter and StatusResponse,
// sets the Content-Type header to application/json, and encodes the data as JSON to the response.
func jsonWriter(w http.ResponseWriter, resp structs.StatusResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
