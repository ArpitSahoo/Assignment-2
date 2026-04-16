package models

// DashboardResponse is the top-level response returned by the dashboard endpoint,
// containing the country info, enabled features, and the time of last retrieval.
type DashboardResponse struct {
	Country       string            `json:"country"`
	ISOCode       string            `json:"isoCode"`
	Features      DashboardFeatures `json:"features"`
	LastRetrieval string            `json:"lastRetrieval"`
}

// DashboardFeatures holds the optional feature fields returned in a dashboard response.
// Each field is a pointer so it is omitted from the JSON output if not enabled.
type DashboardFeatures struct {
	Temperature      *float64           `json:"temperature,omitempty"`
	Precipitation    *float64           `json:"precipitation,omitempty"`
	AirQuality       *AirQuality        `json:"airQuality,omitempty"`
	Capital          *string            `json:"capital,omitempty"`
	Coordinates      *Coordinates       `json:"coordinates,omitempty"`
	Population       *int64             `json:"population,omitempty"`
	Area             *float64           `json:"area,omitempty"`
	TargetCurrencies map[string]float64 `json:"targetCurrencies,omitempty"`
}

// Coordinates holds a geographic latitude and longitude pair.
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// AirQuality holds PM2.5 and PM10 concentration values and a human-readable quality level.
type AirQuality struct {
	PM25  float64 `json:"pm25"`
	PM10  float64 `json:"pm10"`
	Level string  `json:"level"`
}

// RestCountryResponse holds country data returned by the REST Countries API,
// including name, coordinates, capital, population, area, and currencies.
type RestCountryResponse struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`
	ISOCode    string    `json:"cca2"`
	Capital    []string  `json:"capital"`
	Latlng     []float64 `json:"latlng"`
	Population int       `json:"population"`
	Area       float64   `json:"area"`
	Currencies map[string]struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	} `json:"currencies"`
}

// OpenMeteoResponse holds hourly weather data returned by the Open-Meteo API.
type OpenMeteoResponse struct {
	Hourly struct {
		Temperature2M []float64 `json:"temperature_2m"`
		Precipitation []float64 `json:"precipitation"`
	} `json:"hourly"`
}

// ExchangeRateResponse holds the exchange rates returned by the currency API,
// keyed by uppercase currency code.
type ExchangeRateResponse struct {
	Rates map[string]float64 `json:"rates"`
}

// OSMResponse holds the coordinates of a location returned by the OpenStreetMap Nominatim API.
type OSMResponse struct {
	CapLat float64 `json:"lat,string"`
	CapLng float64 `json:"lon,string"`
}

// OpenAQResponse holds the list of air quality monitoring locations returned by the OpenAQ API,
// each with their associated sensors and parameter metadata.
type OpenAQResponse struct {
	Results []struct {
		ID      int `json:"id"`
		Sensors []struct {
			ID        int `json:"id"`
			Parameter struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"parameter"`
		} `json:"sensors"`
	} `json:"results"`
}

// OpenAQLatestResponse holds the latest sensor readings for a given OpenAQ location.
type OpenAQLatestResponse struct {
	Results []struct {
		Value     float64 `json:"value"`
		SensorsID int     `json:"sensorsId"`
	} `json:"results"`
}
