package cache

import "hash/fnv"

func NewMemery() Cache {
	m := &memery{}
	for i := 0; i < len(m.buckets); i++ {
		m.buckets[i] = newBucket()
	}
	return m
}

type memery struct {
	buckets  [256]*bucket
	finished bool
}

func (m *memery) loop() {
	var es []*Entry = make([]*Entry, 0, 32)
	for m.finished {
		for _, b := range m.buckets {
			es = es[0:]
			b.RLock()
			es = b.getExpiredEntries(es)
			b.RUnlock()

			b.RLock()
			for _, e := range es {
				b.del(e.k, e.ek)
			}
			b.RUnlock()
		}

	}
}

func (m *memery) Get(k []byte) (interface{}, error) {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.RLock()
	v := b.getVal(k, ek)
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

func (m *memery) Del(k []byte) error {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.Lock()
	b.del(k, ek)
	b.Unlock()
	return nil
}

func (m *memery) hashKey(k []byte) (uint8, uint16) {
	h := fnv.New32()
	h.Write(k)
	v := h.Sum32()
	return uint8(v >> 24), uint16(v)
}

func (m *memery) List(es []*Entry) []*Entry {
	for _, b := range m.buckets {
		b.RLock()
		es = b.list(es)
		b.RUnlock()
	}
	return es
}

func (m *memery) TTL(k []byte) int64 {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.RLock()
	ttl := b.ttl(k, ek)
	b.RUnlock()
	return ttl
}

func (m *memery) Expire(k []byte, ex int64) {
	bk, ek := m.hashKey(k)
	b := m.buckets[bk]
	b.RLock()
	b.expire(k, ek, ex)
	b.RUnlock()
}
