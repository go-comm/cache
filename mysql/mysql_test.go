package mysql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/go-comm/cache"
	_ "github.com/go-sql-driver/mysql"
)

// Set CACHE_MYSQL_DSN to enable these tests.
// Example: root:password@tcp(127.0.0.1:3306)/test?parseTime=true
func getTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("CACHE_MYSQL_DSN")
	if dsn == "" {
		t.Skip("CACHE_MYSQL_DSN not set, skipping MySQL cache tests")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return db
}

const testTable = "cache_test_tmp"

func newTestCache(t *testing.T) (*MysqlCache, func()) {
	db := getTestDB(t)
	c, err := New(db, testTable, WithAutoCreateTable(), WithNoExpireCheck())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cleanup := func() {
		c.Clear(context.Background())
		c.Close()
		db.Close()
	}
	return c, cleanup
}

// ============================================================================
// New / Options
// ============================================================================

func TestNewNilDB(t *testing.T) {
	_, err := New(nil, "t")
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}

func TestNewEmptyTable(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	_, err := New(db, "")
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestNewWithAutoCreate(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()
	tbl := "cache_autocreate_" + time.Now().Format("150405")
	c, err := New(db, tbl, WithAutoCreateTable())
	if err != nil {
		t.Fatalf("New with auto create: %v", err)
	}
	defer func() {
		c.Close()
		db.Exec("DROP TABLE IF EXISTS " + tbl)
		db.Close()
	}()
	// verify table exists by doing a Put
	if err := c.Put(context.Background(), "k", "v"); err != nil {
		t.Fatalf("Put after auto create: %v", err)
	}
}

// ============================================================================
// Put / Get
// ============================================================================

func TestMysqlPutAndGet(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	if err := c.Put(ctx, "k1", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	v, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatal(err)
	}
	bs, ok := v.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", v)
	}
	if string(bs) != "hello" {
		t.Fatalf("expected hello, got %s", bs)
	}
}

func TestMysqlGetNotFound(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()

	_, err := c.Get(context.Background(), "not_exist")
	if err != cache.ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

// ============================================================================
// PutEx / GetAndTTL / TTL
// ============================================================================

func TestMysqlPutExAndGetAndTTL(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	if err := c.PutEx(ctx, "k2", []byte("world"), 60); err != nil {
		t.Fatal(err)
	}
	v, ttl, err := c.GetAndTTL(ctx, "k2")
	if err != nil {
		t.Fatal(err)
	}
	if string(v.([]byte)) != "world" {
		t.Fatalf("expected world, got %v", v)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("expected ttl in (0,60], got %d", ttl)
	}
}

func TestMysqlTTL(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	// not exist
	_, err := c.TTL(ctx, "not_exist")
	if err != cache.ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}

	// with TTL
	c.PutEx(ctx, "ttl_k", []byte("v"), 30)
	ttl, err := c.TTL(ctx, "ttl_k")
	if err != nil {
		t.Fatal(err)
	}
	if ttl <= 0 || ttl > 30 {
		t.Fatalf("expected ttl in (0,30], got %d", ttl)
	}

	// never expire
	c.Put(ctx, "no_ttl_k", []byte("v"))
	ttl, err = c.TTL(ctx, "no_ttl_k")
	if err != nil {
		t.Fatal(err)
	}
	if ttl != -1 {
		t.Fatalf("expected -1 for never expire, got %d", ttl)
	}
}

// ============================================================================
// Del
// ============================================================================

func TestMysqlDel(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, "del_k", []byte("value"))
	if err := c.Del(ctx, "del_k"); err != nil {
		t.Fatal(err)
	}
	_, err := c.Get(ctx, "del_k")
	if err != cache.ErrNoKey {
		t.Fatalf("expected ErrNoKey after Del, got %v", err)
	}

	// delete non-existent
	err = c.Del(ctx, "not_exist")
	if err != cache.ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

// ============================================================================
// Expire
// ============================================================================

func TestMysqlExpire(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, "exp_k", []byte("v"))
	if err := c.Expire(ctx, "exp_k", 5); err != nil {
		t.Fatal(err)
	}
	ttl, _ := c.TTL(ctx, "exp_k")
	if ttl <= 0 || ttl > 5 {
		t.Fatalf("expected ttl in (0,5], got %d", ttl)
	}

	// set to never expire
	if err := c.Expire(ctx, "exp_k", -1); err != nil {
		t.Fatal(err)
	}
	ttl, _ = c.TTL(ctx, "exp_k")
	if ttl != -1 {
		t.Fatalf("expected -1, got %d", ttl)
	}

	// non-existent key
	err := c.Expire(ctx, "not_exist", 10)
	if err != cache.ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

// ============================================================================
// Clear
// ============================================================================

func TestMysqlClear(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, "a", []byte("1"))
	c.Put(ctx, "b", []byte("2"))
	c.Put(ctx, "c", []byte("3"))

	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := c.Get(ctx, "a")
	if err != cache.ErrNoKey {
		t.Fatalf("expected ErrNoKey after Clear, got %v", err)
	}
}

// ============================================================================
// Scan / ScanAndTTL
// ============================================================================

func TestMysqlScan(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, "scan_k", []byte(`{"name":"test","age":25}`))

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	var u User
	err := c.Scan(ctx, "scan_k", cache.DecodeScanner(&u))
	if err != nil {
		t.Fatal(err)
	}
	if u.Name != "test" || u.Age != 25 {
		t.Fatalf("expected {test 25}, got %+v", u)
	}
}

func TestMysqlScanAndTTL(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.PutEx(ctx, "st_k", []byte(`{"n":42}`), 60)

	var out struct {
		N int `json:"n"`
	}
	ttl, err := c.ScanAndTTL(ctx, "st_k", cache.DecodeScanner(&out))
	if err != nil {
		t.Fatal(err)
	}
	if out.N != 42 {
		t.Fatalf("expected 42, got %d", out.N)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("expected ttl in (0,60], got %d", ttl)
	}
}

