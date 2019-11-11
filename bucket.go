package cache

import (
	"sync"
	"time"
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

func (b *bucket) get(k []byte, ek uint16) interface{} {
	ei := b.data.Get(k, ek)
	if ei == nil {
		return nil
	}
	if e, ok := ei.(*entry); ok {
		return e.loadValue()
	}
	return nil
}

func (b *bucket) put(k []byte, ek uint16, v *interface{}, ex int64, updateEx bool) {
	e := &entry{}
	e.k = k
	e.ctime = time.Now().Unix()
	e.ek = ek
	e.ex = ex
	e.storeValue(v)
	oe := b.data.Set(k, ek, e)
	if !updateEx && oe != nil {
		if oldEntry, ok := oe.(*entry); ok {
			e.ex = oldEntry.ex
		}
	}
}
