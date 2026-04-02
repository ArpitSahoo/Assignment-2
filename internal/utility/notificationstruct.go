package utility

type RegisterWebhook struct {
	ID      string `json:"id,omitempty"`
	Url     string `json:"url"`
	Country string `json:"country"`
	Event   string `json:"event"`
}

type Threshold struct {
	Url       string `json:"url"`
	Country   string `json:"country"`
	Event     string `json:"event"`
	Threshold struct {
		Field    string  `json:"field"`
		Operator string  `json:"operator"`
		Value    float64 `json:"value"`
	} `json:"threshold"`
}
