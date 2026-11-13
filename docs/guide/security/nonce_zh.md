# Nonce 防重放

[English](nonce.md) | 中文文档

Nonce 是独立能力，使用 nonce 包的 Manager 和 Service 显式组合，无需全局注册或 Builder 开关。Manager 管理凭证，Service 在操作后通过可选发布器发送事件。

## 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/Zany2/dtoken-go"
    "github.com/Zany2/dtoken-go/nonce"
)

func main() {
    auth, err := dtoken.NewBuilder().IsPrintBanner(false).Build()
    if err != nil {
        panic(err)
    }
    defer auth.Close()
    cfg := auth.Config()
    manager := nonce.NewNonceManagerWithConfig(
        cfg.AuthType, cfg.KeyPrefix, auth.Storage(),
        &nonce.Config{TTL: 30 * time.Second},
    )
    service := nonce.NewService(manager, auth)
    ctx := context.Background()
    value, err := service.GenerateNonce(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println(service.IsNonceValid(ctx, value))
    fmt.Println(service.VerifyNonce(ctx, value))
    fmt.Println(service.VerifyNonce(ctx, value))
}
```

默认 TTL 为 5 分钟，随机源为 32 字节，输出 64 位十六进制字符串。上例以配置构造器改为 30 秒。也可调用 service.GenerateNonceWithTimeout(ctx, timeout) 为单次生成覆盖有效期。

## 校验与消费

- GenerateNonce / GenerateNonceWithTimeout：生成并保存。
- IsNonceValid：只检查存在，不预留凭证。
- VerifyNonce：校验并消费，返回 bool。
- VerifyAndConsumeNonce：校验并消费，保留错误详情。
- GetNonceTTL：返回剩余有效秒数。

消费契约要求 contract.AtomicStorage，内置内存和 Redis 存储均支持。不支持原子扩展的自定义存储会返回 ErrStorageCapabilityUnsupported；这是 Nonce 的显式契约。过期或已消费的值不能再次成功消费。

## HTTP 注入

通过框架 WithCapabilities(requestcontext.Capabilities{Nonce: service}) 注入请求上下文，requestcontext 是 httpauth/context 的别名。请求可调用 dCtx.Nonce().Generate、Verify 和 VerifyAndConsume。未绑定的能力返回 ErrModuleNotEnabled。完整接入见[根 README](../../../README_zh.md)和[迁移指南](../../design/module-migration_zh.md)。
