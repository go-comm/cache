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

type Cache interface {
	Get(ctx context.Context, k interface{}) (interface{}, error)
	GetAndTTL(ctx context.Context, k interface{}) (interface{}, int64, error)
	Scan(ctx context.Context, k interface{}, scan Scanner) error
	ScanAndTTL(ctx context.Context, k interface{}, scan Scanner) (int64, error)
	Put(ctx context.Context, k interface{}, v interface{}) error
	PutEx(ctx context.Context, k interface{}, v interface{}, sec int64) error
	Del(ctx context.Context, k interface{}) error
	TTL(ctx context.Context, k interface{}) (int64, error)
	Expire(ctx context.Context, k interface{}, sec int64) error
	Tx(ctx context.Context, k interface{}, fn func(*Entry) error) error
	ExpireHandler(h func(k interface{}, v interface{}))
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
