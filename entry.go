package cache

import (
	"unsafe"
)

type entry struct {
	ek    uint16
	ctime int64
	ex    int64
	p     unsafe.Pointer // *interface{}
	k     []byte
}

func (e *entry) storeValue(i *interface{}) {
	e.p = unsafe.Pointer(i)
}

func (e *entry) loadValue() interface{} {
	return *(*interface{})(e.p)
}
