[English](../security/nonce.md) | 中文文档

# Nonce 防重放

## 概览

当前项目已经内置 `Nonce` 管理能力，用来防止请求重放。

公开入口包括：

- `GenerateNonce`
- `GenerateNonceWithTimeout`
- `VerifyNonce`
- `VerifyAndConsumeNonce`
- `IsNonceValid`
- `GetNonceTTL`

## 工作机制

当前实现里的 Nonce 规则很清晰：

1. 生成一个随机 nonce
2. 存入存储层并带 TTL
3. 校验时通过 `GetAndDelete` 原子消费
4. 同一个 nonce 只能成功一次

消费操作要求存储实现 `adapter.AtomicStorage`，内置内存和 Redis 存储均支持。自定义存储的 `GetAndDelete` 必须对共享该存储的所有调用者保持原子性；不支持原子能力时，消费返回 `ErrStorageCapabilityUnsupported`。`IsNonceValid` 仅用于预检查，不会预留 nonce。

## 默认行为

根据 `core/nonce` 当前实现：

- nonce 原始长度为 `32` 字节随机数
- 输出后是 `64` 位十六进制字符串
- 默认 TTL 为 `5` 分钟
- 必须通过 `dtoken.NewBuilder().EnableNonce()` 启用模块，或通过 `SetNonceManager` 注入

## 基本使用

```go
package main

import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go/com/storage/memory"
    "github.com/Zany2/dtoken-go/dtoken"
)

func initDToken() {
    if _, err := dtoken.BuildAndSetManager(
        dtoken.NewBuilder().
            EnableNonce().
            SetStorage(memory.NewStorage()),
    ); err != nil {
        panic(err)
    }
}

func main() {
    initDToken()
    defer dtoken.DeleteAllManager()

    ctx := context.Background()

    nonce, err := dtoken.GenerateNonce(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println(nonce)

    ok := dtoken.VerifyNonce(ctx, nonce)
    fmt.Println(ok) // true

    ok = dtoken.VerifyNonce(ctx, nonce)
    fmt.Println(ok) // false
}
```

## 自定义有效期

```go
ctx := context.Background()

nonce, err := dtoken.GenerateNonceWithTimeout(ctx, 30*time.Second)
_ = nonce
_ = err
```

如果传入的超时时间小于等于 `0`，底层会退回该管理器配置的 TTL（默认 `5` 分钟）。

## 非消费式校验

```go
ctx := context.Background()

nonce, _ := dtoken.GenerateNonce(ctx)

valid := dtoken.IsNonceValid(ctx, nonce) // 只校验，不消费
err := dtoken.VerifyAndConsumeNonce(ctx, nonce)
```

区别是：

- `IsNonceValid`：只检查
- `VerifyNonce`：检查并消费，返回 `bool`
- `VerifyAndConsumeNonce`：检查并消费；nonce 不存在、已过期或已消费时返回 `ErrInvalidNonce`，存储故障返回 `ErrStorageUnavailable`，缺少原子能力返回 `ErrStorageCapabilityUnsupported`

## 查看 TTL

```go
ctx := context.Background()

ttl, err := dtoken.GetNonceTTL(ctx, nonce)
```

返回值约定：

- `-2`：nonce 不存在
- `-1`：永久有效
- `>=0`：向下取整的剩余秒数；`0` 也可能表示 nonce 仍有效，但剩余时间不足一秒

## HTTP 场景示例

```go
r.GET("/nonce", func(c *gin.Context) {
    nonce, err := dtoken.GenerateNonce(ctx)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"nonce": nonce})
})

r.POST("/transfer", func(c *gin.Context) {
    nonce := c.GetHeader("X-Nonce")

    if err := dtoken.VerifyAndConsumeNonce(ctx, nonce); err != nil {
        c.JSON(401, gin.H{"error": "invalid_nonce"})
        return
    }

    c.JSON(200, gin.H{"message": "ok"})
})
```

## 最佳实践

1. 只给敏感写操作加 nonce，比如支付、转账、修改密码、删除操作
2. 建议和登录态一起使用，不要只校验 nonce 不校验用户身份
3. 对表单提交或一次性确认操作，5 分钟左右的 TTL 通常足够
4. 如果客户端需要先预校验，可先调用 `IsNonceValid`，真正提交时再消费

## 相关文档

- [OAuth2 指南](../security/oauth2_zh.md)
- [登录认证](../core/authentication_zh.md)
- [Refresh Token 指南](../security/refresh-token_zh.md)
