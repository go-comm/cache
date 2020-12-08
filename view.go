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

func UnsafeViewEx(ctx context.Context, k []byte, v interface{}, ex int64, c Cache, fn func() (interface{}, error)) error {
	d, err := c.Get(ctx, k)
	if err == nil {
		UnsafeConvert(v, d)
		return nil
	}
	if fn == nil {
		return errors.New("function is nil")
	}
	d, err = fn()
	if err != nil {
		return err
	}
	UnsafeConvert(v, d)
	c.PutEx(ctx, k, d, ex)
	return nil
}
