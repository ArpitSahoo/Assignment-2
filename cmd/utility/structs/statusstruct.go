package structs

// StatusResponse represents the status.
type StatusResponse struct {
	RestCountriesAPI int    `json:"countries_api"`
	MeteoAPI         int    `json:"meteo_api"`
	OpenAQ           int    `json:"openaq_api"`
	NominatimAPI     int    `json:"nominatim_api"`
	CurrencyAPI      int    `json:"currency_api"`
	Version          string `json:"version"`
	Uptime           int    `json:"uptime"`
}
