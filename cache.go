package cache

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
)

var ErrNoKey = errors.New("cache: no key")

type Valuer = driver.Valuer

type Scanner = sql.Scanner

// Cache defines the interface for a thread-safe cache storage.
// Implementations may use in-memory maps, sharded locks, or other backends.
// Keys can be of type string, []byte, int, int64, uint64, or types with a String() method.
// Values can be any type, and optional expiration (TTL) is supported via PutEx.
type Cache interface {
	// Get retrieves the value for a given key.
	// Returns ErrNoKey if the key does not exist or has expired.
	Get(ctx context.Context, k interface{}) (interface{}, error)

	// GetAndTTL returns the value and remaining TTL (seconds).
	// Behavior:
	//   - Key exists and not expired: returns (value, ttl, nil), where ttl = -1 (never expires) or >0.
	//   - Key does not exist or has expired: returns (nil, 0, ErrNoKey).
	GetAndTTL(ctx context.Context, k interface{}) (interface{}, int64, error)

	// Scan retrieves the value for a key and scans it into the provided Scanner (e.g., sql.Scanner).
	// This is useful for automatic deserialization.
	Scan(ctx context.Context, k interface{}, scan Scanner) error

	// ScanAndTTL retrieves the value, scans it into the Scanner, and returns the remaining TTL.
	// Behavior:
	//   - Key exists and not expired: returns (ttl, nil), where ttl = -1 (never expires) or >0.
	//   - Key does not exist or has expired: returns (0, ErrNoKey).
	ScanAndTTL(ctx context.Context, k interface{}, scan Scanner) (int64, error)

	// Put stores a value for a key with no expiration.
	Put(ctx context.Context, k interface{}, v interface{}) error

	// PutEx stores a value for a key with a TTL (in seconds). If sec is negative, the entry never expires.
	PutEx(ctx context.Context, k interface{}, v interface{}, sec int64) error

	// Del removes the key-value pair from the cache.
	// If an ExpireHandler is set, it will be called asynchronously with the key and value.
	Del(ctx context.Context, k interface{}) error

	// TTL returns the remaining time-to-live (seconds) for the key.
	// Behavior:
	//   - Key exists and not expired: returns (ttl, nil), where ttl = -1 (never expires) or >0.
	//   - Key does not exist or has expired: returns (0, ErrNoKey).
	TTL(ctx context.Context, k interface{}) (int64, error)

	// Expire updates the expiration time for an existing key.
	// If sec is negative, the key becomes non-expiring. Returns ErrNoKey if key does not exist.
	Expire(ctx context.Context, k interface{}, sec int64) error

	// Tx executes the given function under a write lock for the specified key.
	// The function receives the current Entry, allowing atomic read-modify-write operations.
	Tx(ctx context.Context, k interface{}, fn func(*Entry) error) error

	// ExpireHandler sets a callback that is triggered when an entry expires or is deleted.
	// The callback runs asynchronously and should not block.
	ExpireHandler(h func(k interface{}, v interface{}))

	// Range iterates over all non-expired entries in the cache.
	// The function fn is called for each entry; if fn returns an error, iteration stops.
	// The iteration does not hold any locks, so the cache can be mutated during iteration.
	Range(ctx context.Context, fn func(k interface{}, v interface{}) error) error

	// Clear removes all entries from the cache.
	Clear(ctx context.Context) error
}

type Entry struct {
	CreatedAt int64       // Creation timestamp (Unix seconds)
	ExpiredAt int64       // Expiration timestamp (Unix seconds), -1 = never expire
	Value     interface{} // Stored value
}

func (e *Entry) Expired() bool {
	if e == nil {
		return true
	}
	if e.ExpiredAt < 0 {
		return false // Never expires
	}
	return e.ExpiredAt <= now()
}

// TTL returns the remaining TTL in seconds for the entry.
// Returns -1 if the entry never expires. Returns 0 if the entry is nil or has expired.
// This method does not trigger any deletion; it just computes the value.
func (e *Entry) TTL() int64 {
	if e == nil {
		return 0
	}
	if e.ExpiredAt < 0 {
		return -1
	}
	ttl := e.ExpiredAt - now()
	if ttl < 0 {
		ttl = 0
	}
	return ttl
}

func (e *Entry) Expire(sec int64) {
	if e == nil {
		return
	}
	if sec < 0 {
		e.ExpiredAt = -1
	} else {
		e.ExpiredAt = now() + sec
	}
}
