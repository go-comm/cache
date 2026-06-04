package cache

import (
	"context"
	"testing"
	"time"
)

func newCache() Cache {
	return NewMemory()
}

// ==================== Entry ====================

func TestEntryNil(t *testing.T) {
	var e *Entry
	if !e.Expired() {
		t.Fatal("nil entry should be expired")
	}
	if e.TTL() != 0 {
		t.Fatalf("nil entry TTL should be 0, got %d", e.TTL())
	}
	e.Expire(10) // should not panic
}

func TestEntryNeverExpire(t *testing.T) {
	e := &Entry{ExpiredAt: -1, Value: "x"}
	if e.Expired() {
		t.Fatal("never-expire entry should not be expired")
	}
	if e.TTL() != -1 {
		t.Fatalf("never-expire TTL should be -1, got %d", e.TTL())
	}
}

func TestEntryExpired(t *testing.T) {
	e := &Entry{ExpiredAt: now() - 10, Value: "x"}
	if !e.Expired() {
		t.Fatal("past entry should be expired")
	}
	if e.TTL() != 0 {
		t.Fatalf("expired TTL should be 0, got %d", e.TTL())
	}
}

func TestEntryExpire(t *testing.T) {
	e := &Entry{ExpiredAt: now() + 100, Value: "x"}
	// set to never expire
	e.Expire(-1)
	if e.ExpiredAt != -1 {
		t.Fatalf("expected -1, got %d", e.ExpiredAt)
	}

	// set to 5s
	e.Expire(5)
	if e.ExpiredAt <= now() {
		t.Fatalf("expected future ExpiredAt, got %d", e.ExpiredAt)
	}
}

func TestEntryValue(t *testing.T) {
	e := &Entry{Value: 42}
	if e.Value != 42 {
		t.Fatalf("expected 42, got %v", e.Value)
	}
}

// ==================== CRUD ====================

func TestPutAndGet(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	if err := c.Put(ctx, "k1", "hello"); err != nil {
		t.Fatal("Put failed:", err)
	}
	v, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if v != "hello" {
		t.Fatalf("expected hello, got %v", v)
	}
}

func TestGetNotFound(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	_, err := c.Get(ctx, "not_exist")
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

func TestPutExAndGetAndTTL(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	if err := c.PutEx(ctx, "k2", "world", 10); err != nil {
		t.Fatal("PutEx failed:", err)
	}
	v, ttl, err := c.GetAndTTL(ctx, "k2")
	if err != nil {
		t.Fatal("GetAndTTL failed:", err)
	}
	if v != "world" {
		t.Fatalf("expected world, got %v", v)
	}
	if ttl <= 0 || ttl > 10 {
		t.Fatalf("expected ttl in (0,10], got %d", ttl)
	}
}

func TestExpiration(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.PutEx(ctx, "k3", "temp", 1)
	time.Sleep(1500 * time.Millisecond)

	_, err := c.Get(ctx, "k3")
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey after expiration, got %v", err)
	}
}

func TestTTL(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	// not exist
	_, err := c.TTL(ctx, "not_exist")
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}

	// with TTL
	c.PutEx(ctx, "ttl_k", "v", 30)
	ttl, err := c.TTL(ctx, "ttl_k")
	if err != nil {
		t.Fatal("TTL failed:", err)
	}
	if ttl <= 0 || ttl > 30 {
		t.Fatalf("expected ttl in (0,30], got %d", ttl)
	}

	// never expire
	c.Put(ctx, "no_ttl_k", "v")
	ttl, err = c.TTL(ctx, "no_ttl_k")
	if err != nil {
		t.Fatal("TTL failed:", err)
	}
	if ttl != -1 {
		t.Fatalf("expected -1 for never expire, got %d", ttl)
	}
}

func TestDel(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "del_k", "value")
	if err := c.Del(ctx, "del_k"); err != nil {
		t.Fatal("Del failed:", err)
	}
	_, err := c.Get(ctx, "del_k")
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey after Del, got %v", err)
	}

	// delete non-existent
	err = c.Del(ctx, "not_exist")
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

