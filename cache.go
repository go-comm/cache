package cache

import (
	"context"
	"errors"
)

var ErrNoKey = errors.New("cache: no key")

type Cache interface {
	Get(ctx context.Context, k []byte) (interface{}, error)
	GetAndTTL(ctx context.Context, k []byte) (interface{}, int64, error)
	Put(ctx context.Context, k []byte, v interface{}) error
	PutEx(ctx context.Context, k []byte, v interface{}, sec int64) error
	Del(ctx context.Context, k []byte) error
	TTL(ctx context.Context, k []byte) (int64, error)
	Expire(ctx context.Context, k []byte, sec int64) error
	Tx(ctx context.Context, k []byte, fn func(interface{}) (interface{}, error)) error
	ExpireHandler(h func(k []byte, v interface{}))
}
