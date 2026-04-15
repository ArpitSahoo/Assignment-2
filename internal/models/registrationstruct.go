package models

// RegistrationRequest represents a JSON payload used to create or replace
// a dashboard configuration through the /registrations endpoint.
type RegistrationRequest struct {
	Country  string               `json:"country" firestore:"country"`
	IsoCode  string               `json:"isoCode" firestore:"isoCode"`
	Features RegistrationFeatures `json:"features" firestore:"features"`
}

// RegistrationFeatures specifies which dashboard fields to be included
// in the dashboard configuration stored through the /registrations endpoint.
type RegistrationFeatures struct {
	Temperature      bool     `json:"temperature" firestore:"temperature"`
	Precipitation    bool     `json:"precipitation" firestore:"precipitation"`
	AirQuality       bool     `json:"airQuality" firestore:"airQuality"`
	Capital          bool     `json:"capital" firestore:"capital"`
	Coordinates      bool     `json:"coordinates" firestore:"coordinates"`
	Population       bool     `json:"population" firestore:"population"`
	Area             bool     `json:"area" firestore:"area"`
	TargetCurrencies []string `json:"targetCurrencies" firestore:"targetCurrencies"`
}

// RegistrationResponse is a JSON response returned by the /registrations
// endpoint after a registration is created.
type RegistrationResponse struct {
	ID         string `json:"id"`
	LastChange string `json:"lastChange"`
}

// StoredRegistration represents a registration document as persisted in Firestore.
type StoredRegistration struct {
	ID         string               `json:"id,omitempty" firestore:"-"`
	Country    string               `json:"country" firestore:"country"`
	IsoCode    string               `json:"isoCode" firestore:"isoCode"`
	Features   RegistrationFeatures `json:"features" firestore:"features"`
	LastChange string               `json:"lastChange" firestore:"lastChange"`
}

// RegistrationPatchRequest is a JSON payload used to partially update
// a stored dashboard configuration through the /registrations/{id} endpoint.
type RegistrationPatchRequest struct {
	Country  *string                    `json:"country,omitempty"`
	IsoCode  *string                    `json:"isoCode,omitempty"`
	Features *RegistrationPatchFeatures `json:"features,omitempty"`
}

// RegistrationPatchFeatures represents feature fields which may be
// partially updated in a PATCH request. Supports replacement of
// the entire targetCurrencies list, and addition/removal of currencies.
type RegistrationPatchFeatures struct {
	Temperature      *bool     `json:"temperature,omitempty"`
	Precipitation    *bool     `json:"precipitation,omitempty"`
	AirQuality       *bool     `json:"airQuality,omitempty"`
	Capital          *bool     `json:"capital,omitempty"`
	Coordinates      *bool     `json:"coordinates,omitempty"`
	Population       *bool     `json:"population,omitempty"`
	Area             *bool     `json:"area,omitempty"`
	TargetCurrencies *[]string `json:"targetCurrencies,omitempty"`

	AddTargetCurrencies    *[]string `json:"addTargetCurrencies,omitempty"`
	RemoveTargetCurrencies *[]string `json:"removeTargetCurrencies,omitempty"`
}
