package cache

import (
	"hash/fnv"
)

func NewMemery() Cache {
	m := &memery{}
	for i := 0; i < len(m.buckets); i++ {
		m.buckets[i] = newBucket()
	}
	return m
}

type memery struct {
	buckets [256]*bucket
}

func (m *memery) Get(k []byte) (interface{}, error) {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.RLock()
	v := b.get(k, ek)
	b.RUnlock()
	return v, nil
}

func (m *memery) Put(k []byte, v interface{}) error {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.Lock()
	b.put(k, ek, &v, 0, false)
	b.Unlock()
	return nil
}

func (m *memery) PutEx(k []byte, v interface{}, sec int64) error {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.Lock()
	b.put(k, ek, &v, sec, true)
	b.Unlock()
	return nil
}

func (m *memery) hashKey(k []byte) (uint8, uint16) {
	h := fnv.New32()
	h.Write(k)
	v := h.Sum32()
	return uint8(v >> 24), uint16(v & 0xFFFF)

	// return 1, uint16(v | 0xFFFF)
}