// ============================================================================
// Tx
// ============================================================================

func TestMysqlTx(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.PutEx(ctx, "tx_k", []byte(`{"count":0}`), 60)

	err := c.Tx(ctx, "tx_k", func(e *cache.Entry) error {
		e.Expire(120)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ttl, _ := c.TTL(ctx, "tx_k")
	if ttl <= 0 || ttl > 120 {
		t.Fatalf("expected ttl in (0,120], got %d", ttl)
	}

	// non-existent key
	err = c.Tx(ctx, "not_exist", func(e *cache.Entry) error {
		return nil
	})
	if err != cache.ErrNoKey {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

// ============================================================================
// ExpireHandler
// ============================================================================

func TestMysqlExpireHandler(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	done := make(chan struct{}, 1)
	c.ExpireHandler(func(k interface{}, v interface{}) {
		close(done)
	})

	c.Put(ctx, "eh_k", []byte("val"))
	c.Del(ctx, "eh_k")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expireHandler not called")
	}
}

// ============================================================================
// Key Types
// ============================================================================

func TestMysqlIntKey(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, 100, []byte("int_val"))
	v, err := c.Get(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if string(v.([]byte)) != "int_val" {
		t.Fatalf("expected int_val, got %v", v)
	}
}

func TestMysqlBytesKey(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, []byte("bk"), []byte("bv"))
	v, err := c.Get(ctx, []byte("bk"))
	if err != nil {
		t.Fatal(err)
	}
	if string(v.([]byte)) != "bv" {
		t.Fatalf("expected bv, got %v", v)
	}
}

// ============================================================================
// Value Types — encode/decode round-trip
// ============================================================================

func TestMysqlEncodeDecodeRoundTrip(t *testing.T) {
	c, cleanup := newTestCache(t)
	defer cleanup()
	ctx := context.Background()

	type Config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	// Store as JSON via EncodeValuer
	if err := c.Put(ctx, "cfg", cache.EncodeValuer(&Config{Host: "localhost", Port: 3306})); err != nil {
		t.Fatal(err)
	}

	// Read back via DecodeScanner
	var cfg Config
	if err := c.Scan(ctx, "cfg", cache.DecodeScanner(&cfg)); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "localhost" || cfg.Port != 3306 {
		t.Fatalf("expected {localhost 3306}, got %+v", cfg)
	}
}

// ============================================================================
// Range (isolated table per test to avoid data pollution)
// ============================================================================

func newRangeTestCache(t *testing.T) (*MysqlCache, func()) {
	db := getTestDB(t)
	tbl := "cache_range_" + time.Now().Format("150405") + "_" + t.Name()[len("TestMysqlRange"):]
	c, err := New(db, tbl, WithAutoCreateTable(), WithNoExpireCheck())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cleanup := func() {
		c.Close()
		db.Exec("DROP TABLE IF EXISTS " + tbl)
		db.Close()
	}
	return c, cleanup
}

func TestMysqlRangeBasic(t *testing.T) {
	c, cleanup := newRangeTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, "a", []byte("1"))
	c.Put(ctx, "b", []byte("2"))
	c.Put(ctx, "c", []byte("3"))

	collected := make(map[string]string)
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		collected[k.(string)] = string(v.([]byte))
		return nil
	})
	if err != nil {
		t.Fatal("Range failed:", err)
	}
	if len(collected) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(collected))
	}
	if collected["a"] != "1" || collected["b"] != "2" || collected["c"] != "3" {
		t.Fatalf("unexpected values: %v", collected)
	}
}

func TestMysqlRangeEmpty(t *testing.T) {
	c, cleanup := newRangeTestCache(t)
	defer cleanup()
	ctx := context.Background()

	count := 0
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal("Range on empty table should not error:", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 entries, got %d", count)
	}
}

func TestMysqlRangeSkipExpired(t *testing.T) {
	c, cleanup := newRangeTestCache(t)
	defer cleanup()
	ctx := context.Background()

	c.Put(ctx, "fresh", []byte("ok"))
	c.PutEx(ctx, "stale", []byte("nope"), 0)
	time.Sleep(1500 * time.Millisecond) // ensure stale entry is definitively expired

	collected := make(map[string]string)
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		collected[k.(string)] = string(v.([]byte))
		return nil
	})
	if err != nil {
		t.Fatal("Range failed:", err)
	}
	if len(collected) != 1 {
		t.Fatalf("expected 1 non-expired entry, got %d", len(collected))
	}
	if collected["fresh"] != "ok" {
		t.Fatalf("expected fresh=ok, got %v", collected)
	}
	if _, exists := collected["stale"]; exists {
		t.Fatal("stale entry should be skipped")
	}
}

func TestMysqlRangeEarlyExit(t *testing.T) {
	c, cleanup := newRangeTestCache(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		c.Put(ctx, string(rune('a'+i)), []byte{byte(i)})
	}

	count := 0
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		count++
		if count >= 2 {
			return context.Canceled
		}
		return nil
	})
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count=2, got %d", count)
	}
}

func TestMysqlRangeCursorPagination(t *testing.T) {
	c, cleanup := newRangeTestCache(t)
	defer cleanup()
	ctx := context.Background()

	n := 10
	for i := 0; i < n; i++ {
		c.Put(ctx, "cursor_"+string(rune('a'+i)), []byte{byte(i)})
	}

	count := 0
	err := c.Range(ctx, func(k interface{}, v interface{}) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal("Range failed:", err)
	}
	if count != n {
		t.Fatalf("expected %d entries, got %d", n, count)
	}
}
