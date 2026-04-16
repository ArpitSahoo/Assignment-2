package main

import (
	"assignment-2/internal"
	"assignment-2/internal/clients"
	"assignment-2/internal/handlers"
	"assignment-2/internal/middleware"
	"assignment-2/internal/utility"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
)

// Main starting point of the service
func main() {

	port := os.Getenv("PORT")
	if port == "" {
		log.Println("$PORT not set. Default: 8080")
		port = "8080"
	}

	addr := ":" + port

	router := http.NewServeMux()

	// Create a single client
	client, errC := internal.GetFirebaseClient()
	if errC != nil {
		log.Fatal(errC)
	}

	// Ensure client is properly closed when application is shut down.
	defer func(client *firestore.Client) {
		err := client.Close()
		if err != nil {
			log.Printf("Error closing Firestore client: %v", err)
		}
	}(client)

	raw := &clients.RawClient{}
	cached := clients.NewCachedClient(raw, client)

	handler := &handlers.Handler{
		Client: client,
		API:    cached,
	}

	router.HandleFunc(utility.AuthPath, handler.HandleAuthenticationReq)
	router.HandleFunc(utility.AuthPathKey, handler.HandleAuthenticationReq)
	router.HandleFunc(utility.RegistrationPath, handler.HandleRegReq)
	router.HandleFunc(utility.RegistrationPathID, handler.HandleRegReq)
	router.HandleFunc(utility.StatusPath, handler.HandleStatus)
	router.HandleFunc(utility.DashboardPath, handler.DashboardHandler)

	mw := middleware.APIKeyMiddleware(handler) // use handler as validator
	protected := mw(router)
	log.Printf("Firestore REST service listening on port %s with URL path /%s/ ...\n", port, utility.RegistrationPath)
	if errSrv := http.ListenAndServe(addr, protected); errSrv != nil {
		panic(errSrv)
	}
}
