package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	cloudfirestore "cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewTestFirestoreClient creates a new Firestore client for testing, configured to connect to the Firestore emulator
// if the FIRESTORE_EMULATOR_HOST environment variable is set. If the variable is not set, the test will be skipped.
func NewTestFirestoreClient(t *testing.T) *cloudfirestore.Client {
	t.Helper()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "test-project"
	}

	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set; skipping Firestore integration test")
	}

	client, err := cloudfirestore.NewClient(context.Background(), projectID)
	require.NoError(t, err, "failed to create firestore client")

	t.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

// ClearFirestoreEmulator sends a DELETE request to the Firestore emulator's REST API to clear all documents in the default database.
func ClearFirestoreEmulator(t *testing.T) {
	t.Helper()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "test-project"
	}

	host := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if host == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set; skipping Firestore integration test")
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("http://%s/emulator/v1/projects/%s/databases/(default)/documents", host, projectID),
		nil,
	)
	require.NoError(t, err, "failed to create emulator cleanup request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to clear firestore emulator")
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Logf("Error closing response body: %v", err)
			return
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected cleanup status")
}
