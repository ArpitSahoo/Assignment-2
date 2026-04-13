package utility

const RegistrationsCollection = "registrations"

const WebhooksCollection = "webhooks"

const RegistrationPath = "/envdash/v1/registrations/"
const RegistrationPathID = "/envdash/v1/registrations/{id}"

const NotificationPath = "/envdash/v1/notifications/{id}"
const NotificationPathBase = "/envdash/v1/notifications/"

const ContentType = "Content-Type"
const ApplicationJSON = "application/json"

const (
	TargetCurrency       = "target currency"
	AddTargetCurrency    = "add target currency"
	RemoveTargetCurrency = "remove target currency"
)

const (
	IsoCodeLength      = 2
	CurrencyCodeLength = 3
)
