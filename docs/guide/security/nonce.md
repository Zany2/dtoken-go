# Nonce replay protection

English | [中文文档](nonce_zh.md)

Nonce is an independent capability composed through the nonce package Manager and Service. It requires no global registration or builder switches. The Manager handles credentials; the Service publishes events through an optional emitter.

## Basic use

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

The default TTL is five minutes. Values use 32 random bytes encoded as 64 hexadecimal characters. The example sets 30 seconds through the config constructor. Use service.GenerateNonceWithTimeout(ctx, timeout) for a single operation override.

## Validation and consumption

- GenerateNonce / GenerateNonceWithTimeout: generate and store a value.
- IsNonceValid: check existence without reserving it.
- VerifyNonce: validate and consume, returning bool.
- VerifyAndConsumeNonce: validate and consume with error details.
- GetNonceTTL: report remaining seconds.

Consumption explicitly requires contract.AtomicStorage. Built-in memory and Redis storage support it. Custom storage without atomic extensions returns ErrStorageCapabilityUnsupported; this is an explicit Nonce contract. Expired or consumed values cannot be consumed again.

## HTTP injection

Use framework WithCapabilities(requestcontext.Capabilities{Nonce: service}), with requestcontext importing httpauth/context. Request facades expose dCtx.Nonce().Generate, Verify, and VerifyAndConsume. Missing bindings return ErrModuleNotEnabled. See the [root README](../../../README.md) and [migration guide](../../design/module-migration.md).
