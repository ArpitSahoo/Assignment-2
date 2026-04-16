package clients

import (
	"assignment-2/internal/models"
	"context"
)

// APIClient lists the methods used by DashboardHandler.
// The interface is intentionally small and focused: it exposes the external
// data surfaces required by handlers so implementations can be swapped in tests
// or at runtime.
type APIClient interface {
	GetCountry(ctx context.Context, iso string) (models.RestCountryResponse, error)
	GetWeather(ctx context.Context, lat, lng float64) (models.OpenMeteoResponse, error)
	GetExchangeRates(ctx context.Context, base string, targets []string) (map[string]float64, error)
	GetAirQuality(ctx context.Context, iso, capital string) (pm10, pm25 float64, err error)
}
