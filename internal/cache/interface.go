package cache

import (
	"context"
	"time"

	"github.com/ntentasd/nostradamus-api/pkg/types"
)

// Cache defines the general caching for the api.
// It abstracts time-series (ZSET) and key-values (SET).
type Cache interface {
	// Store stores a single reading (usually time-series data)
	Store(prefix string, entry types.Entry) error

	// FetchLast retrieves the N most recent entries from a sorted cache
	FetchLast(prefix string, n int) ([]types.Entry, error)

	// StoreAggregate caches a computed aggregate with a TTL
	StoreAggregate(ctx context.Context, key string, data any, ttl time.Duration) error

	// FetchAggregate retrieves an aggregate from cache
	FetchAggregate(ctx context.Context, key string) ([]byte, error)

	// Ping checks cache connection
	Ping(ctx context.Context) error

	// Close gracefully closes any connections
	Close()
}
