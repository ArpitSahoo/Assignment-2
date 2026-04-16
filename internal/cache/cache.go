package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"cloud.google.com/go/firestore"
)

const apiCacheCollection = "api_cache"

type cacheEntry struct {
	Payload   []byte    `firestore:"payload"`
	CreatedAt time.Time `firestore:"createdAt"`
	ExpiresAt time.Time `firestore:"expiresAt"`
}

// MakeCacheKey deterministically maps endpoint+params to a doc ID.
// exported so other packages can use it.
func MakeCacheKey(endpoint string, params any) (string, error) {
	b, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(append([]byte(endpoint+":"), b...))
	return endpoint + "_" + hex.EncodeToString(h[:]), nil
}

// GetCached returns payload, hit, error. hit==true => valid (not expired) payload returned.
func GetCached(ctx context.Context, client *firestore.Client, docID string) ([]byte, bool, error) {
	doc, err := client.Collection(apiCacheCollection).Doc(docID).Get(ctx)
	if err != nil {
		// caller treats any error as miss and will fall back to fetching fresh
		return nil, false, err
	}
	var e cacheEntry
	if err := doc.DataTo(&e); err != nil {
		return nil, false, err
	}
	if time.Now().After(e.ExpiresAt) {
		return nil, false, nil
	}
	return e.Payload, true, nil
}

// SetCached stores the payload in Firestore with the given TTL.
func SetCached(ctx context.Context, client *firestore.Client, docID string, payload []byte, ttl time.Duration) error {
	entry := cacheEntry{
		Payload:   payload,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	_, err := client.Collection(apiCacheCollection).Doc(docID).Set(ctx, entry)
	return err
}
