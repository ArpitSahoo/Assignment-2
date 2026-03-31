package internal

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

// Firebase context and client used by Firestore functions throughout the program.
var ctx context.Context

// GetFirebaseContext returns Firebase context and initializes if not already done.
func GetFirebaseContext() context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

func GetFirebaseClient() (*firestore.Client, error) {
	// Firebase initialization
	ctx = GetFirebaseContext()

	// We use a service account, load credentials file that you downloaded from your project's settings menu.
	// It should reside in your project directory.
	// Make sure this file is git-ignored, since it is the access token to the database.
	sa := option.WithCredentialsFile("./assigment-2-firebase-secret.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// Instantiate client
	client, err := app.Firestore(ctx)

	// Alternative setup, directly through Firestore (without initial reference to Firebase); but requires Project ID; useful if multiple projects are used
	// client, err := firestore.NewClient(ctx, projectID)

	// Check whether there is an error when connecting to Firestore
	if err != nil {
		log.Println(err)
		return client, err
	}

	return client, nil
}
