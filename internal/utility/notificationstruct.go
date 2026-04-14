package utility

// RegisterWebhook represents a JSON payload used to create a webhook through the /notifications endpoint.
type RegisterWebhook struct {
	ID        string     `json:"id,omitempty"`
	Url       string     `json:"url"`
	Country   string     `json:"country"`
	Event     string     `json:"event"`
	Threshold *Threshold `json:"threshold,omitempty"`
}

// WebhookResponse is a JSON response returned by the /notifications endpoint.
type WebhookResponse struct {
	ID string `json:"id"`
}

// Threshold represents the threshold condition for triggering a webhook notification.
type Threshold struct {
	Field         string  `json:"field"`
	Operator      string  `json:"operator"`
	UpperOperator string  `json:"upperOperator,omitempty"`
	UpperValue    float64 `json:"upperValue,omitempty"`
	Value         float64 `json:"value"`
}

// WebhookInvocationPayload represents the JSON body sent when a lifecycle webhook is triggered.
type WebhookInvocationPayload struct {
	ID      string            `json:"id"`
	Country string            `json:"country"`
	Event   string            `json:"event"`
	Time    string            `json:"time"`
	Details *ThresholdDetails `json:"details,omitempty"`
}

// ThresholdDetails provides detailed information about the threshold
// condition that triggered a webhook notification.
type ThresholdDetails struct {
	Field         string  `json:"field"`
	Operator      string  `json:"operator"`
	Threshold     float64 `json:"threshold"`
	UpperOperator string  `json:"upperOperator,omitempty"`
	UpperValue    float64 `json:"upperValue,omitempty"`
	MeasuredValue float64 `json:"measuredValue"`
}
