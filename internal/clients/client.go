package clients

import (
	"context"
	"encoding/json"
	"time"

	"assignment-2/internal/cache"
	"assignment-2/internal/external"
	"assignment-2/internal/utility"

	"cloud.google.com/go/firestore"
)

// APIClient lists the methods used by DashboardHandler.
type APIClient interface {
	GetCountry(ctx context.Context, iso string) (utility.RestCountryResponse, error)
	GetWeather(ctx context.Context, lat, lng float64) (utility.OpenMeteoResponse, error)
	GetExchangeRates(ctx context.Context, base string, targets []string) (map[string]float64, error)
	GetAirQuality(ctx context.Context, iso, capital string) (pm10, pm25 float64, err error)
}

// RawClient calls external APIs directly. It can delegate to the existing fetch* functions.
type RawClient struct {
	// optionally add an http.Client or keys here
}

func (r *RawClient) GetCountry(ctx context.Context, iso string) (utility.RestCountryResponse, error) {
	// call your existing fetchCountryInfo (no ctx available there)
	return external.FetchCountryInfo(iso)
}

func (r *RawClient) GetWeather(ctx context.Context, lat, lng float64) (utility.OpenMeteoResponse, error) {
	return external.FetchWeatherInfo(lat, lng)
}

func (r *RawClient) GetExchangeRates(ctx context.Context, base string, targets []string) (map[string]float64, error) {
	return external.FetchExchangeRate(targets, base)
}

func (r *RawClient) GetAirQuality(ctx context.Context, iso, capital string) (float64, float64, error) {
	return external.FetchAirQualityInfo(iso, capital)
}

// CachedClient decorates another APIClient and uses Firestore to cache JSON payloads.
type CachedClient struct {
	Underlying APIClient
	FSClient   *firestore.Client
	TTLs       map[string]time.Duration
}

func NewCachedClient(underlying APIClient, fs *firestore.Client) *CachedClient {
	return &CachedClient{
		Underlying: underlying,
		FSClient:   fs,
		TTLs: map[string]time.Duration{
			"country":    24 * time.Hour,
			"weather":    10 * time.Minute,
			"exchange":   1 * time.Hour,
			"airquality": 15 * time.Minute,
		},
	}
}

func (c *CachedClient) GetCountry(ctx context.Context, iso string) (utility.RestCountryResponse, error) {
	key, _ := cache.MakeCacheKey("country", iso)
	if payload, hit, _ := cache.GetCached(ctx, c.FSClient, key); hit {
		var out utility.RestCountryResponse
		if err := json.Unmarshal(payload, &out); err == nil {
			return out, nil
		}
		// else fall through to refetch
	}
	res, err := c.Underlying.GetCountry(ctx, iso)
	if err != nil {
		return utility.RestCountryResponse{}, err
	}
	b, _ := json.Marshal(res)
	_ = cache.SetCached(ctx, c.FSClient, key, b, c.TTLs["country"])
	return res, nil
}

func (c *CachedClient) GetWeather(ctx context.Context, lat, lng float64) (utility.OpenMeteoResponse, error) {
	params := map[string]float64{"lat": lat, "lng": lng}
	key, _ := cache.MakeCacheKey("weather", params)
	if payload, hit, _ := cache.GetCached(ctx, c.FSClient, key); hit {
		var out utility.OpenMeteoResponse
		if err := json.Unmarshal(payload, &out); err == nil {
			return out, nil
		}
	}
	res, err := c.Underlying.GetWeather(ctx, lat, lng)
	if err != nil {
		return utility.OpenMeteoResponse{}, err
	}
	b, _ := json.Marshal(res)
	_ = cache.SetCached(ctx, c.FSClient, key, b, c.TTLs["weather"])
	return res, nil
}

func (c *CachedClient) GetExchangeRates(ctx context.Context, base string, targets []string) (map[string]float64, error) {
	key, _ := cache.MakeCacheKey("exchange", map[string]any{"base": base, "targets": targets})
	if payload, hit, _ := cache.GetCached(ctx, c.FSClient, key); hit {
		var out map[string]float64
		if err := json.Unmarshal(payload, &out); err == nil {
			return out, nil
		}
	}
	res, err := c.Underlying.GetExchangeRates(ctx, base, targets)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(res)
	_ = cache.SetCached(ctx, c.FSClient, key, b, c.TTLs["exchange"])
	return res, nil
}

func (c *CachedClient) GetAirQuality(ctx context.Context, iso, capital string) (float64, float64, error) {
	key, _ := cache.MakeCacheKey("airquality", map[string]string{"iso": iso, "capital": capital})
	if payload, hit, _ := cache.GetCached(ctx, c.FSClient, key); hit {
		var out struct{ PM10, PM25 float64 }
		if err := json.Unmarshal(payload, &out); err == nil {
			return out.PM10, out.PM25, nil
		}
	}
	pm10, pm25, err := c.Underlying.GetAirQuality(ctx, iso, capital)
	if err != nil {
		return -1, -1, err
	}
	b, _ := json.Marshal(struct{ PM10, PM25 float64 }{pm10, pm25})
	_ = cache.SetCached(ctx, c.FSClient, key, b, c.TTLs["airquality"])
	return pm10, pm25, nil
}
