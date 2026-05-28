# Advanced Features

This page lists advanced DToken-Go capabilities. Some features are still under development and are marked with `🚧` in the README feature table.

## Token Introspection 🚧

Standardized query for Token validity, ownership information, TTL, and invalid reason.

```go
info, err := dtoken.IntrospectToken(ctx, token)
if err != nil {
	return err
}
if !info.Active {
	fmt.Println("invalid reason:", info.Reason)
}
```

## Refresh Token 🚧

Independent refresh token issuing, refreshing, revocation, expiration, and rotation.

```go
pair, err := dtoken.LoginWithRefreshToken(ctx, "user-1001")
if err != nil {
	return err
}

nextPair, err := dtoken.RefreshToken(ctx, pair.RefreshToken)
if err != nil {
	return err
}
_ = dtoken.RevokeRefreshToken(ctx, nextPair.RefreshToken)
```

## Temporary Ticket 🚧

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

## Short-Key Access Credential 🚧

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

For unified login, ticket exchange, cross-system login-state sharing, and unified logout. Ticket mode is available now; see [SSO](../security/sso.md) for details.

```go
ticket, err := dtoken.GenerateSSOTicket(
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

info, err := dtoken.ConsumeSSOTicket(
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