func TestExpire(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "exp_k", "v")

	if err := c.Expire(ctx, "exp_k", 5); err != nil {
		t.Fatal("Expire failed:", err)
	}
	ttl, _ := c.TTL(ctx, "exp_k")
	if ttl <= 0 || ttl > 5 {
		t.Fatalf("expected ttl in (0,5], got %d", ttl)
	}

	// set to never expire
	if err := c.Expire(ctx, "exp_k", -1); err != nil {
		t.Fatal("Expire failed:", err)
	}
	ttl, _ = c.TTL(ctx, "exp_k")
	if ttl != -1 {
		t.Fatalf("expected -1, got %d", ttl)
	}

	// non-existent key
	err := c.Expire(ctx, "not_exist", 10)
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

func TestClear(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "a", 1)
	c.Put(ctx, "b", 2)
	c.Put(ctx, "c", 3)

	if err := c.Clear(ctx); err != nil {
		t.Fatal("Clear failed:", err)
	}
	_, err := c.Get(ctx, "a")
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey after Clear, got %v", err)
	}
}

func TestTx(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "tx_k", "old")

	err := c.Tx(ctx, "tx_k", func(e *Entry) error {
		if e.Value != "old" {
			t.Fatalf("expected old, got %v", e.Value)
		}
		e.Expire(100)
		return nil
	})
	if err != nil {
		t.Fatal("Tx failed:", err)
	}
	ttl, _ := c.TTL(ctx, "tx_k")
	if ttl <= 0 || ttl > 100 {
		t.Fatalf("expected ttl in (0,100], got %d", ttl)
	}

	// non-existent key
	err = c.Tx(ctx, "not_exist", func(e *Entry) error {
		return nil
	})
	if err != ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

func TestExpireHandler(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	done := make(chan struct{}, 1)
	c.ExpireHandler(func(k interface{}, v interface{}) {
		if k != "eh_k" {
			t.Errorf("expected eh_k, got %v", k)
		}
		close(done)
	})

	// Use Del to trigger handler deterministically
	c.Put(ctx, "eh_k", "val")
	c.Del(ctx, "eh_k")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expireHandler not called within timeout")
	}
}

func TestScan(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "scan_k", "hello")
	var s string
	err := c.Scan(ctx, "scan_k", StringScanner(&s))
	if err != nil {
		t.Fatal("Scan failed:", err)
	}
	if s != "hello" {
		t.Fatalf("expected hello, got %s", s)
	}
}

func TestScanAndTTL(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.PutEx(ctx, "st_k", 42, 60)
	var n int
	ttl, err := c.ScanAndTTL(ctx, "st_k", IntScanner(&n))
	if err != nil {
		t.Fatal("ScanAndTTL failed:", err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("expected ttl in (0,60], got %d", ttl)
	}
}

// ==================== Key Types ====================

func TestIntKey(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, 100, "int_val")
	v, err := c.Get(ctx, 100)
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if v != "int_val" {
		t.Fatalf("expected int_val, got %v", v)
	}
}

func TestInt64Key(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, int64(999), "int64_val")
	v, err := c.Get(ctx, int64(999))
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if v != "int64_val" {
		t.Fatalf("expected int64_val, got %v", v)
	}
}

func TestUint64Key(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, uint64(888), "uint64_val")
	v, err := c.Get(ctx, uint64(888))
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if v != "uint64_val" {
		t.Fatalf("expected uint64_val, got %v", v)
	}
}

// ==================== Init & Config ====================

func TestLazyInit(t *testing.T) {
	c := newCache().(*Memory)
	ctx := context.Background()

	if c.buckets[0] != nil {
		t.Fatal("expected buckets nil before first write")
	}

	// Get should not trigger init
	c.Get(ctx, "lazy")
	if c.buckets[0] != nil {
		t.Fatal("expected buckets still nil after Get")
	}

	// Put triggers init
	c.Put(ctx, "lazy", "v")
	if c.buckets[0] == nil {
		t.Fatal("expected buckets initialized after Put")
	}
}

