package cache

import (
	"sync"
)

func newBucket() *bucket {
	b := &bucket{}
	b.data = &AVLTree{}
	return b
}

type bucket struct {
	sync.RWMutex

	_    [8]uint64
	size int
	bk   uint16
	data *AVLTree
}

func (b *bucket) getExpiredEntries(es []*Entry) []*Entry {
	now := nowTime()
	b.data.Iterator(func(v interface{}) bool {
		e := v.(*Entry)
		if now > e.ctime+e.ex {
			es = append(es, e)
		}
		return true
	})
	return es
}

func (b *bucket) getEntry(k []byte, ek uint16) *Entry {
	ei := b.data.Get(k, ek)
	if ei == nil {
		return nil
	}
	e, ok := ei.(*Entry)
	if !ok || e.Expired() {
		return nil
	}
	return e
}

func (b *bucket) getVal(k []byte, ek uint16) interface{} {
	e := b.getEntry(k, ek)
	if e == nil {
		return nil
	}
	return e.LoadValue()
}

func (b *bucket) ttl(k []byte, ek uint16) int64 {
	e := b.getEntry(k, ek)
	if e == nil {
		return 0
	}
	if e.ex < 0 {
		return -1
	}
	t := e.ctime + e.ex - nowTime()
	if t < 0 {
		t = 0
	}
	return t
}

func (b *bucket) expire(k []byte, ek uint16, ex int64) {
	e := b.getEntry(k, ek)
	if e == nil {
		return
	}
	e.ex = ex
}

func (b *bucket) put(k []byte, ek uint16, v *interface{}, ex int64, updateEx bool) {
	e := &Entry{}
	e.k = k
	e.ctime = nowTime()
	e.ek = ek
	e.ex = ex
	e.StoreValue(v)
	oe := b.data.Set(k, ek, e)
	if !updateEx && oe != nil {
		if oldEntry, ok := oe.(*Entry); ok {
			e.ex = oldEntry.ex
		}
	}
}

func (b *bucket) del(k []byte, ek uint16) {
	b.data.Del(k, ek)
}

func (b *bucket) list(es []*Entry) []*Entry {
	b.data.Iterator(func(v interface{}) bool {
		e := v.(*Entry)
		es = append(es, e)
		return true
	})
	return es
}
