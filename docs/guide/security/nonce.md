English | [中文文档](../security/nonce_zh.md)

# Nonce Anti-Replay

## Overview

The project already includes built-in nonce support to prevent replayed requests.

Public APIs:

- `GenerateNonce`
- `GenerateNonceWithTimeout`
- `VerifyNonce`
- `VerifyAndConsumeNonce`
- `IsNonceValid`
- `GetNonceTTL`

## How It Works

The current implementation works like this:

1. generate a random nonce
2. store it with a TTL
3. verify it through an atomic `GetAndDelete`
4. allow it to succeed only once

Consumption requires `adapter.AtomicStorage`. Built-in memory and Redis storage support it. A custom storage must implement a truly atomic `GetAndDelete` across all callers sharing that storage. Storage without this interface returns `ErrStorageCapabilityUnsupported` on consumption. `IsNonceValid` is only a pre-check and does not reserve a nonce.

## Default Behavior

Based on the current `core/nonce` implementation:

- the raw nonce uses `32` random bytes
- the output is a `64`-character hexadecimal string
- the default TTL is `5` minutes
- the module must be enabled with `dtoken.NewBuilder().EnableNonce()`, or supplied through `SetNonceManager`

## Basic Usage

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

## Custom TTL

```go
ctx := context.Background()

nonce, err := dtoken.GenerateNonceWithTimeout(ctx, 30*time.Second)
_ = nonce
_ = err
```

If the timeout is less than or equal to `0`, the implementation falls back to the manager's configured TTL (`5` minutes by default).

## Non-Consuming Validation

```go
ctx := context.Background()

nonce, _ := dtoken.GenerateNonce(ctx)

valid := dtoken.IsNonceValid(ctx, nonce) // validate only
err := dtoken.VerifyAndConsumeNonce(ctx, nonce)
```

Difference:

- `IsNonceValid`: validate only
- `VerifyNonce`: validate and consume, returning `bool`
- `VerifyAndConsumeNonce`: validate and consume, returning `ErrInvalidNonce` for a missing, expired, or consumed nonce; storage failures return `ErrStorageUnavailable`, and missing atomic capability returns `ErrStorageCapabilityUnsupported`

## TTL Query

```go
ctx := context.Background()

ttl, err := dtoken.GetNonceTTL(ctx, nonce)
```

Return values:

- `-2`: nonce does not exist
- `-1`: no expiration
- `>=0`: remaining whole seconds, rounded down; `0` can mean a nonce is still valid for less than one second

## HTTP Example

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

## Best Practices

1. Use nonce for sensitive write operations such as payment, transfer, password change, and delete
2. Combine nonce checks with normal login checks instead of using nonce alone
3. A TTL around 5 minutes is usually enough for form submissions and one-time confirmations
4. If the client needs a pre-check, use `IsNonceValid` first and only consume on final submit

## Related Documentation

- [OAuth2 Guide](../security/oauth2.md)
- [Authentication Guide](../core/authentication.md)
- [Refresh Token Guide](../security/refresh-token.md)
