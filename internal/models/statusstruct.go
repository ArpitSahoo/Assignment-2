package models

// StatusResponse represents the status.
type StatusResponse struct {
	RestCountriesAPI int    `json:"countries_api"`
	MeteoAPI         int    `json:"meteo_api"`
	OpenAQ           int    `json:"openaq_api"`
	NominatimAPI     int    `json:"nominatim_api"`
	CurrencyAPI      int    `json:"currency_api"`
	NotificationDB   int    `json:"notification_db"`
	Webhooks         int    `json:"webhooks"`
	Registrations    int    `json:"registrations"`
	Version          string `json:"version"`
	Uptime           int    `json:"uptime"`
}

// ApiResult struct for api results
type ApiResult struct {
	Name   string
	Status int
}
