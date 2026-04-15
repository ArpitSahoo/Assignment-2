package handlers

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// triggerLifecycleWebhooks sends webhook notifications for lifecycle events
// such as REGISTER, CHANGE, DELETE, and INVOKE. It loads stored webhook
// registrations, filters matching subscriptions by event and country, and
// sends a JSON POST request to each matching webhook URL.
func (h *Handler) triggerLifecycleWebhooks(ctx context.Context, event string, country string) {
	webhooks, err := h.getAllStoredWebhooks(ctx)
	if err != nil {
		log.Printf("Error retrieving webhooks: %v", err)
		return
	}

	for _, webhook := range webhooks {
		if webhook.Event != event {
			continue
		}

		if matchesCountry(webhook, country) {
			continue
		}

		payload := models.WebhookInvocationPayload{
			ID:      webhook.ID,
			Country: country,
			Event:   event,
			Time:    time.Now().Format("20060102 15:04"),
		}

		if err := sendWebhook(webhook.Url, payload); err != nil {
			log.Printf("Failed to send webhook %s: %v", webhook.ID, err)
		}
	}
}

// triggerThresholdWebhooks checks all THRESHOLD webhooks against the populated
// dashboard and fires a notification for each threshold condition that is met.
func (h *Handler) triggerThresholdWebhooks(ctx context.Context, country string, dashboard models.DashboardResponse) {
	webhooks, err := h.getAllStoredWebhooks(ctx)

	if err != nil {
		log.Printf("Error retrieving webhooks: %v", err)
		return
	}

	for _, webhook := range webhooks {
		h.evaluateThresholdWebhook(webhook, country, dashboard)
	}
}

// evaluateThresholdWebhook checks a single webhook against the dashboard and
// sends a notification if the threshold condition is met.
func (h *Handler) evaluateThresholdWebhook(webhook models.RegisterWebhook, country string, dashboard models.DashboardResponse) {
	if webhook.Event != "THRESHOLD" {
		return
	}

	if matchesCountry(webhook, country) {
		return
	}

	measuredValue, ok := getMeasuredValue(webhook.Threshold.Field, dashboard)
	if !ok {
		return
	}

	if !checkThreshold(webhook.Threshold.Operator, measuredValue, webhook.Threshold.Value) {
		return
	}

	if webhook.Threshold.UpperOperator != "" {
		if !checkThreshold(webhook.Threshold.UpperOperator, measuredValue, webhook.Threshold.UpperValue) {
			return
		}
	}

	payload := models.WebhookInvocationPayload{
		ID:      webhook.ID,
		Country: country,
		Event:   "THRESHOLD",
		Time:    time.Now().Format("20060102 15:04"),
		Details: &models.ThresholdDetails{
			Field:         webhook.Threshold.Field,
			Operator:      webhook.Threshold.Operator,
			Threshold:     webhook.Threshold.Value,
			MeasuredValue: measuredValue,
			UpperOperator: webhook.Threshold.UpperOperator,
			UpperValue:    webhook.Threshold.UpperValue,
		},
	}

	if err := sendWebhook(webhook.Url, payload); err != nil {
		log.Printf("Failed to send webhook %s: %v", webhook.ID, err)
	}
}

// matchesCountry reports whether a webhook should be skipped for the given country.
// Returns true if the webhook has a country filter that does not match the given country.
func matchesCountry(webhook models.RegisterWebhook, country string) bool {
	return webhook.Country != "" && webhook.Country != country
}

// getMeasuredValue extracts the numeric value for a given field from the dashboard response.
// Returns the value and true if the field is enabled, false otherwise.
func getMeasuredValue(field string, dashboard models.DashboardResponse) (float64, bool) {
	switch field {
	case "temperature":
		if dashboard.Features.Temperature != nil {
			return *dashboard.Features.Temperature, true
		}
	case "precipitation":
		if dashboard.Features.Precipitation != nil {
			return *dashboard.Features.Precipitation, true
		}
	case "pm25":
		if dashboard.Features.AirQuality != nil {
			return dashboard.Features.AirQuality.PM25, true
		}
	case "pm10":
		if dashboard.Features.AirQuality != nil {
			return dashboard.Features.AirQuality.PM10, true
		}
	}
	return 0, false
}

// checkThreshold evaluates whether a measured value satisfies the threshold
// condition using the given operator.
func checkThreshold(operator string, measuredValue float64, value float64) bool {
	switch operator {
	case ">":
		return measuredValue > value
	case "<":
		return measuredValue < value
	case ">=":
		return measuredValue >= value
	case "<=":
		return measuredValue <= value
	case "=":
		return measuredValue == value
	default:
		return false
	}
}

// getAllStoredWebhooks retrieves all registered webhooks from the Firestore collection.
// It returns a slice of RegisterWebhook structs or an error if the retrieval fails.
func (h *Handler) getAllStoredWebhooks(ctx context.Context) ([]models.RegisterWebhook, error) {
	docs, err := h.Client.Collection(utility.WebhooksCollection).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var webhooks []models.RegisterWebhook
	for _, doc := range docs {
		var webhook models.RegisterWebhook
		if err := doc.DataTo(&webhook); err != nil {
			log.Printf("Error converting document to webhook: %v", err)
			continue
		}
		webhook.ID = doc.Ref.ID
		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// sendWebhook sends a POST request to the specified URL with the given payload as a JSON body.
func sendWebhook(url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, utility.ApplicationJSON, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	return nil
}
