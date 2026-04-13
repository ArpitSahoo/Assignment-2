package utility

const RegistrationsCollection = "registrations"

const WebhooksCollection = "webhooks"

const RegistrationPath = "/envdash/v1/registrations/"
const RegistrationPathID = "/envdash/v1/registrations/{id}"

const NotificationPath = "/envdash/v1/notifications/{id}"
const NotificationPathBase = "/envdash/v1/notifications/"

const DashboardPath = "/envdash/v1/dashboard/{id}"
const RestCountriesApiUrl = "http://129.241.150.113:8080/v3.1/alpha/"
const OpenMeteoApiUrlBase = "https://api.open-meteo.com/v1/forecast?latitude={lat}&longitude={lng}&hourly=temperature_2m,precipitation"
const OpenAQURLBase = "https://api.openaq.org/v3/locations?coordinates={lat},{lng}&radius=25000&parameters_id=1,2&iso={isoCode}&limit=100"
const OSMURLBase = "https://nominatim.openstreetmap.org/search?city={cap}&countrycodes={isoCode}&format=jsonv2&addressdetails=1&limit=1"

const OpenAQSensorUrl = "https://api.openaq.org/v3/sensors/{id}/measurements"
const OpenAQLatestURL = "https://api.openaq.org/v3/locations/{id}/latest"

const ContentType = "Content-Type"
const ApplicationJSON = "application/json"
