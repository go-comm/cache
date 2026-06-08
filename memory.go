package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

const defaultBucketCap = 16 // Default capacity per bucket (pre-allocated)

// now returns current Unix timestamp in seconds.
// Used for expiration calculations.
func now() int64 {
	return time.Now().Unix()
}

// NewMemory creates a new in-memory cache instance.
// Supports lazy initialization and optional bucket capacity configuration.
//
// Usage:
//
//	cache.NewMemory()                          // default: 16 entries per bucket
//	cache.NewMemory(`{"cap": 32}`)            // custom: 32 entries per bucket
//	cache.NewMemory(`{"cap": -1}`)            // invalid: falls back to default (16)
//
// Note: Buckets are initialized on first write (lazy loading).
func NewMemory(args ...interface{}) Cache { // Parse optional JSON config: {"cap": N}
	bucketCap := defaultBucketCap
	if len(args) > 0 {
		if cfgStr, ok := args[0].(string); ok {
			var cfg struct {
				Cap int `json:"cap"`
			}
			if json.Unmarshal([]byte(cfgStr), &cfg) == nil && cfg.Cap > 0 {
				bucketCap = cfg.Cap
			}
		}
	}

	return &Memory{bucketCap: bucketCap}
}

// bucket is a sharded segment of the cache.
// Each bucket has its own lock to reduce contention.
type bucket struct {
	mu    sync.RWMutex
	_     [6]uint64 // Padding for cache-line alignment (reduce false sharing)
	m     *Memory
	store map[string]*Entry
}

// Memory is the main cache structure.
// Uses 256 sharded buckets for high concurrency.
type Memory struct {
	once          sync.Once                          // Ensures one-time initialization
	bucketCap     int                                // Capacity per bucket (for pre-allocation)
	buckets       [256]*bucket                       // Sharded storage
	expireHandler func(k interface{}, v interface{}) // Optional callback on expiration
}

// ensureStarted initializes buckets and starts the cleanup goroutine.
// Called lazily on first write operation. Thread-safe via sync.Once.
func (m *Memory) ensureStarted() {
	m.once.Do(func() {
		bcap := m.bucketCap
		if bcap <= 0 {
			bcap = defaultBucketCap
		}
		for i := 0; i < len(m.buckets); i++ {
			m.buckets[i] = &bucket{
				m:     m,
				store: make(map[string]*Entry, bcap), // Pre-allocate for performance
			}
		}
		go m.expireInLoop() // Start background cleanup
	})
}

// expireInLoop runs periodic expiration cleanup.
// Strategy: random sampling + full sweep to avoid CPU spikes.
func (m *Memory) expireInLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fullCleanupCounter := 0
	for range ticker.C {
		fullCleanupCounter++

		// Random sampling: clean 10 random buckets per tick (spread CPU load)
		for i := 0; i < 10; i++ {
			idx := rand.Intn(256)
			m.buckets[idx].cleanup()
		}

		// Full sweep: clean all buckets every 30 seconds (6 ticks)
		if fullCleanupCounter >= 6 {
			fullCleanupCounter = 0
			for i := 0; i < 256; i++ {
				m.buckets[i].cleanup()
			}
		}
	}
}

// cleanup removes all expired entries from this bucket.
// Must be called with bucket lock held.
func (b *bucket) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for k, v := range b.store {
		if v.Expired() {
			delete(b.store, k)
			// Async callback: notify handler without blocking cleanup
			if b.m.expireHandler != nil {
				go b.m.expireHandler(k, v.Value)
			}
		}
	}
}

// hashString computes DJB2 hash with proper bit mixing.
// Inlined by compiler for performance.
func hashString(s string) uint8 {
	var h uint32 = 5381
	for i := 0; i < len(s); i++ {
		h = h*33 + uint32(s[i])
	}
	return uint8(h ^ (h >> 8) ^ (h >> 16) ^ (h >> 24))
}

// hashInt computes hash for integer keys without string conversion.
func hashInt(v uint64) uint8 {
	var h uint32 = 5381
	for i := 0; i < 8; i++ {
		h = h*33 + uint32(v&0xff)
		v >>= 8
	}
	return uint8(h ^ (h >> 8) ^ (h >> 16) ^ (h >> 24))
}

