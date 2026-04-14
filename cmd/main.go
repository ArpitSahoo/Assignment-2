package main

import (
	"assignment-2/internal"
	"assignment-2/internal/handlers"
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

	// Handler instance to inject the Firestore client into.
	handler := &handlers.Handler{
		Client: client,
	}

	router.HandleFunc(utility.RegistrationPath, handler.HandleRegReq)
	router.HandleFunc(utility.RegistrationPathID, handler.HandleRegReq)
	log.Printf("Firestore REST service listening on port %s with URL path /%s/ ...\n", port, utility.RegistrationPath)
	if errSrv := http.ListenAndServe(addr, router); errSrv != nil {
		panic(errSrv)
	}
}
