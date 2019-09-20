package cache

import (
	"bytes"
	"sort"
	"sync"
	"time"
)

type entrys []*entry

func (e entrys) Len() int           { return len(e) }
func (e entrys) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e entrys) Less(i, j int) bool { return e[i].ek < e[j].ek }

func newBucket() *bucket {
	b := &bucket{}
	b.data = make(entrys, 0, 16)
	return b
}

type bucket struct {
	_ [7]uint64

	sync.RWMutex

	data entrys
}

func (b *bucket) get(k []byte, ek uint16) interface{} {
	i := b.searchIndex(k, ek)
	if i < 0 {
		return nil
	}
	e := b.data[i]
	if e.expired() {
		return nil
	}
	return e.loadValue()
}

func (b *bucket) put(k []byte, ek uint16, v *interface{}, ex int64, updateEx bool) {
	i, exists, extend := b.search(k, ek)
	var e *entry
	if exists {
		e = b.data[i]
	} else {
		if extend {
			es := make([]*entry, cap(b.data)*2)
			copy(es[:i], b.data[:i])
			if i < len(b.data) {
				copy(es[i+1:], b.data[i:])
			}
			b.data = es[:len(b.data)+1]
		} else {
			b.data = b.data[:i+1]
			if i < len(b.data) {
				copy(b.data[i+1:], b.data[i:])
			}
		}
		e = &entry{
			k:  k,
			ek: ek,
		}
		b.data[i] = e
	}

	if e != nil {
		e.ctime = time.Now().UnixNano()
		if updateEx {
			e.ex = ex
		}
		e.storeValue(v)
	}

}

func (b *bucket) searchIndex(k []byte, ek uint16) int {
	j := -1
	t := len(b.data)
	i := sort.Search(t, func(i int) bool {
		return b.data[i].ek >= ek
	})
	for ; i < t && b.data[i].ek == ek; i++ {
		if bytes.Equal(b.data[i].k, k) {
			j = i
			break
		}
	}
	return j
}

func (b *bucket) search(k []byte, ek uint16) (i int, exists, extend bool) {
	t := len(b.data)
	// i = sort.Search(len(b.data), func(i int) bool {
	// 	return b.data[i].ek >= ek
	// })
	i = b.lookup(ek)
	if i < t && b.data[i].ek == ek {
		for ; i < t && b.data[i].ek == ek; i++ {
			if bytes.Equal(b.data[i].k, k) {
				exists = true
				break
			}
		}
	}
	if !exists {
		extend = cap(b.data) <= len(b.data)
	}
	return
}

func (b *bucket) lookup(ek uint16) int {
	i, j, h := 0, len(b.data), 0
	for i < j {
		h = (i + j) >> 1
		t := b.data[h].ek
		if t < ek {
			i = h + 1
		} else if t == ek {
			i = h
			break
		} else {
			j = h
		}
	}
	return i
}
