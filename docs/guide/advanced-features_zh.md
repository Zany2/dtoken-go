# 高级能力

本页整理 DToken-Go 的高级能力入口。部分能力仍在开发中，README 的核心特性表中会使用 `🚧` 标识。

## Token Introspection 🚧

用于标准化查询 Token 是否有效、归属信息、TTL 和失效原因。

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

用于独立刷新令牌的签发、刷新、撤销、过期和轮换。

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

## Ticket 临时凭证 🚧

用于一次性票据、临时授权和系统间换票。

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

## 短 Key 访问凭证 🚧

用于短链接访问、扫码确认、临时授权和系统间换票。

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

## SSO 单点登录 🚧

用于统一登录、票据交换、跨系统登录态共享和统一登出。

```go
ssoTicket, err := dtoken.CreateSSOTicket(ctx, "user-1001")
if err != nil {
	return err
}

token, err := dtoken.ExchangeSSOTicket(ctx, ssoTicket)
if err != nil {
	return err
}

_ = dtoken.LogoutAllApps(ctx, "user-1001")
```

## Nonce 防重放

用于一次性随机值生成、校验和消费，防止请求重放。

```go
nonce, err := dtoken.GenerateNonce(ctx)
if err != nil {
	return err
}

err = dtoken.VerifyAndConsumeNonce(ctx, nonce)
```

## 事件监听

事件系统可以监听登录、登出、续期、权限、角色、封禁、解封等核心生命周期事件。

```go
eventMgr := mgr.GetEventManager()
eventMgr.RegisterFunc(listener.EventAll, func(data *listener.EventData) {
	fmt.Println(data.Event, data.LoginID, data.Token)
})
```
