package utility

// RegistrationRequest represents a JSON payload used to create or replace
// a dashboard configuration through the /registrations endpoint.
type RegistrationRequest struct {
	Country  string               `json:"country"`
	IsoCode  string               `json:"isoCode"`
	Features RegistrationFeatures `json:"features"`
}

// RegistrationFeatures specifies which dashboard fields to be included
// in the dashboard configuration stored through the /registration endpoint.
type RegistrationFeatures struct {
	Temperature      bool     `json:"temperature"`
	Precipitation    bool     `json:"precipitation"`
	AirQuality       bool     `json:"airQuality"`
	Capital          bool     `json:"capital"`
	Coordinates      bool     `json:"coordinates"`
	Population       bool     `json:"population"`
	Area             bool     `json:"area"`
	TargetCurrencies []string `json:"targetCurrencies"`
}

// RegistrationResponse is a JSON response returned by the /registration endpoint
// after a registration has been created or updated.
type RegistrationResponse struct {
	ID         string `json:"id"`
	LastChange string `json:"lastChange"`
}
