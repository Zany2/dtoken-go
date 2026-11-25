English | [中文文档](../security/refresh-token_zh.md)

# Refresh Token Guide

Examples primarily use a root Auth instance; dt aliases the root package. Build auth with dt.NewBuilder().Build() and close it after use. Place operation fragments inside a function holding that instance and use the current request context as ctx.

## Overview

DToken-Go supports refresh tokens in two places:

- normal business login through `LoginWithRefreshToken(...)`
- OAuth2 token flow through `RefreshOAuth2AccessToken(...)`

This guide focuses on the normal business login flow.

## Login With Token Pair

```go
pair, err := auth.LoginWithRefreshTokenOptions(ctx, dt.RefreshTokenOptions{
    LoginOptions: dt.LoginOptions{LoginID: "user-1001", Device: "web", DeviceID: "browser-1"},
})
if err != nil {
	return err
}

fmt.Println(pair.AccessToken)
fmt.Println(pair.RefreshToken)
fmt.Println(pair.ExpiresIn)
fmt.Println(pair.RefreshExpiresIn)
```

`AccessToken` is used to access protected APIs. `RefreshToken` is stored by the client and used only to request a fresh token pair.

Login callbacks run before refresh issuance. Issuance rechecks the original login identity and active session under the account lock. If a callback logs out and reuses the access-token value for another login, the outer call fails instead of attaching a refresh token to that replacement. Failure cleanup is also restricted to the original lifecycle; it does not undo a later login.

## Login With Options

Use `LoginWithRefreshTokenOptions(...)` when a single login needs custom timeout, device, extra data, or concurrency behavior.

```go
pair, err := auth.LoginWithRefreshTokenOptions(ctx, dt.RefreshTokenOptions{
	LoginOptions: dt.LoginOptions{
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
nextPair, err := auth.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}
```

Refresh is a rotation operation:

1. validates the refresh token
2. rejects disabled accounts or devices; if the original access lifecycle has exceeded its inactivity timeout, retires it and revokes the refresh credential
3. consumes the old refresh token once and removes only its own reverse lookup
4. issues a fresh access token and refresh token, skipping duplicate concurrency eviction only when replacing the original attached terminal; otherwise applies the current account- or device-scoped login limit and non-concurrent policy
5. retires the old access token only if it still belongs to the original login lifecycle

If replacement issuance fails, the old refresh token has already been consumed, but the old access token is not proactively logged out.

The refresh token is independent from the old access token TTL. After natural access expiry, a valid refresh token can still rotate if the current disable and login policies permit it. If the original access record still exists and has exceeded its inactivity timeout, both direct refresh and authentication retire its credentials. Timeout events run after releasing the account lock.

Rotation preserves the login-time `Extra` (token data) and `TerminalExtra` (terminal data) separately, including when the old access token and Session have expired. Terminal data is stored as an optional field in the internal refresh record; existing public types and storage keys are unchanged. Older records without this field remain usable, but rotate without terminal extension data.

New logins have a random lifecycle identity in the internal access record, also saved with the refresh record. Renewal and shared login preserve it. Reusing the same access-token text, even within one second, creates a different identity: rotating or revoking an old refresh token must not remove the new login, its metadata, or its refresh binding. Cleanup rechecks this identity under the account lock; if the access record is already absent or inactive, it skips access-side cleanup. Expired terminal entries remain subject to normal session expiration or pruning.

A fresh login that reuses expired access-token text detaches the old reverse index while preserving the old lifecycle's independently valid refresh credential. Issuing or logging out the new login cannot revoke that credential through the old index; later rotation of the old credential still follows the current login policy.

Legacy pairs without lifecycle identities remain readable and refreshable. Access cleanup between two legacy records uses account, device, and creation-time checks on a best-effort basis; it cannot reliably distinguish token reuse that already happened before this upgrade. A legacy refresh record never matches a new-format access record. All writers sharing the storage should be upgraded: an older writer may drop the identity when rewriting token metadata. This does not introduce distributed transactions or require CAS support from custom storage.

## Revoke Flow

```go
err := auth.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

Revoking a refresh token also logs out its original access token when the lifecycle binding still matches. A later login that reuses the same token value is left untouched.

## TTL

```go
ttl, err := auth.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
```

The default refresh-token TTL is `30` days. Configure it globally:

```go
auth, err := dt.NewBuilder().
	RefreshTokenTimeout(30 * 24 * 60 * 60).
	Build()
```

Or use duration:

```go
auth, err := dt.NewBuilder().
	RefreshTokenTimeoutDuration(30 * 24 * time.Hour).
	Build()
```

## Global And Framework Access

Build registers automatically. Select the namespace through the root; auth below is the built instance:

```go
selected := dt.AuthType(auth.Config().AuthType)
pair, err := selected.LoginWithRefreshTokenOptions(ctx, dt.RefreshTokenOptions{LoginOptions: dt.LoginOptions{LoginID: "user-1001"}})
if err != nil {
	return err
}

nextPair, err := selected.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}

ttl, err := selected.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
if err != nil {
	return err
}
fmt.Println(ttl)

if err := selected.RevokeRefreshToken(ctx, nextPair.RefreshToken); err != nil {
	return err
}
```

In GoFrame, first bind Auth through WithAuth(auth), then use the request helpers from gfdt (integrations/gf). They use the request instance:

```go
func (c *AuthController) Login(r *ghttp.Request) {
	pair, err := gfdt.LoginWithRefreshTokenByCtx(r.Context(), "user-1001", "web", "browser-1")
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}

func (c *AuthController) Refresh(r *ghttp.Request) {
	pair, err := gfdt.RefreshTokenByCtx(r.Context(), r.Get("refreshToken").String())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": err.Error()})
	}
	r.Response.WriteJsonExit(pair)
}
```

## OAuth2 Refresh Token

OAuth2 refresh uses an independent service created with oauth2.NewService(server, auth). See the OAuth2 guide for initialization; this is a capability service method:

```go
newToken, err := service.RefreshOAuth2AccessToken(
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
