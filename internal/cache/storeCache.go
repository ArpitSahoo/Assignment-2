package cache

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"cloud.google.com/go/firestore"
)

// MakeCacheKey returns a cache key for the given endpoint and parameters.
// Parameters are JSON-marshaled and returns an error if marshal fails.
func MakeCacheKey(endpoint string, params any) (string, error) {
	b, err := json.Marshal(params) //
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(append([]byte(endpoint+":"), b...))
	return endpoint + "_" + hex.EncodeToString(h[:]), nil
}

// GetCached attempts to retrieve a cached payload by document ID.
// Returns (payload, true, nil) on hit.
// Returns (nil, false, nil) on miss (document missing or expired).
// Returns (nil, false, err) on retrieval or decode error.
// If client is nil, acts as a miss.
func GetCached(ctx context.Context, client *firestore.Client, docID string) ([]byte, bool, error) {
	if client == nil {
		// Firestore not configured; treat as cache miss
		return nil, false, nil
	}
	doc, err := client.Collection(utility.ApiCacheCollection).Doc(docID).Get(ctx)
	if err != nil {
		// caller treats any error as miss and will fall back to fetching fresh
		return nil, false, err
	}
	var e models.CacheEntry
	if err := doc.DataTo(&e); err != nil {
		// if document is malformed, treat as miss
		return nil, false, err
	}
	if time.Now().After(e.ExpiresAt) {
		// if expired, treat as miss
		return nil, false, nil
	}
	return e.Payload, true, nil
}

// SetCached stores payload with the given time to live (TTL).
// If client is nil, caching is skipped.
// Timestamps use local time via time.Now().
func SetCached(ctx context.Context, client *firestore.Client, docID string, payload []byte, ttl time.Duration) error {
	if client == nil {
		// if no firestore client configured, skip caching
		return nil
	}
	entry := models.CacheEntry{
		Payload:   payload,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	_, err := client.Collection(utility.ApiCacheCollection).Doc(docID).Set(ctx, entry)
	return err
}
