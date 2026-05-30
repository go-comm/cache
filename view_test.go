package cache

import (
	"context"
	"errors"
	"testing"
)

func TestView(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	called := 0
	v, err := View(ctx, []byte("view_k"), c, func() (interface{}, error) {
		called++
		return "from_fn", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if v != "from_fn" {
		t.Fatalf("expected from_fn, got %v", v)
	}
	if called != 1 {
		t.Fatalf("expected 1 call, got %d", called)
	}

	// second call hits cache
	v, err = View(ctx, []byte("view_k"), c, func() (interface{}, error) {
		called++
		return "other", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if v != "from_fn" {
		t.Fatalf("expected from_fn (cached), got %v", v)
	}
	if called != 1 {
		t.Fatalf("expected fn not called again, got %d", called)
	}
}

func TestViewFnNil(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	_, err := View(ctx, []byte("vn"), c, nil)
	if err == nil {
		t.Fatal("expected error for nil fn")
	}
}

func TestViewFnError(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	_, err := View(ctx, []byte("ve"), c, func() (interface{}, error) {
		return nil, errors.New("fn error")
	})
	if err == nil || err.Error() != "fn error" {
		t.Fatalf("expected fn error, got %v", err)
	}
}

func TestViewEx(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	v, err := ViewEx(ctx, []byte("vex_k"), 5, c, func() (interface{}, error) {
		return 99, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if v != 99 {
		t.Fatalf("expected 99, got %v", v)
	}

	ttl, _ := c.TTL(ctx, "vex_k")
	if ttl <= 0 || ttl > 5 {
		t.Fatalf("expected ttl in (0,5], got %d", ttl)
	}
}

func TestViewExFnNil(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	_, err := ViewEx(ctx, []byte("vexn"), 5, c, nil)
	if err == nil {
		t.Fatal("expected error for nil fn")
	}
}

func TestViewExFnError(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	_, err := ViewEx(ctx, []byte("vexe"), 5, c, func() (interface{}, error) {
		return nil, errors.New("fn error")
	})
	if err == nil || err.Error() != "fn error" {
		t.Fatalf("expected fn error, got %v", err)
	}
}

func TestViewScan(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	called := 0
	var s string
	err := ViewScan(ctx, "vs_k", c, StringScanner(&s), func() (Valuer, error) {
		called++
		return AnyValuer("scanned"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if s != "scanned" {
		t.Fatalf("expected scanned, got %s", s)
	}
	if called != 1 {
		t.Fatalf("expected 1 call, got %d", called)
	}

	// second call hits cache
	err = ViewScan(ctx, "vs_k", c, StringScanner(&s), func() (Valuer, error) {
		called++
		return AnyValuer("other"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if s != "scanned" {
		t.Fatalf("expected scanned (cached), got %s", s)
	}
	if called != 1 {
		t.Fatalf("expected fn not called again, got %d", called)
	}
}

func TestViewScanFnNil(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var s string
	err := ViewScan(ctx, "vsn", c, StringScanner(&s), nil)
	if err == nil {
		t.Fatal("expected error for nil fn")
	}
}

func TestViewScanFnError(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var s string
	err := ViewScan(ctx, "vse", c, StringScanner(&s), func() (Valuer, error) {
		return nil, errors.New("fn error")
	})
	if err == nil || err.Error() != "fn error" {
		t.Fatalf("expected fn error, got %v", err)
	}
}

func TestViewScanWithEncodeValuer(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	type Info struct {
		Name string `json:"name"`
	}
	var info Info
	err := ViewScan(ctx, "vsd", c, DecodeScanner(&info), func() (Valuer, error) {
		return EncodeValuer(&Info{Name: "go"}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "go" {
		t.Fatalf("expected go, got %s", info.Name)
	}
}

func TestViewScanEx(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var n int64
	err := ViewScanEx(ctx, "vsx_k", 10, c, Int64Scanner(&n), func() (Valuer, error) {
		return AnyValuer(int64(123)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 123 {
		t.Fatalf("expected 123, got %d", n)
	}

	ttl, _ := c.TTL(ctx, "vsx_k")
	if ttl <= 0 || ttl > 10 {
		t.Fatalf("expected ttl in (0,10], got %d", ttl)
	}
}

func TestViewScanExFnNil(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var n int
	err := ViewScanEx(ctx, "vsxn", 10, c, IntScanner(&n), nil)
	if err == nil {
		t.Fatal("expected error for nil fn")
	}
}

func TestViewScanExFnError(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var n int
	err := ViewScanEx(ctx, "vsxe", 10, c, IntScanner(&n), func() (Valuer, error) {
		return nil, errors.New("fn error")
	})
	if err == nil || err.Error() != "fn error" {
		t.Fatalf("expected fn error, got %v", err)
	}
}

// ==================== ViewScanAny / ViewScanAnyEx ====================

func TestViewScanAny(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	type User struct {
		Name string
		Age  int
	}

	called := 0
	var user User
	err := ViewScanAny(ctx, "vg_k", c, &user, func() (interface{}, error) {
		called++
		return &User{Name: "test", Age: 25}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "test" || user.Age != 25 {
		t.Fatalf("expected {test 25}, got %+v", user)
	}
	if called != 1 {
		t.Fatalf("expected 1 call, got %d", called)
	}

	// second call hits cache
	var user2 User
	err = ViewScanAny(ctx, "vg_k", c, &user2, func() (interface{}, error) {
		called++
		return &User{Name: "other", Age: 30}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if user2.Name != "test" || user2.Age != 25 {
		t.Fatalf("expected cached {test 25}, got %+v", user2)
	}
	if called != 1 {
		t.Fatalf("expected fn not called again, got %d", called)
	}
}

func TestViewScanAnyFnNil(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var s string
	err := ViewScanAny(ctx, "vgn", c, &s, nil)
	if err == nil {
		t.Fatal("expected error for nil fn")
	}
}

func TestViewScanAnyFnError(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var s string
	err := ViewScanAny(ctx, "vge", c, &s, func() (interface{}, error) {
		return nil, errors.New("fn error")
	})
	if err == nil || err.Error() != "fn error" {
		t.Fatalf("expected fn error, got %v", err)
	}
}

func TestViewScanAnyEx(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var n int
	err := ViewScanAnyEx(ctx, "vgx_k", 10, c, &n, func() (interface{}, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}

	ttl, _ := c.TTL(ctx, "vgx_k")
	if ttl <= 0 || ttl > 10 {
		t.Fatalf("expected ttl in (0,10], got %d", ttl)
	}
}

func TestViewScanAnyExFnNil(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var n int
	err := ViewScanAnyEx(ctx, "vgxn", 10, c, &n, nil)
	if err == nil {
		t.Fatal("expected error for nil fn")
	}
}

func TestViewScanAnyExFnError(t *testing.T) {
	c := newCache()
	ctx := context.Background()

	var n int
	err := ViewScanAnyEx(ctx, "vgxe", 10, c, &n, func() (interface{}, error) {
		return nil, errors.New("fn error")
	})
	if err == nil || err.Error() != "fn error" {
		t.Fatalf("expected fn error, got %v", err)
	}
}

