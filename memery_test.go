package cache

import (
	"encoding/binary"
	"sync"
	"testing"
)

func Test_Memery(t *testing.T) {

	m := NewMemery()

	m.Put([]byte("user"), "admin")
	t.Log(m.Get([]byte("user")))

	m.Put([]byte("user"), "guest")
	t.Log(m.Get([]byte("user")))

	t.Log(m.Get([]byte("user1")))
}

func Benchmark_MemeryPut(b *testing.B) {
	m := NewMemery()
	var key [8]byte
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m.Put(key[:], make([]byte, 8))
	}
}

func Benchmark_MapSet(b *testing.B) {
	m := make(map[string][]byte)
	var key [8]byte
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m[string(key[:])] = make([]byte, 8)
	}
}

func Benchmark_SyncMapStore(b *testing.B) {
	m := &sync.Map{}
	var key [8]byte
	for i := 0; i < b.N; i++ {
		binary.LittleEndian.PutUint64(key[:], uint64(i))
		m.Store(string(key[:]), make([]byte, 8))
	}
}

func Test_Concurrent_Memery(t *testing.T) {
	var wg sync.WaitGroup
	m := NewMemery()
	N := 5000
	goroutines := 60
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := N * base; j < N*(base+1); j++ {
				var key, value [8]byte
				binary.LittleEndian.PutUint64(key[:], uint64(j))
				m.Put(key[:], value)
			}
		}(i)
	}
	wg.Wait()

}

func Test_Concurrent_SyncMap(t *testing.T) {
	var wg sync.WaitGroup
	m := &sync.Map{}
	N := 5000
	goroutines := 60
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := N * base; j < N*(base+1); j++ {
				var key, value [8]byte
				binary.LittleEndian.PutUint64(key[:], uint64(j))
				m.Store(key, value)
			}
		}(i)
	}
	wg.Wait()
}
