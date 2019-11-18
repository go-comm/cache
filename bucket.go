package cache

import (
	"context"
	"sync"
	"time"

	"github.com/go-comm/cache/internal/timingwheel"
)

func newBucket() *bucket {
	b := &bucket{}
	b.data = &AVLTree{}
	return b
}

type bucket struct {
	mutex sync.RWMutex
	_     [8]uint64
	wheel timingwheel.TimingWheel
	size  int
	bk    uint8
	data  *AVLTree
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
	b.mutex.RLock()
	ei := b.data.Get(k, ek)
	b.mutex.RUnlock()
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
	e.bk = b.bk
	e.ek = ek
	e.ex = ex
	e.StoreValue(v)

	b.mutex.Lock()
	ie := b.data.Set(k, ek, e)
	b.mutex.Unlock()

	if ie != nil {
		olde := ie.(*Entry)
		if olde.future != nil {
			olde.future.Cancel()
		}
		if !updateEx {
			e.ex = olde.ex
		}
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, _ContextKeyCacheKey, k)
	ctx = context.WithValue(ctx, _ContextKeyCacheHashKey, ek)
	e.future = b.wheel.PostDelayed(ctx, b.expireCallback, time.Duration(ex)*time.Millisecond)
}

func (b *bucket) del(k []byte, ek uint16) {
	b.mutex.Lock()
	b.data.Del(k, ek)
	b.mutex.Unlock()
}

func (b *bucket) expireCallback(ctx context.Context) error {
	ik := ctx.Value(_ContextKeyCacheKey)
	iek := ctx.Value(_ContextKeyCacheHashKey)
	if ik != nil && iek != nil {
		k := ik.([]byte)
		ek := iek.(uint16)
		b.del(k, ek)
	}
	return nil
}
