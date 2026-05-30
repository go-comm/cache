# go-comm/cache

高性能 Go 本地缓存库，提供线程安全的内存缓存实现，支持 TTL 过期、分片并发、Cache-Aside 模式和灵活的序列化体系。

## 特性

- **256 分片锁** — 每个分片独立 `sync.RWMutex`，显著降低高并发场景下的锁竞争
- **惰性初始化** — 读操作零开销，写操作时才初始化分片和启动后台清理
- **双策略过期清理** — 随机抽样（平滑 CPU）+ 定期全量扫描（兜底），避免延迟毛刺
- **Cache-Aside 模式** — `View` / `ViewScan` 系列函数封装「查缓存 → 未命中则回调 → 回填」流程
- **Scan/Valuer 序列化体系** — 借鉴 `database/sql` 接口设计，解耦缓存存储与业务序列化
- **零拷贝优化** — string/[]byte 互转不分配内存，整型 key 避免 string 转换
- **Go 1.12+ 兼容** — 通过 build tag 适配不同 Go 版本的 unsafe API

## 安装

```bash
go get github.com/go-comm/cache
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-comm/cache"
)

func main() {
	c := cache.NewMemory()
	ctx := context.Background()

	// 存储
	c.Put(ctx, "name", "hello")

	// 读取
	v, _ := c.Get(ctx, "name")
	fmt.Println(v) // hello

	// 带过期时间
	c.PutEx(ctx, "token", "abc123", 60) // 60 秒后过期

	// 获取值和剩余 TTL
	val, ttl, _ := c.GetAndTTL(ctx, "token")
	fmt.Println(val, ttl) // abc123 60

	// 删除
	c.Del(ctx, "name")
}
```

## API 一览

### 创建缓存

```go
// 默认配置（每分片预分配 16 个 entry）
c := cache.NewMemory()

// 自定义分片容量
c := cache.NewMemory(`{"cap": 64}`)
```

### Cache 接口

```go
type Cache interface {
	Get(ctx context.Context, k interface{}) (interface{}, error)
	GetAndTTL(ctx context.Context, k interface{}) (interface{}, int64, error)
	Scan(ctx context.Context, k interface{}, scan Scanner) error
	ScanAndTTL(ctx context.Context, k interface{}, scan Scanner) (int64, error)
	Put(ctx context.Context, k interface{}, v interface{}) error
	PutEx(ctx context.Context, k interface{}, v interface{}, sec int64) error
	Del(ctx context.Context, k interface{}) error
	TTL(ctx context.Context, k interface{}) (int64, error)
	Expire(ctx context.Context, k interface{}, sec int64) error
	Tx(ctx context.Context, k interface{}, fn func(*Entry) error) error
	ExpireHandler(h func(k interface{}, v interface{}))
	Clear(ctx context.Context) error
}
```

| 方法 | 说明 |
|---|---|
| `Get` | 获取值，未找到或已过期返回 `ErrNoKey` |
| `GetAndTTL` | 获取值和剩余 TTL（秒），永不过期返回 `-1` |
| `Scan` | 获取值并通过 Scanner 反序列化到目标变量 |
| `ScanAndTTL` | Scan + 返回 TTL |
| `Put` | 存储值，永不过期 |
| `PutEx` | 存储值并设置 TTL（秒），`sec < 0` 表示永不过期 |
| `Del` | 删除键，触发 ExpireHandler 回调 |
| `TTL` | 查询剩余 TTL |
| `Expire` | 更新过期时间，`sec < 0` 设为永不过期 |
| `Tx` | 对单个 key 加写锁执行原子读-改-写 |
| `ExpireHandler` | 设置过期/删除时的异步回调 |
| `Clear` | 清空全部缓存 |

### 支持的 key 类型

`string`、`[]byte`、`int`、`int64`、`uint64`，以及实现了 `String() string` 接口的任意类型。其他类型通过 `fmt.Sprintf("%v", k)` 转为字符串。

## Scan/Valuer 序列化体系

缓存存取时可自定义序列化和反序列化逻辑，复用 `database/sql` 中 `driver.Valuer` 和 `sql.Scanner` 的设计思路。

### Valuer（写入时序列化）

