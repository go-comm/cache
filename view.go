package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

func View(ctx context.Context, k interface{}, c Cache, fn func() (interface{}, error)) (interface{}, error) {
	return ViewEx(ctx, k, -1, c, fn)
}

func ViewEx(ctx context.Context, k interface{}, ex int64, c Cache, fn func() (interface{}, error)) (interface{}, error) {
	v, err := c.Get(ctx, k)
	if err == nil {
		return v, nil
	}
	if fn == nil {
		return nil, errors.New("function is nil")
	}
	v, err = fn()
	if err != nil {
		return nil, err
	}
	c.PutEx(ctx, k, v, ex)
	return v, nil
}

// ViewScan is a cache-aside pattern that uses Scan to assign the cached value
// directly into the target via a Scanner (e.g. sql.Scanner).
// On cache hit, the value is scanned into scan without returning the raw value.
// On cache miss, fn is called, the result (as Valuer) is stored in cache, then scanned.
//
// fn returns a Valuer so that Put can serialize the value for storage,
// and the resolved form is also passed to scan.Scan for correct deserialization.
//
// Usage:
//
//	var user User
//	ViewScan(ctx, "user:1", c, DecodeScanner(&user), func() (Valuer, error) {
//		u, err := db.GetUser(1)
//		return EncodeValuer(&u), err
//	})
func ViewScan(ctx context.Context, k interface{}, c Cache, scan Scanner, fn func() (Valuer, error)) error {
	return ViewScanEx(ctx, k, -1, c, scan, fn)
}

// ViewScanEx is like ViewScan but stores the value with a TTL (in seconds).
func ViewScanEx(ctx context.Context, k interface{}, ex int64, c Cache, scan Scanner, fn func() (Valuer, error)) error {
	err := c.Scan(ctx, k, scan)
	if err == nil {
		return nil
	}
	if fn == nil {
		return errors.New("function is nil")
	}
	v, err := fn()
	if err != nil {
		return err
	}
	bv, err := v.Value()
	if err != nil {
		return err
	}
	c.PutEx(ctx, k, bv, ex)
	return scan.Scan(bv)
}

// ViewScanAny is a simplified ViewScan that automatically uses AnyScanner and AnyValuer.
// dst is a pointer to the target variable (e.g. &user).
// fn returns the raw value, which is wrapped as AnyValuer for storage and scanned via AnyScanner.
//
// Usage:
//
//	var user User
//	ViewScanAny(ctx, "user:1", c, &user, func() (interface{}, error) {
//		return db.GetUser(1)
//	})
func ViewScanAny(ctx context.Context, k interface{}, c Cache, dst interface{}, fn func() (interface{}, error)) error {
	return ViewScanAnyEx(ctx, k, -1, c, dst, fn)
}

// ViewScanAnyEx is like ViewScanAny but stores the value with a TTL (in seconds).
func ViewScanAnyEx(ctx context.Context, k interface{}, ex int64, c Cache, dst interface{}, fn func() (interface{}, error)) error {
	if fn == nil {
		return errors.New("function is nil")
	}
	return ViewScanEx(ctx, k, ex, c, AnyScanner(dst), func() (Valuer, error) {
		v, err := fn()
		if err != nil {
			return nil, err
		}
		return AnyValuer(v), nil
	})
}

// SingleflightGroup abstracts the singleflight.Group interface to allow pluggable
// singleflight implementations (including mocks). The Do method executes a function
// for a given key and returns the result, ensuring that concurrent calls with the
// same key wait for the first call to complete and share its result.
// Typically you can pass &singleflight.Group{} as the implementation.
type SingleflightGroup interface {
	Do(key string, fn func() (interface{}, error)) (v interface{}, err error, shared bool)
}

// ViewWithSingleflightEx performs a cache-aside lookup with singleflight coalescing.
// It first attempts to get the key from the cache. On hit, it returns the value.
// On miss, it uses the provided SingleflightGroup g to ensure that only one call
// executes fn() for the given key concurrently; other callers wait for that single
// call to complete and share its result. The returned value is stored in the cache
// with the given TTL (ex seconds) before being returned.
//
// This is useful for preventing cache stampede when the cache is cold or has expired.
// The function signature is similar to ViewEx but adds a singleflight group parameter.
//
// Example:
//
//	var sfGroup singleflight.Group
//	val, err := ViewWithSingleflightEx(ctx, "key", 60, cache, &sfGroup, func() (interface{}, error) {
//	    return expensiveQuery(), nil
//	})
func ViewWithSingleflightEx(ctx context.Context, k interface{}, ex int64, c Cache, g SingleflightGroup, fn func() (interface{}, error)) (interface{}, error) {
	v, err := c.Get(ctx, k)
	if err == nil {
		return v, nil
	}
	if fn == nil {
		return nil, errors.New("function is nil")
	}
	ret, err, _ := g.Do(keyStr(k), func() (interface{}, error) {
		v, err := fn()
		if err != nil {
			return nil, err
		}
		_ = c.PutEx(ctx, k, v, ex)
		return v, nil
	})
	return ret, err
}

func keyStr(k interface{}) string {
	switch d := k.(type) {
	case string:
		return d
	case []byte:
		return string(d)
	case int:
		return strconv.Itoa(d)
	case int64:
		return strconv.FormatInt(d, 10)
	case uint64:
		return strconv.FormatUint(d, 10)
	default:
		var s string
		if ss, ok := k.(interface{ String() string }); ok {
			s = ss.String()
		} else {
			s = fmt.Sprintf("%v", d)
		}
		return s
	}
}
