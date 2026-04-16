package clients

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

var FetchExchangeRateFunc = FetchExchangeRate

// FetchExchangeRate gets exchange rates for fetches exchange rates for the given target currencies,
// relative to the base currency. Uses ctx when making the HTTP request. Returns a map of currency code to rate,
// or nil if no target currencies are specified
func FetchExchangeRate(ctx context.Context, targetCur []string, curr string) (map[string]float64, error) {
	url := strings.Replace(utility.CurrencyAPIURL, utility.CurrencyCodePlaceholder, curr, 1)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating exchange request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching exchange rates: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("exchange rate api returned %d: %s", resp.StatusCode, string(body))
	}

	var exchangeRate models.ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&exchangeRate); err != nil {
		return nil, fmt.Errorf("decoding exchange rates: %w", err)
	}

	// Filter to requested targets (if targets empty, return all)
	if len(targetCur) == 0 {
		return exchangeRate.Rates, nil
	}
	results := make(map[string]float64, len(targetCur))
	for _, curr := range targetCur {
		if v, ok := exchangeRate.Rates[t]; ok {
			results[curr] = v
		}
	}
	return results, nil
}
