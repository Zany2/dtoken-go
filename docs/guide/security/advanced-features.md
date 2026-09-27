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

Rotation consumes the old refresh token once, issues a replacement pair, then retires the old access token. Failed issuance does not restore the consumed refresh token; the old access token remains until its original expiry unless another operation terminates it.

Natural access-token expiry preserves an independently valid refresh credential. Rotation prunes expired session terminals while preserving other terminals and account data. Explicit logout, kickout, or replacement cleans up linked refresh credentials.

New reverse indexes from access tokens to refresh tokens use the refresh lifetime so access-token renewal does not lose revocation links. Existing shorter-lived indexes are not migrated automatically; newly issued or rotated pairs use the new rule.

Consumption prefers `AtomicStorage.GetAndDelete`. Basic storage is serialized by the account lock within one Manager, without atomic consumption guarantees across instances or processes.

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

Ticket validation and state updates use the precise `ExpiresAt` deadline. `CreateTime` remains a Unix timestamp in seconds and `ExpiresIn` remains the TTL rounded up to seconds. Stored tickets without `ExpiresAt` retain their previous whole-second expiration behavior.

Consumption prefers `adapter.AtomicStorage`. With plain `Storage`, consumption and revocation are serialized within one Ticket manager instance; this fallback does not provide atomic consumption across manager instances or processes. Use atomic storage when multiple instances share tickets.

```go
createdTicket, err := dtoken.CreateTicket(ctx, "user-1001")
if err != nil {
	return err
}

result, err := dtoken.ConsumeTicket(ctx, createdTicket.Ticket)
if err != nil {
	return err
}
fmt.Println(result.Ticket.LoginID)
```

## Short-Key Access Credential

For short-link access, QR confirmation, temporary authorization, and system-to-system ticket exchange.

Short keys use the precise `ExpiresAt` deadline for validation and state updates. `CreateTime`, `UpdateTime`, and `ExpiresIn` retain their second-based units; stored keys without `ExpiresAt` retain the previous expiration rules.

Creation writes, confirmation, consumption, and revocation are serialized within each ShortKey manager instance. Creation and consumption prefer `adapter.AtomicStorage`; plain `Storage` supports a local fallback. This instance lock does not coordinate separate managers or processes, and atomic create/consume operations do not make cross-instance confirmation and revocation transactional.

```go
createdKey, err := dtoken.CreateShortKey(ctx)
if err != nil {
	return err
}

confirmedKey, err := dtoken.ConfirmShortKey(ctx, createdKey.Key, "user-1001")
if err != nil {
	return err
}

result, err := dtoken.ConsumeShortKey(ctx, confirmedKey.Key)
if err != nil {
	return err
}
fmt.Println(result.ShortKey.LoginID)
```

## SSO

For unified login, ticket exchange, cross-system login-state sharing, and unified logout. SSO lives in the optional module `github.com/Zany2/dtoken-go/sso`; it is not coupled to the base auth architecture and only depends on storage and codec adapters. It provides primitives for Ticket, shared token, remote session, and OAuth2 authorization-code modes, plus HTTP service wrappers for redirect, token exchange, introspection, revoke, userinfo, logout, and callback handling. See [SSO](../../../sso/README.md) for details.

```go
server := sso.NewServer()
err := server.RegisterClient(&sso.Client{
	ClientID:     "app-a",
	ClientSecret: "secret-a",
	RedirectURIs: []string{
		"https://app.example.com/sso/callback",
	},
	Modes: []sso.Mode{sso.ModeTicket},
})
if err != nil {
	return err
}

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
