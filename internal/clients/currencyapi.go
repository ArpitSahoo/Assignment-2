package clients

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

var FetchExchangeRateFunc = FetchExchangeRate

// FetchExchangeRate fetches exchange rates for the given target currencies,
// relative to the base currency. Returns a map of currency code to rate,
// or nil if no target currencies are specified.
func FetchExchangeRate(targetCur []string, cur string) (map[string]float64, error) {
	if len(targetCur) == 0 {
		return nil, nil
	}
	if cur == "" {
		return nil, fmt.Errorf("base currency is empty")
	}

	curApiURL := strings.NewReplacer(
		utility.CurrencyCodePlaceholder, strings.ToUpper(cur),
	).Replace(utility.CurrencyAPIURL)

	resp, err := http.Get(curApiURL)
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

	result := make(map[string]float64)
	for _, cur := range targetCur {
		if rate, ok := exchangeRate.Rates[strings.ToUpper(cur)]; ok {
			result[strings.ToUpper(cur)] = rate
		}
	}

	return result, nil
}
