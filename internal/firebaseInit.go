package internal

import (
	"context"
	"fmt"
	"log"
	"sync"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

var (
	firestoreClient *firestore.Client
	once            sync.Once
)

// GetFirebaseClient returns a shared Firestore client, initializing it once.
// Requires GOOGLE_APPLICATION_CREDENTIALS env var or GCP environment.
func GetFirebaseClient() (*firestore.Client, error) {
	var initErr error

	once.Do(func() {
		ctx := context.Background()

		app, err := firebase.NewApp(ctx, nil)
		if err != nil {
			initErr = fmt.Errorf("firebase.NewApp: %w", err)
			log.Println(initErr)
			return
		}

		client, err := app.Firestore(ctx)
		if err != nil {
			initErr = fmt.Errorf("app.Firestore: %w", err)
			log.Println(initErr)
			return
		}

		firestoreClient = client
	})

	if initErr != nil {
		return nil, initErr
	}

	return firestoreClient, nil
}
