package cache

import (
	"encoding/binary"
	"sync"
	"testing"
)

func Benchmark_MemeryPut(b *testing.B) {
	b.StopTimer()
	m := New()
	var key [8]byte

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m.Put(key[:], make([]byte, 8))
	}
}

func Benchmark_MapSet(b *testing.B) {
	b.StopTimer()
	m := make(map[string][]byte)
	var key [8]byte

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m[string(key[:])] = make([]byte, 8)
	}
}

func Benchmark_SyncMapStore(b *testing.B) {
	b.StopTimer()
	m := &sync.Map{}
	var key [8]byte

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m.Store(string(key[:]), make([]byte, 8))
	}
}
