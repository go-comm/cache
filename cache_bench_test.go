package cache

import (
	"context"
	"sync"
	"testing"
)

// ================== Unified Interface ==================

// CacheBackend is the common interface for all three cache implementations,
// ensuring a fair benchmark comparison.
type CacheBackend interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
}

// ================== Backend 1: single sync.RWMutex + map ==================

type SingleMutexCache struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func NewSingleMutexCache() *SingleMutexCache {
	return &SingleMutexCache{data: make(map[string]interface{})}
}

func (c *SingleMutexCache) Set(key string, value interface{}) {
	c.mu.Lock()
	c.data[key] = value
	c.mu.Unlock()
}

func (c *SingleMutexCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	v, ok := c.data[key]
	c.mu.RUnlock()
	return v, ok
}

// ================== Backend 2: sync.Map ==================

type SyncMapCache struct {
	m sync.Map
}

func NewSyncMapCache() *SyncMapCache {
	return &SyncMapCache{}
}

func (c *SyncMapCache) Set(key string, value interface{}) {
	c.m.Store(key, value)
}

func (c *SyncMapCache) Get(key string) (interface{}, bool) {
	return c.m.Load(key)
}

// ================== Backend 3: Memory (256-way sharded RWMutex) ==================

type MemoryCache struct {
	c   Cache
	ctx context.Context
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		c:   NewMemory(),
		ctx: context.Background(),
	}
}

func (c *MemoryCache) Set(key string, value interface{}) {
	c.c.Put(c.ctx, key, value)
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	v, err := c.c.Get(c.ctx, key)
	return v, err == nil
}

// ================== Benchmark Scenarios ==================

const numKeys = 50000 // number of keys to pre-populate

var keys []string

func init() {
	keys = make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = string(rune('a'+i%26)) + string(rune('A'+i/26%26)) + string(rune('0'+i%10))
	}
}

// prePopulate fills the cache with initial data so reads always hit.
func prePopulate(b CacheBackend) {
	for i := 0; i < numKeys; i++ {
		b.Set(keys[i], i)
	}
}

// ================== 90% Read / 10% Write ==================

func benchmarkReadHeavy(b *testing.B, backend CacheBackend) {
	prePopulate(backend)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			idx := i % numKeys
			if i%10 == 0 {
				backend.Set(keys[idx], i)
			} else {
				backend.Get(keys[idx])
			}
			i++
		}
	})
}

func BenchmarkSingleMutex_ReadHeavy(b *testing.B) { benchmarkReadHeavy(b, NewSingleMutexCache()) }
func BenchmarkSyncMap_ReadHeavy(b *testing.B)     { benchmarkReadHeavy(b, NewSyncMapCache()) }
func BenchmarkMemory_ReadHeavy(b *testing.B)      { benchmarkReadHeavy(b, NewMemoryCache()) }

// ================== 50% Read / 50% Write ==================

func benchmarkReadWrite(b *testing.B, backend CacheBackend) {
	prePopulate(backend)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			idx := i % numKeys
			if i%2 == 0 {
				backend.Set(keys[idx], i)
			} else {
				backend.Get(keys[idx])
			}
			i++
		}
	})
}

func BenchmarkSingleMutex_ReadWrite(b *testing.B) { benchmarkReadWrite(b, NewSingleMutexCache()) }
func BenchmarkSyncMap_ReadWrite(b *testing.B)     { benchmarkReadWrite(b, NewSyncMapCache()) }
func BenchmarkMemory_ReadWrite(b *testing.B)      { benchmarkReadWrite(b, NewMemoryCache()) }

// ================== 90% Write / 10% Read ==================

func benchmarkWriteHeavy(b *testing.B, backend CacheBackend) {
	prePopulate(backend)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			idx := i % numKeys
			if i%10 == 0 {
				backend.Get(keys[idx])
			} else {
				backend.Set(keys[idx], i)
			}
			i++
		}
	})
}

func BenchmarkSingleMutex_WriteHeavy(b *testing.B) { benchmarkWriteHeavy(b, NewSingleMutexCache()) }
func BenchmarkSyncMap_WriteHeavy(b *testing.B)     { benchmarkWriteHeavy(b, NewSyncMapCache()) }
func BenchmarkMemory_WriteHeavy(b *testing.B)     { benchmarkWriteHeavy(b, NewMemoryCache()) }

// ================== 100% Read ==================

func benchmarkReadOnly(b *testing.B, backend CacheBackend) {
	prePopulate(backend)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			backend.Get(keys[i%numKeys])
			i++
		}
	})
}

func BenchmarkSingleMutex_ReadOnly(b *testing.B) { benchmarkReadOnly(b, NewSingleMutexCache()) }
func BenchmarkSyncMap_ReadOnly(b *testing.B)     { benchmarkReadOnly(b, NewSyncMapCache()) }
func BenchmarkMemory_ReadOnly(b *testing.B)       { benchmarkReadOnly(b, NewMemoryCache()) }

// ================== 100% Write ==================

func benchmarkWriteOnly(b *testing.B, backend CacheBackend) {
	prePopulate(backend)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			backend.Set(keys[i%numKeys], i)
			i++
		}
	})
}

func BenchmarkSingleMutex_WriteOnly(b *testing.B) { benchmarkWriteOnly(b, NewSingleMutexCache()) }
func BenchmarkSyncMap_WriteOnly(b *testing.B)     { benchmarkWriteOnly(b, NewSyncMapCache()) }
func BenchmarkMemory_WriteOnly(b *testing.B)      { benchmarkWriteOnly(b, NewMemoryCache()) }
