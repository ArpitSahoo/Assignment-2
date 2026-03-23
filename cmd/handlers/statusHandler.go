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
	restCountriesAPIStatus := checkAPIStatus(consts.RestCountriesAPI)
	meteoAPIStatus := checkAPIStatus(consts.MeteoAPI)
	openAQAPIStatus := checkAPIStatus(consts.OpenAQAPI)
	nominatimAPIStatus := checkAPIStatus(consts.NominatimAPI)
	currencyAPIStatus := checkAPIStatus(consts.CurrencyAPI)
	uptimeSeconds := int(time.Since(startTime).Seconds())

	resp := structs.StatusResponse{
		RestCountriesAPI: restCountriesAPIStatus,
		MeteoAPI:         meteoAPIStatus,
		OpenAQ:           openAQAPIStatus,
		NominatimAPI:     nominatimAPIStatus,
		CurrencyAPI:      currencyAPIStatus,
		Version:          "v1",
		Uptime:           uptimeSeconds,
	}

	jsonWriter(w, resp)
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
