package utility

// TODO add comments
type DashboardResponse struct {
	Country       string            `json:"country"`
	ISOCode       string            `json:"isoCode"`
	Features      DashboardFeatures `json:"features"`
	LastRetrieval string            `json:"lastRetrieval"`
}

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

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type AirQuality struct {
	PM25  float64 `json:"pm25"`
	PM10  float64 `json:"pm10"`
	Level string  `json:"level"`
}

type OpenMeteoInfo struct {
	Hourly struct {
		Temperature2M []float64 `json:"temperature_2m"`
		Precipitation []float64 `json:"precipitation"`
	} `json:"hourly"`
}

type OSMResponse struct {
	CapLat float64 `json:"lat,string"`
	CapLng float64 `json:"lon,string"`
}

type RestCountryInfo struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`
	ISOCode    string    `json:"cca2"`
	Capital    []string  `json:"capital"`
	Latlng     []float64 `json:"latlng"`
	Population int       `json:"population"`
	Area       float64   `json:"area"`
}

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

type OpenAQLatestResponse struct {
	Results []struct {
		Value     float64 `json:"value"`
		SensorsID int     `json:"sensorsId"`
	} `json:"results"`
}
