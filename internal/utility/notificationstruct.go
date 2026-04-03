package utility

// RegisterWebhook represents a JSON payload used to create a webhook through the /notifications endpoint.
type RegisterWebhook struct {
	ID        string     `json:"id"`
	Url       string     `json:"url"`
	Country   string     `json:"country"`
	Event     string     `json:"event"`
	Threshold *Threshold `json:"threshold,omitempty"`
}

// WebhookResponse is a JSON response returned by the /notifications endpoint
type WebhookResponse struct {
	ID string `json:"id"`
}

// Threshold represents the threshold condition for triggering a webhook notification.
type Threshold struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
}

type WebhookInvocationPayload struct {
	ID      string `json:"id"`
	Country string `json:"country"`
	Event   string `json:"event"`
	Time    string `json:"time"`
}