```go
// 直接存储原始值（指针传递）
c.Put(ctx, "user", cache.AnyValuer(&user))

// JSON 编码后存储
c.Put(ctx, "user", cache.EncodeValuer(&user))

// 自定义编码器
c.Put(ctx, "user", cache.EncodeValuer{
	Marshal: msgpack.Marshal,
	Ptr:     &user,
})
```

### Scanner（读取时反序列化）

```go
// 直接赋值（反射）
c.Scan(ctx, "user", cache.AnyScanner(&user))

// JSON 解码
c.Scan(ctx, "user", cache.DecodeScanner(&user))

// 基础类型强类型扫描
var count int
c.Scan(ctx, "count", cache.IntScanner(&count))

var enabled bool
c.Scan(ctx, "flag", cache.BoolScanner(&enabled))

var name string
c.Scan(ctx, "name", cache.StringScanner(&name))
```

可用的 Scanner：

| Scanner | 目标类型 | 接受的输入类型 |
|---|---|---|
| `AnyScanner` | 任意 | 任意（反射赋值） |
| `DecodeScanner` | 任意 | `string` / `[]byte`（默认 JSON） |
| `IntScanner` | `*int` | `int` / `int64` / `uint64` / `float64` / `string` / `[]byte` |
| `Int64Scanner` | `*int64` | 同上 |
| `Uint64Scanner` | `*uint64` | 同上 |
| `BoolScanner` | `*bool` | `bool` / `int` / `string` / `[]byte` |
| `StringScanner` | `*string` | `string` / `[]byte` |

## Cache-Aside 模式

`view.go` 提供一组便捷函数，封装常见的「缓存未命中 → 调用函数 → 回填缓存」流程：

```go
// 基础用法
v, err := cache.View(ctx, []byte("user:1"), c, func() (interface{}, error) {
	return db.GetUser(1)
})

// 带 TTL
v, err := cache.ViewEx(ctx, []byte("user:1"), 300, c, func() (interface{}, error) {
	return db.GetUser(1)
})
```

### ViewScan — 扫描到目标变量

```go
var user User
err := cache.ViewScan(ctx, "user:1", c, cache.DecodeScanner(&user), func() (cache.Valuer, error) {
	u, err := db.GetUser(1)
	return cache.EncodeValuer(&u), err
})
```

### ViewScanAny — 最简写法

```go
var user User
err := cache.ViewScanAny(ctx, "user:1", c, &user, func() (interface{}, error) {
	return db.GetUser(1)
})

// 带 TTL
err = cache.ViewScanAnyEx(ctx, "user:1", 300, c, &user, func() (interface{}, error) {
	return db.GetUser(1)
})
```

## 事务

`Tx` 对单个 key 加写锁，回调中可读取和修改 entry（如调整 TTL），实现原子读-改-写：

```go
err := c.Tx(ctx, "counter", func(e *cache.Entry) error {
	val := e.Value().(int)
	// ... 修改逻辑
	return nil
})
```

## 过期回调

```go
c.ExpireHandler(func(k interface{}, v interface{}) {
	log.Printf("key %v expired, value: %v", k, v)
})
```

回调在过期清理和 `Del` 时异步触发，不阻塞缓存操作。

## 内部机制

### 分片与哈希

缓存内部维护 256 个 bucket，key 通过 DJB2 哈希的低 8 位确定分片。每个 bucket 独立加锁，并填充 cache-line padding 避免 false sharing。

### 过期清理策略

- **每 5 秒**：随机抽取 10 个 bucket 清理过期 entry
- **每 30 秒**：全量扫描所有 256 个 bucket

此策略兼顾及时性和 CPU 平滑性，避免大量 key 同时过期导致的延迟尖峰。

### 惰性初始化

调用 `Get` / `TTL` 等读操作时，若缓存未初始化直接返回 `ErrNoKey`，不触发任何分配。首次调用 `Put` / `PutEx` / `Del` 等写操作时才初始化所有 bucket 并启动后台清理协程。

## 错误处理

```go
import "github.com/go-comm/cache"

v, err := c.Get(ctx, "key")
if err == cache.ErrNoKey {
	// key 不存在或已过期
}
```

## 测试

```bash
go test -v ./...

# 启用竞态检测
go test -race -v ./...
```

## License

MIT
