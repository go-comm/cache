//go:build go1.21

package cache

import "unsafe"

func BytesToStr(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func StrToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
