# 高级能力

本页整理 DToken-Go 在普通登录、登出、权限和角色 API 之外可以直接使用的高级能力。

## Token Introspection

Token Introspection 用来无续期副作用地检查 token 当前是否活跃，并返回归属信息、TTL、权限、角色、扩展数据和非活跃原因。

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

各框架包也会导出同名 API。例如 GoFrame 项目只引入 `github.com/Zany2/dtoken-go/integrations/gf` 后，可以直接调用 `gfdt.IntrospectToken(...)`。

## Refresh Token

普通登录流程支持 access token + refresh token 双令牌。刷新时先一次性消费旧 refresh token，再签发新令牌对，成功后撤销旧 access token。新令牌对签发失败时，旧 refresh token 不恢复；未被其他操作终止的旧 access token 保留到原有效期结束。

访问令牌自然过期不撤销仍有效的刷新凭证；轮换会清理 Session 中过期的终端条目，并保留其他终端及账号数据。显式登出、踢下线或顶替会清理已关联的刷新凭证。

新签发的访问令牌到刷新令牌反向索引沿用刷新令牌有效期，避免访问令牌续期后失去撤销关联。已有短期索引不自动迁移；重新签发或轮换后的令牌对使用新规则。

消费优先使用 `AtomicStorage.GetAndDelete`；基础存储通过同一 Manager 的账号锁串行处理，不保证跨实例或跨进程原子消费。

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

默认 refresh token 有效期是 `30` 天。可以通过 `RefreshTokenTimeout(...)` 设置全局有效期，也可以通过 `LoginWithRefreshTokenOptions(...)` 为单次登录覆盖有效期。

## Ticket 临时凭证

用于一次性票据、临时授权和系统间换票。

Ticket 校验和状态回写使用精确截止时间 `ExpiresAt`。`CreateTime` 仍为 Unix 秒时间戳，`ExpiresIn` 仍为向上取整后的有效秒数。存储中不含 `ExpiresAt` 的旧票据继续使用原有整秒到期规则。

消费时优先使用 `adapter.AtomicStorage`。仅提供基础 `Storage` 时，消费和撤销在同一个 Ticket 管理器实例内串行执行；该回退不保证不同管理器实例或进程之间的原子消费。多个实例共享票据时应使用原子存储。

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

## Short-Key 访问凭证

用于短链访问、扫码确认、临时授权和系统间换票。

短 Key 的校验和状态回写使用精确截止时间 `ExpiresAt`。`CreateTime`、`UpdateTime` 和 `ExpiresIn` 保留原有秒单位；存储中不含 `ExpiresAt` 的旧短 Key 继续使用原有到期规则。

创建写入、确认、消费和撤销在同一个 ShortKey 管理器实例内串行执行。创建和消费优先使用 `adapter.AtomicStorage`，基础 `Storage` 提供本地回退。实例锁不协调其他管理器实例或进程；原子创建和消费也不使跨实例的确认、撤销成为事务操作。

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

## SSO 单点登录

用于统一登录、票据交换、跨系统登录态共享和统一登出。SSO 位于独立模块 `github.com/Zany2/dtoken-go/sso`，不绑定基础认证鉴权架构，只依赖存储和编解码适配器；当前已提供 Ticket、共享 Token、远程会话和 OAuth2 授权码模式原语，也提供 redirect、token exchange、introspection、revoke、userinfo、logout 和 callback 处理等 HTTP 服务封装。更多说明见 [SSO 单点登录](../../../sso/README_zh.md)。

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
