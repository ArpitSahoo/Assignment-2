package handlers

import (
	"assignment-2/internal/utility"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

//TODO update handler in registrationhandlr to include 	Webhooks *HandlerWebhooks

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

		if webhook.Country != "" && webhook.Country != country {
			continue
		}

		payload := utility.WebhookInvocationPayload{
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

// getAllStoredWebhooks retrieves all registered webhooks from the Firestore collection.
// It returns a slice of RegisterWebhook structs or an error if the retrieval fails.
func (h *Handler) getAllStoredWebhooks(ctx context.Context) ([]utility.RegisterWebhook, error) {
	docs, err := h.Client.Collection(utility.WebhooksCollection).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var webhooks []utility.RegisterWebhook
	for _, doc := range docs {
		var webhook utility.RegisterWebhook
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

/*
------------DRAFT------------
TODO - Fullfør triggerthreshold når dashboard er tiligjengelig
func (h *HandlerWebhooks) triggerThresholdWebhooks(ctx context.Context, country string, dashboard values) {
	webhooks, err := h.getListOfWebhooks(ctx)
	if err != nil {
		log.Printf("Error retrieving webhooks: %v", err)
		return
	}

	for _, webhook := range webhooks {
		if webhook.Event != "THRESHOLD" {
			continue
		}

		//Field som skal sjekkes er det temperatur tallet currencies osv.?
		field := webhook.Threshold.Field

		//Verdien som skal sjekkes
		value := webhook.Threshold.Value

		//Metode som bruker den fielden og henter verdien til den
		meassuredValue := getMeasuredValue(webhook.Threshold.Field, dashboardet hentet)

		//Sjekk opperatøren
		opperator := webhook.Threshold.Operator

		if !checkThreshold(opperator, meassuredValue, value) {
			continue
		}

		payload := utility.ThresholdWebhookInvocationPayload{
			ID:      webhook.ID,
			Country: country,
			Event:   "THRESHOLD",
			Time:    time.Now().Format(time.RFC3339),
			Details: &utility.ThresholdDetails{
				Field:    field,
				Operator: opperator,
				Value:    value,
				MeassuredValue: meassuredValue,
			},
		}

		if err := sendWebhook(webhook.Url, payload); err != nil {
			log.Printf("Failed to send webhook %s: %v", webhook.ID, err)
		}
	}
}

func checkThreshold(opperator string, meassuredValue float64, value float64) bool {
	switch opperator {
	case ">": return meassuredValue > value, true
	case "<": return meassuredValue < value, true
	case ">=": return meassuredValue >= value, true
	case "<=": return meassuredValue <= value, true
	case "=": return meassuredValue = value, true
		//Advanced task bare legg til flere cases?
		//TODO hør med gruppe om denne løsningen
		//TODO opptadetr validering i notificationhandler.go
			//validOperators := map[string]bool{">": true, "<": true, ">=": true, "<=": true, "=": true}
	default:
		return false
	}
	return false
}
*/
