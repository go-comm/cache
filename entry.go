package cache

import (
	"github.com/go-comm/cache/internal/timingwheel"
	"unsafe"
)

type Entry struct {
	bk     uint8
	ek     uint16
	ctime  int64
	ex     int64
	p      unsafe.Pointer // *interface{}
	k      []byte
	future timingwheel.Future
}

func (e *Entry) Key() []byte {
	return e.k
}

func (e *Entry) StoreValue(i *interface{}) {
	e.p = unsafe.Pointer(i)
}

func (e *Entry) LoadValue() interface{} {
	return *(*interface{})(e.p)
}

func (e *Entry) Expired() bool {
	return e.ex+e.ctime < nowTime()
}
