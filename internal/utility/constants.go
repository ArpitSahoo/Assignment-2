package utility

// API route paths
const (
	RegistrationPath     = "/envdash/v1/registrations/"
	RegistrationPathID   = "/envdash/v1/registrations/{id}"
	NotificationPath     = "/envdash/v1/notifications/{id}"
	NotificationPathBase = "/envdash/v1/notifications/"
	DashboardPath        = "/envdash/v1/dashboards/{id}"
	StatusPath           = "/envdash/v1/status/"
	AuthPath             = "/envdash/v1/auth/"
	AuthPathKey          = "/envdash/v1/auth/{key}"
)

// Firestore collections
const (
	RegistrationsCollection = "registrations"
	WebhooksCollection      = "webhooks"
	APIKeysCollection       = "apiKeys"
)

// HTTP constants
const (
	ContentType     = "Content-Type"
	ApplicationJSON = "application/json"
	HeaderUserAgent = "User-Agent"
	UserAgentValue  = "assignment-2/1.0"
)

// Authentication
const (
	APIKeyHeader      = "X-API-Key"
	BaseAPIKeyStarter = "sk-envdash-"
	OpenAQAPIKey      = "OPENAQ_API_KEY"
)

// External API base URLs
const (
	RestCountriesAPIURL     = "http://129.241.150.113:8080/v3.1/alpha/"
	RestCountriesAPIURLName = "http://129.241.150.113:8080/v3.1/name/{name}"
	OpenMeteoAPIURL         = "https://api.open-meteo.com/v1/forecast?latitude={lat}&longitude={lng}&hourly=temperature_2m,precipitation"
	OpenAQURL               = "https://api.openaq.org/v3/locations?coordinates={lat},{lng}&radius=25000&limit=100"
	OpenAQLatestURL         = "https://api.openaq.org/v3/locations/{id}/latest"
	OSMURL                  = "https://nominatim.openstreetmap.org/search?city={cap}&countrycodes={isoCode}&format=jsonv2&addressdetails=1&limit=1"
	CurrencyAPIURL          = "https://api.exchangerate-api.com/v4/latest/{cur}"
)

// URL template placeholders
const (
	CurrencyCodePlaceholder = "{cur}"
	CapitalPlaceholder      = "{cap}"
	ISOCodePlaceholder      = "{isoCode}"
	LatPlaceholder          = "{lat}"
	LngPlaceholder          = "{lng}"
	OpenAQIDPlaceholder     = "{id}"
)

// Health check probe URLs
const (
	RestCountriesProbe = "http://129.241.150.113:8080/v3.1/alpha/NO"
	CurrencyProbe      = "http://129.241.150.113:9090/currency/NOK/"
	MeteoProbe         = "https://api.open-meteo.com/v1/forecast?latitude=0&longitude=0&current=temperature_2m"
	OpenAQProbe        = "https://api.openaq.org/v3/parameters/2"
	NominatimProbe     = "https://nominatim.openstreetmap.org/status.php?format=json"
)

// Client/formatting constants
const (
	MaxOpenAQLocations = 5
	Pm10ParameterID    = 1
	Pm25ParameterID    = 2
	FloatPrecision     = 6
	FloatBitSize       = 64
)

// Validation lengths
const (
	IsoCodeLength           = 2
	CurrencyCodeLength      = 3
	MinCountryCoordinates   = 2
	RegistrationDocIDLength = 20
)

// Currency patch field names used in validation error messages
const (
	TargetCurrency       = "target currency"
	AddTargetCurrency    = "add target currency"
	RemoveTargetCurrency = "remove target currency"
)

// Air quality thresholds (PM25 based on AQI breakpoints)
const (
	UnknownAirQualityValue = -1
	Pm25GoodMax            = 12.0
	Pm25ModerateMax        = 35.4
	Pm25SensitiveGroupsMax = 55.4
	Pm25UnhealthyMax       = 150.4
	Pm25VeryUnhealthyMax   = 250.4
)

// Air quality level labels
const (
	AirQualityUnknown         = "unknown"
	AirQualityGood            = "Good"
	AirQualityModerate        = "Moderate"
	AirQualitySensitiveGroups = "Unhealthy for Sensitive Groups"
	AirQualityUnhealthy       = "Unhealthy"
	AirQualityVeryUnhealthy   = "Very Unhealthy"
	AirQualityHazardous       = "Hazardous"
)

const ApiCacheCollection = "cache"
