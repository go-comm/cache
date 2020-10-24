package cache

import (
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

func NewMemery(args ...string) Cache {
	var config struct {
		Cap int `json:"cap"`
	}
	if len(args) > 0 {
		json.Unmarshal([]byte(args[0]), &config)
	}
	bcap := (config.Cap >> 8) * 4 / 3
	if bcap < 8 {
		bcap = 8
	}

	m := &memory{}
	for i := 0; i < len(m.buckets); i++ {
		m.buckets[i] = &bucket{store: make(map[string]*entry, bcap), m: m}
	}
	go m.expireInLoop()
	return m
}

type entry struct {
	ctime int64
	ex    int64
	v     interface{}
}

func (e *entry) Expired() bool {
	return e.TTL() == 0
}

// TTL -1: never expired, 0: expired, >0: not expired
func (e *entry) TTL() int64 {
	if e.ex < 0 {
		return -1
	}
	ttl := e.ex + e.ctime - now()
	if ttl < 0 {
		ttl = 0
	}
	return ttl
}

type bucket struct {
	mutex sync.RWMutex
	_     [6]uint64
	m     *memory
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
		for _, v := range vals {
			h(v)
		}
	}

}

type memory struct {
	buckets       [256]*bucket
	expireHandler func(interface{})
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

func (m *memory) Get(k []byte) (interface{}, error) {
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

func (m *memory) GetAndTTL(k []byte) (interface{}, int64, error) {
	b := m.buckets[m.hashKey(k)]
	key := string(k)
	b.mutex.RLock()
	e, ok := b.store[key]
	b.mutex.RUnlock()
	if !ok || e.Expired() {
		return nil, 0, ErrNoKey
	}
	return e.v, e.TTL(), nil
}

func (m *memory) Put(k []byte, v interface{}) error {
	b := m.buckets[m.hashKey(k)]
	e := &entry{ctime: now(), ex: -1, v: v}
	key := BytesToString(k)
	b.mutex.Lock()
	b.store[key] = e
	b.mutex.Unlock()
	return nil
}

func (m *memory) PutEx(k []byte, v interface{}, sec int64) error {
	b := m.buckets[m.hashKey(k)]
	e := &entry{ctime: now(), ex: sec, v: v}
	key := BytesToString(k)
	b.mutex.Lock()
	b.store[key] = e
	b.mutex.Unlock()
	return nil
}

func (m *memory) Del(k []byte) error {
	b := m.buckets[m.hashKey(k)]
	key := BytesToString(k)
	b.mutex.RLock()
	e, ok := b.store[key]
	b.mutex.RUnlock()
	if !ok || e.Expired() {
		return ErrNoKey
	}
	b.mutex.Lock()
	delete(b.store, key)
	b.mutex.Unlock()
	h := m.expireHandler
	if h != nil {
		h(e.v)
	}
	return nil
}

func (m *memory) TTL(k []byte) (int64, error) {
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

func (m *memory) Expire(k []byte, sec int64) error {
	b := m.buckets[m.hashKey(k)]
	key := BytesToString(k)
	b.mutex.Lock()
	e, ok := b.store[key]
	if ok {
		e.ex = sec
	}
	b.mutex.Unlock()
	if !ok {
		return ErrNoKey
	}
	return nil
}

func (m *memory) Tx(k []byte, fn func(interface{}) (interface{}, error)) error {
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

func (m *memory) ExpireHandler(h func(interface{})) {
	m.expireHandler = h
}
