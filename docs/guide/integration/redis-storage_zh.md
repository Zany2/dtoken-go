# Redis 存储指南

[English](../integration/redis-storage.md) | 中文文档

## 概览

当前 Redis 存储实现位于：

- `com/storage/redis`

公开构造方式只有 3 种：

1. `redis.NewStorage(url string)`
2. `redis.NewStorageFromConfig(cfg *redis.Config)`
3. `redis.NewStorageFromClient(client *redis.Client)`

## 安装

```bash
go get github.com/Zany2/dtoken-go/com/storage/redis
go get github.com/redis/go-redis/v9
```

## 使用方式

### 方式一：Redis URL

```go
package main

import (
    "github.com/Zany2/dtoken-go/com/storage/redis"
    "github.com/Zany2/dtoken-go/defaults"
    "github.com/Zany2/dtoken-go/dtoken"
)

func initDToken() {
    storage, err := redis.NewStorage("redis://localhost:6379/0")
    if err != nil {
        panic(err)
    }

    dtoken.SetManager(
        defaults.NewBuilder().
            SetStorage(storage).
            Build(),
    )
}
```

### 方式二：结构化配置

```go
storage, err := redis.NewStorageFromConfig(&redis.Config{
    Host:         "127.0.0.1",
    Port:         6379,
    Password:     "",
    Database:     0,
    PoolSize:     20,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    PoolTimeout:  4 * time.Second,
})
if err != nil {
    panic(err)
}

dtoken.SetManager(
    defaults.NewBuilder().
        SetStorage(storage).
        Build(),
)
```

### 方式三：复用现有 go-redis Client

```go
rdb := goredis.NewClient(&goredis.Options{
    Addr:     "127.0.0.1:6379",
    Password: "",
    DB:       0,
})

storage := redis.NewStorageFromClient(rdb)

dtoken.SetManager(
    defaults.NewBuilder().
        SetStorage(storage).
        Build(),
)
```

## Config 字段

当前 `redis.Config` 支持：

| 字段 | 说明 |
|------|------|
| `Host` | 主机 |
| `Port` | 端口 |
| `Password` | 密码 |
| `Database` | 库索引 |
| `PoolSize` | 连接池大小 |
| `DialTimeout` | 建连超时 |
| `ReadTimeout` | 读超时 |
| `WriteTimeout` | 写超时 |
| `PoolTimeout` | 取连接超时 |
| `OperationTimeout` | 预留字段，当前存储实现未在各操作中单独套用 |

## 和 DToken 搭配使用

```go
storage, _ := redis.NewStorage("redis://localhost:6379/0")

dtoken.SetManager(
    defaults.NewBuilder().
        SetStorage(storage).
        TokenName("dtoken").
        Timeout(2 * 60 * 60).
        ActiveTimeout(30 * 60).
        AutoRenew(true).
        Build(),
)
```

这时登录态、Session、权限、角色、Nonce、OAuth2 Token 等数据都会走 Redis。

## Redis Key 结构

核心存储 key 使用统一格式：

```text
KeyPrefix + AuthType + 业务前缀 + 业务标识
```

默认配置下：

```text
KeyPrefix = dtoken:
AuthType  = auth:
```

常见 key 示例：

```text
dtoken:auth:token:{token}                         -> Token 信息
dtoken:auth:session:{loginID}                     -> Session 信息
dtoken:auth:renew:{token}                         -> 续期间隔标记
dtoken:auth:active:{token}                        -> 活跃超时标记
dtoken:auth:disable:{loginID}                     -> 账号封禁信息
dtoken:auth:disable:service:{loginID}:{service}   -> 服务封禁信息
dtoken:auth:disable:device:{loginID}:{device}     -> 设备类型封禁信息
dtoken:auth:oauth2:client:{clientID}              -> OAuth2 客户端
```

### 多认证体系

多认证体系通过 `AuthType` 隔离。同一个 Redis DB 中可以同时存在：

```text
dtoken:user-auth:token:{token}
dtoken:user-auth:session:{loginID}

dtoken:admin-auth:token:{token}
dtoken:admin-auth:session:{loginID}
```

同一个 `loginID` 在不同 `AuthType` 下互不影响，Token、Session、权限、角色、封禁和 OAuth2 数据都会按认证体系隔离。

### 测试前缀

`tests/gin_core_flow` 为了避免污染 Redis，会使用短前缀：

```text
dt:gcf:1:
dt:gcf:2:
```

测试清理时只删除当前前缀下的 key，例如：

```text
dt:gcf:1:*
```

不会执行全库清空。

## 当前存储能力

当前 Redis 适配器实现了这些基础操作：

- `Set`
- `Get`
- `GetAndDelete`
- `Delete`
- `Exists`
- `Keys`
- `Expire`
- `TTL`
- `Clear`
- `Ping`
- `Close`
- `GetClient`

## 注意事项

### 不存在 Redis Builder

当前包里**没有** `redis.NewBuilder()` 这一套 API，旧文档里这部分已经过时。

### NewStorageFromClient 只接收 *redis.Client

当前 `NewStorageFromClient` 的签名是：

```go
func NewStorageFromClient(client *redis.Client) *Storage
```

这意味着：

1. 直接传入标准单机 `*redis.Client` 没问题
2. Redis Cluster / Sentinel 目前没有现成的同名适配入口
3. 如果你要接其他客户端形态，需要自己补一层 `adapter.Storage`

### 不存在 Key 不算错误

当前实现里：

- `Get()` 取不到 key 时返回 `nil, nil`
- `GetAndDelete()` 取不到 key 时也返回 `nil, nil`
- `Expire()` 如果 key 不存在，则返回 `ErrKeyNotFound`

## 简单排查

```go
ctx := context.Background()

if err := storage.Ping(ctx); err != nil {
    panic(err)
}

client := storage.GetClient()
_ = client
```

## 最佳实践

1. 开发环境可以先用内存存储，联调或生产再切 Redis
2. 生产环境建议显式配置连接池大小和超时
3. JWT 风格并不会绕开存储层，配 Redis 仍然有意义
4. 用完自定义 client 后记得在应用退出时关闭连接

## 相关文档

- [登录认证](../core/authentication_zh.md)
- [JWT 指南](../security/jwt_zh.md)
- [OAuth2 指南](../security/oauth2_zh.md)
- [核心流程测试指南](../integration/core-flow-testing_zh.md)
