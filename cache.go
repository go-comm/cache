package cache

import (
	"errors"
)

var ErrNoKey = errors.New("cache: no key")

type Cache interface {
	Get(k []byte) (interface{}, error)
	GetAndTTL(k []byte) (interface{}, int64, error)
	Put(k []byte, v interface{}) error
	PutEx(k []byte, v interface{}, sec int64) error
	Del(k []byte) error
	TTL(k []byte) (int64, error)
	Expire(k []byte, sec int64) error
	Tx(k []byte, fn func(interface{}) (interface{}, error)) error
	ExpireHandler(h func(interface{}))
}
