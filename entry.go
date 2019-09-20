package cache

import (
	"time"
	"unsafe"
)

type entry struct {
	value   unsafe.Pointer // *interface{}
	ek      uint16
	ctime   int64
	ex      int64
	k       []byte
	deleted bool
}

func (e *entry) storeValue(v *interface{}) {
	e.value = unsafe.Pointer(v)
}

func (e *entry) expired() bool {
	if e.ex < 0 {
		return false
	}
	return e.ctime+e.ex < time.Now().Unix()
}

func (e *entry) expire() int64 {
	if e.ex < 0 {
		return -1
	}
	sec := e.ctime + e.ex - time.Now().Unix()
	if sec < 0 {
		return 0
	}
	return sec
}

func (e *entry) loadValue() interface{} {
	return *(*interface{})(e.value)
}
