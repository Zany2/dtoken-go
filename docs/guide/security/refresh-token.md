English | [中文文档](../security/refresh-token_zh.md)

# Refresh Token Guide

## Overview

DToken-Go supports refresh tokens in two places:

- normal business login through `LoginWithRefreshToken(...)`
- OAuth2 token flow through `RefreshOAuth2AccessToken(...)`

This guide focuses on the normal business login flow.

## Login With Token Pair

```go
pair, err := dtoken.LoginWithRefreshToken(ctx, "user-1001", "web", "browser-1")
if err != nil {
	return err
}

fmt.Println(pair.AccessToken)
fmt.Println(pair.RefreshToken)
fmt.Println(pair.ExpiresIn)
fmt.Println(pair.RefreshExpiresIn)
```

`AccessToken` is used to access protected APIs. `RefreshToken` is stored by the client and used only to request a fresh token pair.

## Login With Options

Use `LoginWithRefreshTokenOptions(...)` when a single login needs custom timeout, device, extra data, or concurrency behavior.

```go
pair, err := dtoken.LoginWithRefreshTokenOptions(ctx, dtoken.RefreshTokenOptions{
	LoginOptions: dtoken.LoginOptions{
		LoginID: "user-1001",
		Device:  "app",
		Extra: map[string]any{
			"tenant": "main",
		},
	},
	RefreshTimeout: 30 * 24 * time.Hour,
})
```

## Refresh Flow

```go
nextPair, err := dtoken.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}
```

Refresh is a rotation operation:

1. validates the refresh token
2. rejects disabled accounts or disabled devices
3. revokes the old access token and old refresh token
4. issues a fresh access token and refresh token

The refresh token is independent from the old access token TTL. If the access token has expired but the refresh token is still valid, refresh can still succeed.

## Revoke Flow

```go
err := dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

Revoking a refresh token also logs out the related access token.

## TTL

```go
ttl, err := dtoken.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
```

The default refresh-token TTL is `30` days. Configure it globally:

```go
mgr, err := dtoken.NewBuilder().
	RefreshTokenTimeout(30 * 24 * 60 * 60).
	Build()
```

Or use duration:

```go
mgr, err := dtoken.NewBuilder().
	RefreshTokenTimeoutDuration(30 * 24 * time.Hour).
	Build()
```

## Framework Facade Example

Framework packages re-export these APIs. For GoFrame:

```go
import gfdt "github.com/Zany2/dtoken-go/integrations/gf"

pair, err := gfdt.LoginWithRefreshToken(ctx, "user-1001")
nextPair, err := gfdt.RefreshToken(ctx, pair.RefreshToken)
ttl, err := gfdt.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
_ = gfdt.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

In a GoFrame controller, the login and refresh handlers can keep using the same framework package:

```go
func (c *AuthController) Login(r *ghttp.Request) {
	pair, err := gfdt.LoginWithRefreshToken(r.Context(), "user-1001", "web", "browser-1")
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}

func (c *AuthController) Refresh(r *ghttp.Request) {
	pair, err := gfdt.RefreshToken(r.Context(), r.Get("refreshToken").String())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}
```

## OAuth2 Refresh Token

OAuth2 refresh token support remains available through the OAuth2 APIs:

```go
newToken, err := dtoken.RefreshOAuth2AccessToken(
	ctx,
	"web-app",
	oldToken.RefreshToken,
	"secret",
)
```

See [OAuth2 Guide](../security/oauth2.md) for OAuth2-specific behavior.

## Related Documentation

- [OAuth2 Guide](../security/oauth2.md)
- [Authentication Guide](../core/authentication.md)
- [Advanced Features](../security/advanced-features.md)
