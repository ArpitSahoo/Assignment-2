package clients

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// FetchCountryInfo fetches country data from the REST Countries API using the given ISO code.
// Returns the first result, or an error if the request fails or no country is found.
func FetchCountryInfo(isoCode string) (models.RestCountryResponse, error) {
	resp, err := http.Get(utility.RestCountriesApiUrl + isoCode)
	if err != nil {
		return models.RestCountryResponse{}, fmt.Errorf("fetching country: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return models.RestCountryResponse{}, fmt.Errorf("countries api returned %d: %s", resp.StatusCode, string(body))
	}

	var results []models.RestCountryResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return models.RestCountryResponse{}, fmt.Errorf("decoding country: %w", err)
	}
	if len(results) == 0 {
		return models.RestCountryResponse{}, fmt.Errorf("no country found for code %s", isoCode)
	}
	return results[0], nil
}