func TestNewMemoryWithConfig(t *testing.T) {
	c := NewMemory(`{"cap": 32}`)
	ctx := context.Background()

	c.Put(ctx, "cfg_test", "v")
	v, err := c.Get(ctx, "cfg_test")
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if v != "v" {
		t.Fatalf("expected v, got %v", v)
	}
}

func TestNewMemoryInvalidConfig(t *testing.T) {
	// should fall back to default cap
	c := NewMemory(`{"cap": -1}`)
	ctx := context.Background()

	c.Put(ctx, "icfg", "v")
	v, err := c.Get(ctx, "icfg")
	if err != nil {
		t.Fatal("Get failed:", err)
	}
	if v != "v" {
		t.Fatalf("expected v, got %v", v)
	}
}

// ==================== Concurrency ====================

func TestConcurrent(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			key := "conc_" + string(rune('A'+n%26)) + string(rune('0'+n/26))
			c.PutEx(ctx, key, n, 10)
			c.Get(ctx, key)
			c.TTL(ctx, key)
			c.Del(ctx, key)
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}

// ==================== Range ====================

func TestRangeBasic(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	// populate
	c.Put(ctx, "a", 1)
	c.Put(ctx, "b", 2)
	c.Put(ctx, "c", 3)

	collected := make(map[string]int)
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		collected[k.(string)] = v.(int)
		return nil
	})
	if err != nil {
		t.Fatal("Range failed:", err)
	}
	if len(collected) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(collected))
	}
	if collected["a"] != 1 || collected["b"] != 2 || collected["c"] != 3 {
		t.Fatalf("unexpected values: %v", collected)
	}
}

func TestRangeEmptyCache(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	count := 0
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal("Range on empty cache should not error:", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 entries, got %d", count)
	}
}

func TestRangeUninitialized(t *testing.T) {
	c := NewMemory() // no writes, buckets nil
	ctx := context.Background()

	count := 0
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal("Range on uninitialized cache should not error:", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 entries, got %d", count)
	}
}

func TestRangeSkipExpired(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "fresh", 1)
	c.PutEx(ctx, "stale", 2, 0) // immediate expiry
	time.Sleep(10 * time.Millisecond)

	collected := make(map[string]int)
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		collected[k.(string)] = v.(int)
		return nil
	})
	if err != nil {
		t.Fatal("Range failed:", err)
	}
	if len(collected) != 1 {
		t.Fatalf("expected 1 non-expired entry, got %d", len(collected))
	}
	if collected["fresh"] != 1 {
		t.Fatalf("expected fresh=1, got %v", collected)
	}
	if _, exists := collected["stale"]; exists {
		t.Fatal("stale entry should be skipped")
	}
}

func TestRangeEarlyExit(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		c.Put(ctx, string(rune('a'+i)), i)
	}

	count := 0
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		count++
		if count >= 3 {
			return context.Canceled // simulate early exit
		}
		return nil
	})
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	// We can't assert exact count due to sharding order, but it should be >= 3 and < 10
	if count < 3 || count > 10 {
		t.Fatalf("expected count in [3,10], got %d", count)
	}
}

func TestRangeCallbackCanWrite(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "a", 1)
	c.Put(ctx, "b", 2)

	// Writes in the callback should not deadlock.
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		c.Put(ctx, "c", 3)
		c.Del(ctx, k)
		return nil
	})
	if err != nil {
		t.Fatal("Range with write in callback failed:", err)
	}

	// original keys were deleted
	_, err = c.Get(ctx, "a")
	if err != ErrNoKey {
		t.Fatalf("expected a to be deleted")
	}
	_, err = c.Get(ctx, "b")
	if err != ErrNoKey {
		t.Fatalf("expected b to be deleted")
	}
	// "c" may or may not exist depending on bucket iteration order — no assertion needed.
}
