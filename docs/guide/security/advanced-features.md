# Advanced Features

This page lists advanced DToken-Go capabilities that can be used on top of the normal login, logout, permission, and role APIs.

## Token Introspection

Token introspection checks whether a token is currently active without renewing it. It returns ownership information, TTL, permissions, roles, token extra data, and an inactive reason.

```go
info, err := dtoken.IntrospectToken(ctx, token)
if err != nil {
	return err
}
if !info.Active {
	fmt.Println("invalid reason:", info.Error)
	return nil
}
fmt.Println(info.LoginID, info.ExpiresIn, info.Permissions, info.Roles)
```

Framework packages re-export the same API, so a GoFrame project can call `gfdt.IntrospectToken(...)` after importing `github.com/Zany2/dtoken-go/integrations/gf`.

## Refresh Token

The normal login flow supports an access-token + refresh-token pair. Refreshing rotates the refresh token: the old access token and old refresh token are revoked, then a fresh pair is issued.

```go
pair, err := dtoken.LoginWithRefreshToken(ctx, "user-1001")
if err != nil {
	return err
}

nextPair, err := dtoken.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}

ttl, err := dtoken.GetRefreshTokenTTL(ctx, nextPair.RefreshToken)
if err != nil {
	return err
}
fmt.Println(ttl)

_ = dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

The default refresh-token TTL is `30` days. You can override it globally with `RefreshTokenTimeout(...)` or per login with `LoginWithRefreshTokenOptions(...)`.

## Temporary Ticket

For one-time tickets, temporary authorization, and system-to-system ticket exchange.

```go
ticket, err := dtoken.CreateTicket(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ConsumeTicket(ctx, ticket)
if err != nil {
	return err
}
fmt.Println(token)
```

## Short-Key Access Credential

For short-link access, QR confirmation, temporary authorization, and system-to-system ticket exchange.

```go
shortKey, err := dtoken.CreateShortKey(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ConsumeShortKey(ctx, shortKey)
if err != nil {
	return err
}
fmt.Println(token)
```

## SSO

For unified login, ticket exchange, cross-system login-state sharing, and unified logout. SSO lives in the optional module `github.com/Zany2/dtoken-go/sso`; it is not coupled to the base auth architecture and only depends on storage and codec adapters. It provides primitives for Ticket, shared token, remote session, and OAuth2 authorization-code modes. Full HTTP SSO service support is still under development. See [SSO](../../../sso/README.md) for details.

```go
server := sso.NewServerWithConfig("sso:", "dtoken:", storage, codec, sso.DefaultConfig())

ticket, err := server.GenerateTicket(
	ctx,
	"app-a",
	"user-1001",
	"https://app.example.com/sso/callback",
	nil,
	nil,
)
if err != nil {
	return err
}

info, err := server.ConsumeTicket(
	ctx,
	ticket.Ticket,
	"app-a",
	"secret-a",
	"https://app.example.com/sso/callback",
)
if err != nil {
	return err
}
fmt.Println(info.LoginID)
```

## Nonce Anti-Replay

One-time random value generation, verification, and consumption to prevent replay attacks.

```go
nonce, err := dtoken.GenerateNonce(ctx)
if err != nil {
	return err
}

err = dtoken.VerifyAndConsumeNonce(ctx, nonce)
```

## Event Listener

The event system can listen to login, logout, renewal, permission, role, ban, unban, and other core lifecycle events.

```go
eventMgr := mgr.GetEventManager()
eventMgr.RegisterFunc(listener.EventAll, func(data *listener.EventData) {
	fmt.Println(data.Event, data.LoginID, data.Token)
})
```
