package clients

import (
	"assignment-2/internal/models"
	"context"
)

// RawClient is the concrete, default implementation of APIClient.
// It currently has no fields, it groups the methods
// and allows future extension.
type RawClient struct{}

// GetCountry delegates to FetchCountryInfoFunc.
func (r *RawClient) GetCountry(ctx context.Context, iso string) (models.RestCountryResponse, error) {
	return FetchCountryInfoFunc(ctx, iso)
}

// GetWeather delegates to FetchWeatherInfoFunc.
func (r *RawClient) GetWeather(ctx context.Context, lat, lng float64) (models.OpenMeteoResponse, error) {
	return FetchWeatherInfoFunc(ctx, lat, lng)
}

// GetExchangeRates delegates to FetchExchangeRateFunc.
func (r *RawClient) GetExchangeRates(ctx context.Context, base string, targets []string) (map[string]float64, error) {
	return FetchExchangeRateFunc(ctx, targets, base)
}

// GetAirQuality delegates to FetchAirQualityInfoFunc.
func (r *RawClient) GetAirQuality(ctx context.Context, iso, capital string) (float64, float64, error) {
	return FetchAirQualityInfoFunc(ctx, iso, capital)
}
