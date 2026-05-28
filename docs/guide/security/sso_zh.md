# SSO 单点登录

SSO 用于把多个业务系统接入同一个统一登录中心。第一版先提供 Ticket 模式：用户在统一登录中心完成登录后，服务端为目标应用生成一次性 SSO Ticket，目标应用在回调地址中拿到 Ticket 后向服务端校验并消费，消费成功后即可在自己的系统中创建本地登录态。

后续会在同一套 SSO 模型上继续补充共享 Token、远程会话、OAuth2 模式和统一登出回调等能力，因此客户端配置中已经预留了 `Modes`、`Scopes`、`AllowOrigins` 和 `Extra` 等字段。

## 适用场景

- 统一登录中心向多个子系统分发登录态。
- 子系统通过一次性 Ticket 换取自己的本地 Token。
- 后台管理、用户中心、开放平台等多应用登录场景。
- 需要把 SSO 客户端、回调地址、授权范围集中管理的系统。

## 基本流程

1. 服务端注册 SSO 客户端，配置 `clientId`、`clientSecret`、回调地址和允许的模式。
2. 用户访问子系统，子系统发现未登录后跳转到统一登录中心。
3. 用户在统一登录中心完成登录。
4. 统一登录中心调用 `GenerateSSOTicket` 生成一次性 Ticket，并重定向到子系统回调地址。
5. 子系统调用 `ConsumeSSOTicket` 校验并消费 Ticket。
6. 子系统根据返回的 `LoginID` 创建自己的本地登录态。

## 示例

```go
ctx := context.Background()

err := dtoken.RegisterSSOClient(&sso.Client{
	ClientID:     "app-a",
	ClientSecret: "secret-a",
	Name:         "App A",
	RedirectURIs: []string{
		"https://app.example.com/sso/callback",
	},
	Modes:  []sso.Mode{sso.ModeTicket},
	Scopes: []string{"profile", "email"},
})
if err != nil {
	return err
}

ticket, err := dtoken.GenerateSSOTicket(
	ctx,
	"app-a",
	"user-1001",
	"https://app.example.com/sso/callback",
	[]string{"profile"},
	map[string]any{"scene": "web"},
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

## 核心 API

| API | 说明 |
| --- | --- |
| `RegisterSSOClient` | 注册 SSO 客户端 |
| `UnregisterSSOClient` | 注销 SSO 客户端 |
| `GetSSOClient` | 查询 SSO 客户端配置 |
| `GenerateSSOTicket` | 使用默认有效期生成一次性 Ticket |
| `GenerateSSOTicketWithTimeout` | 使用指定有效期生成一次性 Ticket |
| `ValidateSSOTicket` | 只校验 Ticket，不消费 |
| `ConsumeSSOTicket` | 校验并消费 Ticket |
| `RevokeSSOTicket` | 主动撤销 Ticket |
| `GetSSOTicketTTL` | 查询 Ticket 剩余有效期 |

## 注意事项

- Ticket 是一次性凭证，成功消费后会从存储中删除。
- `ConsumeSSOTicket` 会校验客户端密钥、目标客户端、回调地址、过期状态和允许的 SSO 模式。
- 如果使用自定义存储，Ticket 消费需要存储实现 `adapter.AtomicStorage`，保证读取并删除是原子操作。
- 当前第一版实现的是 `sso.ModeTicket`；`ModeSharedToken`、`ModeRemoteSession` 和 `ModeOAuth2` 是后续模式预留。
