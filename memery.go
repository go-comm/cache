package cache

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

func now() int64 {
	return time.Now().Unix()
}

/**
args: "{cap:1000}"
*/

func NewMemery(args ...interface{}) Cache {
	var config struct {
		Cap int `json:"cap"`
	}
	if len(args) > 0 {
		if vstr, ok := args[0].(string); ok {
			json.Unmarshal([]byte(vstr), &config)
		}
	}
	bcap := (config.Cap >> 8) * 4 / 3
	if bcap < 8 {
		bcap = 8
	}
	m := &memory{}
	for i := 0; i < len(m.buckets); i++ {
		m.buckets[i] = &bucket{m: m, bcap: bcap}
	}
	go m.expireInLoop()
	return m
}

type entry struct {
	createAt int64
	expireAt int64
	v        interface{}
}

func (e *entry) Expired() bool {
	return e.TTL() == 0
}

// TTL -1: never expired, 0: expired, >0: not expired
func (e *entry) TTL() int64 {
	if e.expireAt < 0 {
		return -1
	}
	ttl := e.expireAt - now()
	if ttl < 0 {
		ttl = 0
	}
	return ttl
}

type bucket struct {
	mutex sync.RWMutex
	_     [6]uint64
	m     *memory
	bcap  int
	store map[string]*entry
}

func (b *bucket) expire() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	var keys []string
	var vals []interface{}
	h := b.m.expireHandler

	b.mutex.RLock()
	for k, v := range b.store {
		if v != nil && v.Expired() {
			keys = append(keys, k)
			if h != nil {
				vals = append(vals, v.v)
			}
		}
	}
	b.mutex.RUnlock()

	if len(keys) > 0 {
		b.mutex.Lock()
		for _, k := range keys {
			if e, ok := b.store[k]; ok {
				if e != nil && e.Expired() {
					delete(b.store, k)
				}
			}
		}
		b.mutex.Unlock()
	}

	if h != nil && len(vals) > 0 {
		for i, v := range vals {
			h([]byte(keys[i]), v)
		}
	}
}

func (b *bucket) getStore() map[string]*entry {
	if b.store == nil {
		b.store = make(map[string]*entry, b.bcap)
	}
	return b.store
}

type memory struct {
	buckets       [256]*bucket
	cap           int
	expireHandler func([]byte, interface{})
}

func (m *memory) expireInLoop() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		<-ticker.C
		for i := len(m.buckets) - 1; i >= 0; i-- {
			b := m.buckets[i]
			b.expire()
		}
	}
}

func (m *memory) hashKey(k []byte) uint8 {
	hashed := 0
	for i := len(k) - 1; i >= 0; i-- {
		hashed = hashed*33 + int(k[i])
	}
	return uint8(hashed & 0xff)
}

func (m *memory) Get(ctx context.Context, k []byte) (interface{}, error) {
	b := m.buckets[m.hashKey(k)]
	key := string(k)
	b.mutex.RLock()
	e, ok := b.store[key]
	b.mutex.RUnlock()
	if !ok || e.Expired() {
		return nil, ErrNoKey
	}
	return e.v, nil
}

func (m *memory) GetAndTTL(ctx context.Context, k []byte) (interface{}, int64, error) {
	b := m.buckets[m.hashKey(k)]
	key := string(k)
	b.mutex.RLock()
	e, ok := b.store[key]
	b.mutex.RUnlock()
	if !ok {
		return nil, 0, ErrNoKey
	}
	ttl := e.TTL()
	if ttl == 0 {
		return nil, 0, ErrNoKey
	}
	return e.v, ttl, nil
}

func (m *memory) Put(ctx context.Context, k []byte, v interface{}) error {
	b := m.buckets[m.hashKey(k)]
	e := &entry{createAt: now(), expireAt: -1}
	var err error
	if event, ok := v.(*Event); ok {
		if e.v, err = event.Marshal(nil); err != nil {
			return err
		}
	} else {
		e.v = v
	}
	key := BytesToString(k)
	b.mutex.Lock()
	b.getStore()[key] = e
	b.mutex.Unlock()
	return nil
}

func (m *memory) PutEx(ctx context.Context, k []byte, v interface{}, sec int64) error {
	b := m.buckets[m.hashKey(k)]
	createAt := now()
	expireAt := createAt + sec
	if sec < 0 {
		expireAt = -1
	}
	e := &entry{createAt: now(), expireAt: expireAt}
	var err error
	if event, ok := v.(*Event); ok {
		if e.v, err = event.Marshal(nil); err != nil {
			return err
		}
	} else {
		e.v = v
	}
	key := BytesToString(k)
	b.mutex.Lock()
	b.getStore()[key] = e
	b.mutex.Unlock()
	return nil
}

func (m *memory) Del(ctx context.Context, k []byte) error {
	b := m.buckets[m.hashKey(k)]
	key := BytesToString(k)
	b.mutex.RLock()
	e, ok := b.store[key]
	b.mutex.RUnlock()
	if !ok || e.Expired() {
		return ErrNoKey
	}
	b.mutex.Lock()
	delete(b.getStore(), key)
	b.mutex.Unlock()
	h := m.expireHandler
	if h != nil {
		h(k, e.v)
	}
	return nil
}

func (m *memory) TTL(ctx context.Context, k []byte) (int64, error) {
	b := m.buckets[m.hashKey(k)]
	key := BytesToString(k)
	b.mutex.RLock()
	e, ok := b.store[key]
	b.mutex.RUnlock()
	if !ok {
		return 0, ErrNoKey
	}
	ttl := e.TTL()
	if ttl == 0 {
		return 0, ErrNoKey
	}
	return ttl, nil
}

func (m *memory) Expire(ctx context.Context, k []byte, sec int64) error {
	b := m.buckets[m.hashKey(k)]
	key := BytesToString(k)
	b.mutex.Lock()
	e, ok := b.store[key]
	if ok {
		if sec < 0 {
			e.expireAt = -1
		} else {
			e.expireAt = e.createAt + sec
		}
	}
	b.mutex.Unlock()
	if !ok {
		return ErrNoKey
	}
	return nil
}

func (m *memory) Tx(ctx context.Context, k []byte, fn func(interface{}) (interface{}, error)) error {
	b := m.buckets[m.hashKey(k)]
	key := BytesToString(k)
	b.mutex.Lock()
	defer b.mutex.Unlock()
	e, ok := b.store[key]
	if !ok {
		return ErrNoKey
	}
	o, err := fn(e.v)
	if err != nil {
		return err
	}
	e.v = o
	return nil
}

func (m *memory) ExpireHandler(h func([]byte, interface{})) {
	m.expireHandler = h
}
