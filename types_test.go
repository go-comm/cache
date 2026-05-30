package cache

import (
	"context"
	"testing"
)

func TestIntScanner(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "si", 42)
	var i int
	if err := c.Scan(ctx, "si", IntScanner(&i)); err != nil {
		t.Fatal(err)
	}
	if i != 42 {
		t.Fatalf("expected 42, got %d", i)
	}
}

func TestIntScannerFromFloat64(t *testing.T) {
	var i int
	if err := IntScanner(&i).Scan(float64(3.9)); err != nil {
		t.Fatal(err)
	}
	if i != 3 {
		t.Fatalf("expected 3, got %d", i)
	}
}

func TestIntScannerFromString(t *testing.T) {
	var i int
	if err := IntScanner(&i).Scan("123"); err != nil {
		t.Fatal(err)
	}
	if i != 123 {
		t.Fatalf("expected 123, got %d", i)
	}
}

func TestIntScannerFromBytes(t *testing.T) {
	var i int
	if err := IntScanner(&i).Scan([]byte("456")); err != nil {
		t.Fatal(err)
	}
	if i != 456 {
		t.Fatalf("expected 456, got %d", i)
	}
}

func TestIntScannerInvalidString(t *testing.T) {
	var i int
	err := IntScanner(&i).Scan("abc")
	if err == nil {
		t.Fatal("expected error for invalid string")
	}
}

func TestInt64Scanner(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "si64", int64(99))
	var i64 int64
	if err := c.Scan(ctx, "si64", Int64Scanner(&i64)); err != nil {
		t.Fatal(err)
	}
	if i64 != 99 {
		t.Fatalf("expected 99, got %d", i64)
	}
}

func TestInt64ScannerFromInt(t *testing.T) {
	var i64 int64
	if err := Int64Scanner(&i64).Scan(int(77)); err != nil {
		t.Fatal(err)
	}
	if i64 != 77 {
		t.Fatalf("expected 77, got %d", i64)
	}
}

func TestInt64ScannerFromString(t *testing.T) {
	var i64 int64
	if err := Int64Scanner(&i64).Scan("9876543210"); err != nil {
		t.Fatal(err)
	}
	if i64 != 9876543210 {
		t.Fatalf("expected 9876543210, got %d", i64)
	}
}

func TestInt64ScannerInvalidString(t *testing.T) {
	var i64 int64
	err := Int64Scanner(&i64).Scan("not_a_number")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUint64Scanner(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "su64", uint64(77))
	var u64 uint64
	if err := c.Scan(ctx, "su64", Uint64Scanner(&u64)); err != nil {
		t.Fatal(err)
	}
	if u64 != 77 {
		t.Fatalf("expected 77, got %d", u64)
	}
}

func TestUint64ScannerFromInt(t *testing.T) {
	var u64 uint64
	if err := Uint64Scanner(&u64).Scan(int(55)); err != nil {
		t.Fatal(err)
	}
	if u64 != 55 {
		t.Fatalf("expected 55, got %d", u64)
	}
}

func TestUint64ScannerFromString(t *testing.T) {
	var u64 uint64
	if err := Uint64Scanner(&u64).Scan("18446744073709551615"); err != nil {
		t.Fatal(err)
	}
	if u64 != 18446744073709551615 {
		t.Fatalf("expected max uint64, got %d", u64)
	}
}

func TestUint64ScannerInvalidString(t *testing.T) {
	var u64 uint64
	err := Uint64Scanner(&u64).Scan("xyz")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBoolScanner(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "sb", true)
	var b bool
	if err := c.Scan(ctx, "sb", BoolScanner(&b)); err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Fatal("expected true")
	}
}

func TestBoolScannerFromInt(t *testing.T) {
	var b bool
	if err := BoolScanner(&b).Scan(int(1)); err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Fatal("expected true")
	}
	if err := BoolScanner(&b).Scan(int(0)); err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("expected false")
	}
}