func hashKey(k interface{}) (string, uint8) {
	switch d := k.(type) {
	case string:
		return d, hashString(d)
	case []byte:
		return string(d), hashString(BytesToStr(d))
	case int:
		return strconv.Itoa(d), hashInt(uint64(d))
	case int64:
		return strconv.FormatInt(d, 10), hashInt(uint64(d))
	case uint64:
		return strconv.FormatUint(d, 10), hashInt(d)
	default:
		var s string
		if ss, ok := k.(interface{ String() string }); ok {
			s = ss.String()
		} else {
			s = fmt.Sprintf("%v", d)
		}
		return s, hashString(s)
	}
}

// ==================== READ OPERATIONS (Lazy: no auto-init) ====================

// Get retrieves a value by key.
// Returns ErrNoKey if not found or expired.
// Note: Does NOT trigger initialization if cache is uninitialized.
func (m *Memory) Get(ctx context.Context, k interface{}) (interface{}, error) {
	// Fast path: uninitialized = empty cache
	if m.buckets[0] == nil {
		return nil, ErrNoKey
	}

	// Optimized: hashKey returns both string key and bucket index in one pass
	keyStr, idx := hashKey(k)
	b := m.buckets[idx]

	b.mu.RLock()
	e, ok := b.store[keyStr]
	b.mu.RUnlock()

	if !ok || e == nil || e.Expired() {
		return nil, ErrNoKey
	}
	return e.Value, nil
}

// GetAndTTL retrieves value and its remaining TTL.
// Returns ErrNoKey if not found or expired.
func (m *Memory) GetAndTTL(ctx context.Context, k interface{}) (interface{}, int64, error) {
	if m.buckets[0] == nil {
		return nil, 0, ErrNoKey
	}

	keyStr, idx := hashKey(k)
	b := m.buckets[idx]

	b.mu.RLock()
	e, ok := b.store[keyStr]
	b.mu.RUnlock()

	if !ok || e == nil {
		return nil, 0, ErrNoKey
	}

	ttl := e.TTL()
	if ttl == 0 {
		return nil, 0, ErrNoKey
	}
	return e.Value, ttl, nil
}

// TTL returns remaining TTL for a key.
// Returns ErrNoKey if not found or expired.
func (m *Memory) TTL(ctx context.Context, k interface{}) (int64, error) {
	if m.buckets[0] == nil {
		return 0, ErrNoKey
	}

	keyStr, idx := hashKey(k)
	b := m.buckets[idx]

	b.mu.RLock()
	e, ok := b.store[keyStr]
	b.mu.RUnlock()

	if !ok || e == nil {
		return 0, ErrNoKey
	}
	ttl := e.TTL()
	if ttl == 0 {
		return 0, ErrNoKey
	}
	return ttl, nil
}

// ==================== WRITE OPERATIONS (Trigger lazy init) ====================

// Put stores a value with no expiration.
// Triggers lazy initialization on first call.
func (m *Memory) Put(ctx context.Context, k interface{}, v interface{}) error {
	return m.PutEx(ctx, k, v, -1)
}

// PutEx stores a value with TTL in seconds.
// sec < 0 means never expire.
func (m *Memory) PutEx(ctx context.Context, k interface{}, v interface{}, sec int64) error {
	m.ensureStarted()

	keyStr, idx := hashKey(k)
	b := m.buckets[idx]

	nowTime := now()
	expiredAt := nowTime + sec
	if sec < 0 {
		expiredAt = -1
	}

	var err error
	if vv, ok := v.(Valuer); ok {
		if v, err = vv.Value(); err != nil {
			return err
		}
	}

	e := &Entry{CreatedAt: nowTime, ExpiredAt: expiredAt, Value: v}
	b.mu.Lock()
	b.store[keyStr] = e
	b.mu.Unlock()
	return nil
}

