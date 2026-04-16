package models

import "time"

// CacheEntry is the Firestore document format used for cached API responses.
type CacheEntry struct {
	Payload []byte `firestore:"payload"` // JSON bytes (stored as Firestore bytes) containing the marshaled
	// response for the cached endpoint
	CreatedAt time.Time `firestore:"createdAt"` // the time it was cached/created
	ExpiresAt time.Time `firestore:"expiresAt"` // the time it should be considered stale.
}
