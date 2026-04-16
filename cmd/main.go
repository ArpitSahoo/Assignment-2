package main

import (
	"assignment-2/internal"
	"assignment-2/internal/clients"
	"assignment-2/internal/handlers"
	"assignment-2/internal/middleware"
	"assignment-2/internal/utility"
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("PORT not set. Default: 8080")
		port = "8080"
	}

	client, err := internal.GetFirebaseClient()
	if err != nil {
		log.Fatalf("failed to initialize Firestore client: %v", err)
	}

	defer func(client *firestore.Client) {
		if err := client.Close(); err != nil {
			log.Printf("error closing Firestore client: %v", err)
		}
	}(client)

	raw := &clients.RawClient{}
	cached := clients.NewCachedClient(raw, client)

	handler := &handlers.Handler{
		Client: client,
		API:    cached,
	}

	docs, err := client.Collection(utility.WebhooksCollection).Documents(context.Background()).GetAll()
	if err != nil {
		log.Printf("Could not fetch webhook count: %v", err)
	} else {
		handler.WebhookCount.Store(int64(len(docs)))
		log.Printf("Webhooks loaded from Firestore: %d", len(docs))
	}

	router := http.NewServeMux()

	router.HandleFunc(utility.RegistrationPath, handler.HandleRegReq)
	router.HandleFunc(utility.RegistrationPathID, handler.HandleRegReq)
	router.HandleFunc(utility.DashboardPath, handler.DashboardHandler)
	router.HandleFunc(utility.NotificationPathBase, handler.WebhookHandler)
	router.HandleFunc(utility.NotificationPath, handler.WebhookIDHandler)
	router.HandleFunc(utility.StatusPath, handler.HandleStatus)
	router.HandleFunc(utility.AuthPath, handler.HandleAuthenticationReq)
	router.HandleFunc(utility.AuthPathKey, handler.HandleAuthenticationReq)

	// Wrap the router with the API key middleware so every request goes through validation.
	protectedHandler := middleware.APIKeyMiddleware(handler)(router)

	log.Printf("Server running on http://localhost:%s", port)
	log.Printf("Registration endpoint: http://localhost:%s%s", port, utility.RegistrationPath)
	log.Printf("Dashboard endpoint:    http://localhost:%s%s", port, utility.DashboardPath)
	log.Printf("Auth create endpoint:  http://localhost:%s%s", port, utility.AuthPath)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: protectedHandler,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