// Del removes a key from cache.
// Triggers expireHandler callback asynchronously if set.
func (m *Memory) Del(ctx context.Context, k interface{}) error {
	m.ensureStarted()

	keyStr, idx := hashKey(k)
	b := m.buckets[idx]

	var val interface{}
	b.mu.Lock()
	e, ok := b.store[keyStr]
	if ok && e != nil {
		delete(b.store, keyStr)
		val = e.Value
	}
	b.mu.Unlock()

	if !ok || e == nil {
		return ErrNoKey
	}
	// Async callback: non-blocking notification
	if h := b.m.expireHandler; h != nil {
		go h(k, val)
	}
	return nil
}

// Expire updates the expiration time for an existing key.
// sec < 0 sets to never expire.
func (m *Memory) Expire(ctx context.Context, k interface{}, sec int64) error {
	m.ensureStarted()

	keyStr, idx := hashKey(k)
	b := m.buckets[idx]

	b.mu.Lock()
	e, ok := b.store[keyStr]
	if ok && e != nil {
		if sec < 0 {
			e.ExpiredAt = -1
		} else {
			e.ExpiredAt = now() + sec
		}
	}
	b.mu.Unlock()

	if !ok || e == nil {
		return ErrNoKey
	}
	return nil
}

// Tx executes a function with exclusive access to a key's entry.
// The entry passed to fn implements Valuer, allowing TTL/Expire manipulation.
// Useful for atomic read-modify-write operations.
func (m *Memory) Tx(ctx context.Context, k interface{}, fn func(*Entry) error) error {
	m.ensureStarted()

	keyStr, idx := hashKey(k)
	b := m.buckets[idx]

	b.mu.Lock()
	defer b.mu.Unlock()

	e, ok := b.store[keyStr]
	if !ok || e == nil {
		return ErrNoKey
	}

	err := fn(e)
	if err != nil {
		return err
	}
	return nil
}

// ExpireHandler sets a callback function invoked when entries expire or are deleted.
// Callbacks run asynchronously to avoid blocking cache operations.
func (m *Memory) ExpireHandler(h func(k interface{}, v interface{})) {
	m.expireHandler = h
}

func (m *Memory) Scan(ctx context.Context, k interface{}, scan Scanner) error {
	v, err := m.Get(ctx, k)
	if err != nil {
		return err
	}
	return scan.Scan(v)
}

// ScanAndTTL retrieves value by key, scans it into scan, and returns the remaining TTL.
// Returns ErrNoKey if not found or expired.
func (m *Memory) ScanAndTTL(ctx context.Context, k interface{}, scan Scanner) (int64, error) {
	v, ttl, err := m.GetAndTTL(ctx, k)
	if err != nil {
		return 0, err
	}
	return ttl, scan.Scan(v)
}

// Range iterates over all entries in the cache.
// It snapshots one bucket at a time, then calls fn for each entry in that bucket
// without holding any locks, so the callback can safely perform cache writes.
// The iteration stops if fn returns an error, and that error is returned.
// Safe to call on uninitialized cache (no-op).
func (m *Memory) Range(ctx context.Context, fn func(k interface{}, v interface{}) error) error {
	if m.buckets[0] == nil {
		return nil
	}

	type pair struct {
		k interface{}
		v interface{}
	}
	for _, b := range m.buckets {
		// Snapshot one bucket under lock.
		b.mu.RLock()
		snapshot := make([]pair, 0, len(b.store))
		for k, e := range b.store {
			if !e.Expired() {
				snapshot = append(snapshot, pair{k, e.Value})
			}
		}
		b.mu.RUnlock()

		// Iterate over this bucket's snapshot without holding the lock.
		for _, p := range snapshot {
			if err := fn(p.k, p.v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Clear removes all entries from the cache.
// Safe to call on uninitialized cache (no-op).
func (m *Memory) Clear(ctx context.Context) error {
	if m.buckets[0] == nil {
		return nil
	}
	bcap := m.bucketCap
	if bcap <= 0 {
		bcap = defaultBucketCap
	}
	for _, b := range m.buckets {
		b.mu.Lock()
		// Replace map to release old entries for GC
		b.store = make(map[string]*Entry, bcap)
		b.mu.Unlock()
	}
	return nil
}
