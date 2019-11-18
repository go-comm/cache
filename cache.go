package cache

import (
	"hash/fnv"
	"time"

	"github.com/go-comm/cache/internal/timingwheel"
)

const (
	_ContextKeyCacheKey     = "_cache_key"
	_ContextKeyCacheHashKey = "_cache_hash_key"
)

type Cache interface {
	Get(k []byte) (interface{}, error)
	Put(k []byte, v interface{}) error
	PutEx(k []byte, v interface{}, sec int64) error
	Del(k []byte) error
	TTL(k []byte) int64
	Expire(k []byte, ex int64)
}

func New() Cache {
	m := &cache{}
	m.init()
	return m
}

type cache struct {
	buckets  [256]*bucket
	finished bool
	wheel    timingwheel.TimingWheel
}

func (m *cache) init() {
	m.wheel = timingwheel.New(time.Millisecond*50, 1<<10)

	for i := 0; i < len(m.buckets); i++ {
		b := newBucket()
		b.bk = uint8(i)
		b.wheel = m.wheel
		m.buckets[i] = b
	}
}

func (m *cache) Get(k []byte) (interface{}, error) {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	v := b.getVal(k, ek)
	return v, nil
}

func (m *cache) Put(k []byte, v interface{}) error {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.put(k, ek, &v, -1, false)
	return nil
}

func (m *cache) PutEx(k []byte, v interface{}, sec int64) error {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.put(k, ek, &v, sec, true)
	return nil
}

func (m *cache) Del(k []byte) error {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.del(k, ek)
	return nil
}

func (m *cache) hashKey(k []byte) (uint8, uint16) {
	h := fnv.New32()
	h.Write(k)
	v := h.Sum32()
	return uint8(v >> 24), uint16(v & 0xFFFF)
}

func (m *cache) TTL(k []byte) int64 {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	ttl := b.ttl(k, ek)
	return ttl
}

func (m *cache) Expire(k []byte, ex int64) {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.expire(k, ek, ex)
}
