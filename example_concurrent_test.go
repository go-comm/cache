package cache

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
)

func Test_Concurrent_Memery(t *testing.T) {
	var wg sync.WaitGroup
	m := NewMemery()
	N := 10000
	goroutines := 200
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := N * base; j < N*(base+1); j++ {
				var key, value [16]byte
				binary.LittleEndian.PutUint64(key[:], uint64(j))
				binary.LittleEndian.PutUint64(key[8:], uint64(j))
				m.Put(context.TODO(), key[:], value)

				m.Get(context.TODO(), key[:])
			}
		}(i)
	}
	wg.Wait()
}

func Test_Concurrent_SyncMap(t *testing.T) {
	var wg sync.WaitGroup
	m := &sync.Map{}
	N := 10000
	goroutines := 200
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := N * base; j < N*(base+1); j++ {
				var key, value [16]byte
				binary.LittleEndian.PutUint64(key[:], uint64(j))
				binary.LittleEndian.PutUint64(key[8:], uint64(j))
				m.Store(key, value)

				m.Load(key)
			}
		}(i)
	}
	wg.Wait()
}
