package cache

import (
	"context"
	"errors"
)

func View(ctx context.Context, k []byte, c Cache, fn func() (interface{}, error)) (interface{}, error) {
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
	c.Put(ctx, k, v)
	return v, nil
}

func ViewEx(ctx context.Context, k []byte, ex int64, c Cache, fn func() (interface{}, error)) (interface{}, error) {
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
	c.Put(ctx, k, bv)
	return scan.Scan(bv)
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
	if fn == nil {
		return errors.New("function is nil")
	}
	return ViewScan(ctx, k, c, AnyScanner(dst), func() (Valuer, error) {
		v, err := fn()
		if err != nil {
			return nil, err
		}
		return AnyValuer(v), nil
	})
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
