package clients

import (
	"assignment-2/internal/cache"
	"assignment-2/internal/models"
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/firestore"
)

// CachedClient decorates an underlying APIClient and provides a Firestore-backed
// cache for API responses. Callers use CachedClient exactly like any other
// APIClient: it implements the same methods and performs caching transparently.
type CachedClient struct {
	Underlying APIClient
	FSClient   *firestore.Client
	TTLs       map[string]time.Duration
}

// NewCachedClient creates a new CachedClient that wraps the provided underlying
// APIClient and uses the provided Firestore client for caching. The function also
// initializes sensible default TTLs for the supported endpoints.
func NewCachedClient(underlying APIClient, fsClient *firestore.Client) *CachedClient {
	return &CachedClient{
		Underlying: underlying,
		FSClient:   fsClient,
		TTLs: map[string]time.Duration{
			"country":    24 * time.Hour,   //RestCounties have 24 hours TTL, as country information doesn't change often
			"weather":    10 * time.Minute, // Weather have 10 min, since the weather changes often
			"exchange":   15 * time.Hour,   // Exchange rates have 15 hours TTL, as they are updated every 24 hours, but we want to be sure to refresh before that
			"airquality": 15 * time.Minute, // Air quality can change rapidly,
			// so we set a short TTL of 15 minutes to ensure relatively fresh data while avoiding excessive API calls
		},
	}
}

// GetCountry returns country information for the given ISO code. It first tries
// to return a cached value from Firestore. On cache miss or unmarshalling error,
// it delegates to the underlying APIClient, then caches the JSON-marshaled
// result with the configured TTL. It returns RestCountryResponse or an error if the underlying call fails.
func (c *CachedClient) GetCountry(ctx context.Context, iso string) (models.RestCountryResponse, error) {
	key, _ := cache.MakeCacheKey("country", iso)
	if payload, hit, _ := cache.GetCached(ctx, c.FSClient, key); hit {
		var out models.RestCountryResponse
		if err := json.Unmarshal(payload, &out); err == nil {
			return out, nil
		}
		// If unmarshalling fails, fall through to refetch from underlying client.
	}
	res, err := c.Underlying.GetCountry(ctx, iso)
	if err != nil {
		return models.RestCountryResponse{}, err
	}
	b, _ := json.Marshal(res)
	_ = cache.SetCached(ctx, c.FSClient, key, b, c.TTLs["country"])
	return res, nil
}

// GetWeather returns weather/time-series data for the given latitude and longitude.
// It attempts a Firestore-backed cache lookup keyed by the lat/lng params and falls
// back to the underlying APIClient on miss. Successful live responses are cached.
func (c *CachedClient) GetWeather(ctx context.Context, lat, lng float64) (models.OpenMeteoResponse, error) {
	params := map[string]float64{"lat": lat, "lng": lng}
	key, _ := cache.MakeCacheKey("weather", params)
	if payload, hit, _ := cache.GetCached(ctx, c.FSClient, key); hit {
		var out models.OpenMeteoResponse
		if err := json.Unmarshal(payload, &out); err == nil {
			return out, nil
		}
	}
	res, err := c.Underlying.GetWeather(ctx, lat, lng)
	if err != nil {
		return models.OpenMeteoResponse{}, err
	}
	b, _ := json.Marshal(res)
	_ = cache.SetCached(ctx, c.FSClient, key, b, c.TTLs["weather"])
	return res, nil
}

// GetExchangeRates returns exchange rates for the provided base currency and the
// requested target currencies. The cache key includes both base and target list.
// On cache miss the underlying client is used and the resulting map is written to
// Firestore (JSON-marshaled).
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

// GetAirQuality returns PM10 and PM2.5 measurements for the given country ISO
// and capital name. It uses the airquality cache key namespace. On cache miss,
// it delegates to the underlying APIClient and stores the returned values.
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