func TestBoolScannerFromString(t *testing.T) {
	var b bool
	if err := BoolScanner(&b).Scan("true"); err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Fatal("expected true")
	}
	if err := BoolScanner(&b).Scan("0"); err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("expected false")
	}
}

func TestBoolScannerInvalidString(t *testing.T) {
	var b bool
	err := BoolScanner(&b).Scan("maybe")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStringScanner(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	c.Put(ctx, "ss", "text")
	var s string
	if err := c.Scan(ctx, "ss", StringScanner(&s)); err != nil {
		t.Fatal(err)
	}
	if s != "text" {
		t.Fatalf("expected text, got %s", s)
	}
}

func TestStringScannerFromBytes(t *testing.T) {
	var s string
	if err := StringScanner(&s).Scan([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Fatalf("expected hello, got %s", s)
	}
}

func TestStringScannerUnsupported(t *testing.T) {
	var s string
	err := StringScanner(&s).Scan(12345)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestAnyScanner(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	type User struct {
		Name string
		Age  int
	}
	c.Put(ctx, "user", &User{Name: "test", Age: 25})

	var u User
	if err := c.Scan(ctx, "user", AnyScanner(&u)); err != nil {
		t.Fatal(err)
	}
	if u.Name != "test" || u.Age != 25 {
		t.Fatalf("expected {test 25}, got %+v", u)
	}
}

func TestDecodeScanner(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	type Config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	c.Put(ctx, "cfg", []byte(`{"host":"localhost","port":8080}`))

	var cfg Config
	if err := c.Scan(ctx, "cfg", DecodeScanner(&cfg)); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "localhost" || cfg.Port != 8080 {
		t.Fatalf("expected {localhost 8080}, got %+v", cfg)
	}
}

func TestDecodeScannerFromString(t *testing.T) {
	type S struct {
		A string `json:"a"`
	}
	var s S
	if err := DecodeScanner(&s).Scan(`{"a":"b"}`); err != nil {
		t.Fatal(err)
	}
	if s.A != "b" {
		t.Fatalf("expected b, got %s", s.A)
	}
}

func TestDecodeScannerUnsupported(t *testing.T) {
	var s struct{}
	err := DecodeScanner(&s).Scan(123)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnyValuer(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	type User struct {
		Name string
	}
	v := AnyValuer(&User{Name: "test"})
	if err := c.Put(ctx, "av", v); err != nil {
		t.Fatal("Put with AnyValuer failed:", err)
	}
	// retrieve raw value
	raw, err := c.Get(ctx, "av")
	if err != nil {
		t.Fatal(err)
	}
	u, ok := raw.(*User)
	if !ok {
		t.Fatalf("expected *User, got %T", raw)
	}
	if u.Name != "test" {
		t.Fatalf("expected test, got %s", u.Name)
	}
}

func TestEncodeValuer(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	type Config struct {
		K string `json:"k"`
	}
	v := EncodeValuer(&Config{K: "v"})
	if err := c.Put(ctx, "ev", v); err != nil {
		t.Fatal("Put with EncodeValuer failed:", err)
	}
	// retrieve and decode
	var cfg Config
	if err := c.Scan(ctx, "ev", DecodeScanner(&cfg)); err != nil {
		t.Fatal(err)
	}
	if cfg.K != "v" {
		t.Fatalf("expected v, got %s", cfg.K)
	}
}

func TestIntScannerUnsupported(t *testing.T) {
	var i int
	err := IntScanner(&i).Scan(struct{}{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestInt64ScannerUnsupported(t *testing.T) {
	var i int64
	err := Int64Scanner(&i).Scan(struct{}{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestUint64ScannerUnsupported(t *testing.T) {
	var u uint64
	err := Uint64Scanner(&u).Scan(struct{}{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestBoolScannerUnsupported(t *testing.T) {
	var b bool
	err := BoolScanner(&b).Scan(struct{}{})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}
