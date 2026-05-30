//go:build !go1.21

package cache

import "unsafe"

func BytesToStr(b []byte) string {
	bh := (*[3]uintptr)(unsafe.Pointer(&b))
	sh := [2]uintptr{bh[0], bh[1]}
	return *(*string)(unsafe.Pointer(&sh))
}

func StrToBytes(s string) []byte {
	sh := (*[2]uintptr)(unsafe.Pointer(&s))
	bh := [3]uintptr{sh[0], sh[1], sh[1]}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
