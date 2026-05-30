# SSO 单点登录

SSO 用于把多个业务系统接入同一个统一登录中心。SSO 能力位于独立模块 `github.com/Zany2/dtoken-go/sso`，不会默认绑定到基础认证鉴权架构中；默认构造器已经内置 JSON 编解码和内存存储，适合本地验证和单元测试，生产环境推荐使用 `github.com/Zany2/dtoken-go/sso/storage/redis` 或自定义持久化存储。当前已经提供 Ticket、共享 Token、远程会话和 OAuth2 授权码四类模式原语：用户在统一登录中心完成登录后，服务端可以按目标应用配置生成对应凭证，子应用再通过校验、消费或远程会话检查创建自己的本地登录态。

模块内已提供协议端点、参数名、签名工具、Server 选项、Client URL 构建辅助和基于 `net/http` 的基础服务端路由。Gin、Fiber、Echo 等框架可以直接挂载标准库 Handler，后续也可以继续补充更贴近各框架的集成封装。

## 模块边界

- `Server`：统一登录中心核心，负责注册子系统、生成/校验/消费 Ticket、共享 Token、远程会话和授权码。
- `Client`：接入统一登录中心的业务系统配置，例如后台系统、开放平台、订单系统。
- `ClientApp`：子系统侧辅助对象，用于构建授权地址、Ticket 校验地址和单点注销地址。
- `HTTPServer`：基于标准库 `net/http` 的 SSO 服务端路由，支持授权重定向、Ticket 换取和注销。
- `CookieOptions`：同域共享 Cookie 辅助配置，适合统一主域下的轻量 SSO。
- `Endpoints` / `ParamNames`：统一维护 SSO HTTP 端点和参数名。
- `Signer`：基于 HMAC-SHA256 的请求参数签名工具。

## 子包结构

| 子包 | 说明 |
| --- | --- |
| `sso` | 核心协议、凭证模型、服务端 API 和向后兼容入口 |
| `sso/httpserver` | 标准库 `net/http` 接入外观，适合新项目按模块引入 |
| `sso/codec/json` | 内置 JSON 编解码器 |
| `sso/storage/memory` | 内置内存存储，仅建议本地验证和单元测试使用 |
| `sso/storage/redis` | Redis 生产存储组件 |

根包会继续保留 `sso.NewServer()`、`sso.NewHTTPServer()`、`sso.JSONCodec`、`sso.MemoryStorage` 等旧入口，已有代码无需修改。

## 适用场景

- 统一登录中心向多个子系统分发登录态。
- 子系统通过一次性 Ticket 换取自己的本地 Token。
- 可信系统之间复用短期共享 Token。
- 子系统通过远程会话 ID 向统一登录中心实时校验登录态。
- 通过类似 OAuth2 授权码的流程对接标准化回调。
- 后台管理、用户中心、开放平台等多应用登录场景。
- 需要把 SSO 客户端、回调地址、授权范围集中管理的系统。

## 基本流程

1. 服务端注册 SSO 客户端，配置 `clientId`、`clientSecret`、回调地址和允许的模式。
2. 用户访问子系统，子系统发现未登录后跳转到统一登录中心。
3. 用户在统一登录中心完成登录。
4. 统一登录中心调用 `GenerateTicket` 生成一次性 Ticket，并重定向到子系统回调地址。
5. 子系统调用 `ConsumeTicket` 校验并消费 Ticket。
6. 子系统根据返回的 `LoginID` 创建自己的本地登录态。

## 设计路线

| 能力 | 定位 | 说明 |
| --- | --- | --- |
| `ModeTicket` | 默认推荐模式 | 子系统通过一次性 Ticket 换取用户身份，适合大多数 Web SSO |
| `ModeSharedToken` | 可信内部系统模式 | 可信系统间复用短期共享 Token，适合内网和强信任场景 |
| `ModeRemoteSession` | 中心化会话模式 | 子系统不保存完整登录态，每次向统一登录中心远程校验会话 |
| `ModeOAuth2` | 授权码式扩展 | SSO 场景下的授权码原语，不等同完整 OAuth2 Token Server |

## 示例

```go
ctx := context.Background()

// NewServer 默认自带 JSON 编解码和内存存储，适合快速接入和本地验证。
server := sso.NewServer()

err := server.RegisterClient(&sso.Client{
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

ticket, err := server.GenerateTicket(
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

生产环境推荐使用 Redis 默认组件：

```go
import ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
```

也可以显式注入已有存储：

```go
server := sso.NewServer(
	sso.WithStorage(storage),
)
```

## Client 辅助

```go
app := sso.NewClientApp(sso.ClientConfig{
	ClientID:  "app-a",
	ServerURL: "https://sso.example.com",
	SecretKey: "sign-secret",
	CheckSign: true,
	Endpoints: sso.DefaultEndpoints(),
	Params:    sso.DefaultParamNames(),
})

callbackURL := "https://app.example.com/sso/callback"
authURL, err := app.AuthURL(callbackURL, nil)
if err != nil {
	return err
}

fmt.Println(authURL)

exchangeURL, err := app.ExchangeTicketURLWithRedirect("ticket-value", callbackURL, nil)
if err != nil {
	return err
}

fmt.Println(exchangeURL)

result, err := app.ExchangeTicket(ctx, "ticket-value", callbackURL)
if err != nil {
	return err
}

fmt.Println(result.LoginID)
```

如果子系统需要接收统一登出通知，可以开启回调注册：

```go
app := sso.NewClientApp(sso.ClientConfig{
	ClientID:           "app-a",
	ClientSecret:       "secret-a",
	ServerURL:          "https://sso.example.com",
	SecretKey:          "sign-secret",
	CheckSign:          true,
	RegisterCallback:  true,
	LogoutCallbackURL: "https://app.example.com/sso/logout-callback",
})
```

`ClientApp` 构建授权地址时会把 `callback` 参数带到 SSO Server。SSO Server 在授权成功后记录当前登录主体与子系统回调地址，后续中心执行 `/sso/logout` 时会向已登记的子系统推送注销回调。

子系统可以直接使用内置 Handler 处理回调，框架会完成 POST、表单和签名校验：

```go
mux.HandleFunc("/sso/logout-callback", app.LogoutCallbackHandler(func(r *http.Request, callback sso.LogoutCallback) error {
	// 在这里删除子系统本地 Session、Cookie 或 Token。
	deleteLocalSessionsByLoginID(callback.LoginID)
	return nil
}))
```

`LogoutCallbackMaxAge` 默认 5 分钟，用于拒绝过旧或明显超前的注销回调时间戳，降低回调重放风险。

`ClientApp` 也可以直接调用 SSO Server 的 HTTP 接口：

| API | 说明 |
| --- | --- |
| `ExchangeTicket` | 使用 Ticket 换取登录主体信息 |
| `ExchangeOAuth2Code` | 使用 OAuth2 Code 换取登录主体信息 |
| `Introspect` | 检查 Ticket、共享 Token、远程会话或 OAuth2 Code 是否有效 |
| `UserInfo` | 使用有效凭证读取登录主体信息 |
| `Revoke` | 撤销 Ticket、共享 Token、远程会话或 OAuth2 Code |

## HTTP 重定向接入

`HTTPServer` 提供标准库 Handler，可以直接挂载到 `http.ServeMux`，也可以被 Gin、Echo、Fiber 等框架转接。

```go
server := sso.NewServer()
server.RegisterClient(&sso.Client{
	ClientID:     "app-a",
	ClientSecret: "secret-a",
	RedirectURIs: []string{
		"https://app.example.com/sso/callback",
	},
	Modes: []sso.Mode{sso.ModeTicket},
})

httpSSO := sso.NewHTTPServer(server, sso.HTTPOptions{
	ServerOptions: sso.ServerOptions{
		EnableSLO: true,
		LogoutCallbackTimeout:    3 * time.Second,
		LogoutCallbackBestEffort: false,
		CheckSign: false,
		Endpoints: sso.DefaultEndpoints(),
		Params:    sso.DefaultParamNames(),
	},
	LoginIDResolver: func(r *http.Request) (string, bool) {
		// 在这里接入你的登录态，例如读取中心登录 Cookie 或调用已有认证模块。
		return "user-1001", true
	},
})

mux := http.NewServeMux()
httpSSO.Register(mux)
```

默认会注册：

| 路由 | 说明 |
| --- | --- |
| `GET /sso/authorize` | 校验中心登录态，生成 Ticket，并重定向回子系统 |
| `GET/POST /sso/token` | 子系统使用 Ticket 或 OAuth2 Code 换取登录主体信息 |
| `GET/POST /sso/introspect` | 检查 Ticket、共享 Token、远程会话或 OAuth2 Code 是否有效 |
| `GET/POST /sso/userinfo` | 使用有效凭证读取登录主体、客户端和授权范围 |
| `GET/POST /sso/revoke` | 撤销 Ticket、共享 Token、远程会话或 OAuth2 Code |
| `GET/POST /sso/logout` | 清除共享 Cookie，推送已登记的子系统注销回调，并返回注销结果 |

## 单点注销

`EnableSLO` 默认开启。子系统在跳转登录时携带注销回调地址，登录中心会保存一条客户端会话记录：

```text
/sso/authorize?client=app-a&redirect=https://app.example.com/sso/callback&callback=https://app.example.com/sso/logout-callback
```

当用户从登录中心退出时，`HTTPServer` 会向该用户已登录的子系统并发发送 `POST` 回调，请求表单包含：

| 参数 | 说明 |
| --- | --- |
| `loginId` | 登录主体 ID |
| `client` | 子系统客户端 ID |
| `timestamp` | 回调发起时间 |
| `sign` | 开启签名校验时携带的签名 |

子系统收到回调后应删除自己的本地 Session、Cookie 或 Token。所有回调成功后，登录中心会清理该登录主体的客户端会话记录。

`LogoutCallbackBestEffort` 用于控制失败策略：为 `false` 时任意子系统回调失败会让本次中心注销返回错误；为 `true` 时会尽量推送全部回调，并继续清理中心侧客户端会话记录。`LogoutHTTPClient` 可用于注入自定义 HTTP Client，例如统一代理、TLS 配置或更细的超时控制。

为了避免恶意 `callback` 造成 SSRF，登录中心只会登记属于当前客户端的注销回调地址：完整匹配 `RedirectURIs`、与某个 `RedirectURIs` 同源，或匹配 `AllowOrigins` 中配置的来源。

## Redis 存储

生产环境建议把 SSO Server 切换到 Redis 存储。Ticket、OAuth2 Code 这类一次性凭证依赖原子读删，Redis 组件已经提供对应能力；ClientSession 这类单点注销记录也可以跨多实例共享。

```go
import ssoredis "github.com/Zany2/dtoken-go/sso/storage/redis"

server, err := ssoredis.NewServer(
	"redis://:password@127.0.0.1:6379/0",
	sso.WithKeyPrefix("dtoken:"),
	sso.WithAuthType("sso:"),
	sso.WithConfig(sso.DefaultConfig()),
)
if err != nil {
	return err
}
```

Redis 下建议重点验证四条链路：Ticket 生成与消费、OAuth2 Code 生成与消费、`RegisterClientSession` 写入客户端会话、`/sso/logout` 推送回调后清理客户端会话。

也可以通过环境变量运行 Redis 可选集成测试：

```powershell
$env:DTOKEN_SSO_REDIS="redis://:password@127.0.0.1:6379/0"
go test ./sso/storage/redis/... -v
```

## 共享 Cookie

同主域部署时，可以使用共享 Cookie 作为登录中心会话来源。它适合 `sso.example.com`、`app-a.example.com`、`app-b.example.com` 这类场景。

```go
cookie := sso.CookieOptions{
	Name:     "dtoken_sso",
	Domain:   ".example.com",
	Path:     "/",
	MaxAge:   2 * time.Hour,
	HTTPOnly: true,
	Secure:   true,
	SameSite: http.SameSiteLaxMode,
}

// 登录中心登录成功后写入共享 Cookie。
sso.SetLoginIDCookie(w, cookie, "user-1001")

// HTTPServer 可以直接从共享 Cookie 解析当前登录用户。
httpSSO := sso.NewHTTPServer(server, sso.HTTPOptions{
	Cookie:          cookie,
	LoginIDResolver: sso.LoginIDFromCookie(cookie),
})
```

## 签名约定

```go
values := url.Values{}
values.Set("client", "app-a")
values.Set("ticket", "ticket-value")

signer := sso.NewSigner("sign-secret")
signedValues := signer.AttachSign(values)

if !signer.Verify(signedValues) {
	return errors.New("invalid sign")
}
```

## 核心 API

| API | 说明 |
| --- | --- |
| `RegisterClient` | 注册 SSO 客户端 |
| `UnregisterClient` | 注销 SSO 客户端 |
| `GetClient` | 查询 SSO 客户端配置 |
| `GenerateTicket` | 使用默认有效期生成一次性 Ticket |
| `GenerateTicketWithTimeout` | 使用指定有效期生成一次性 Ticket |
| `ValidateTicket` | 只校验 Ticket，不消费 |
| `ConsumeTicket` | 校验并消费 Ticket |
| `RevokeTicket` | 主动撤销 Ticket |
| `GetTicketTTL` | 查询 Ticket 剩余有效期 |
| `GenerateSharedToken` / `ValidateSharedToken` | 生成并校验可复用的 SSO 共享 Token |
| `RevokeSharedToken` / `GetSharedTokenTTL` | 撤销共享 Token 并查询剩余有效期 |
| `CreateRemoteSession` / `ValidateRemoteSession` | 创建并校验中心化远程会话 |
| `RenewRemoteSession` / `RevokeRemoteSession` | 续期或撤销远程会话 |
| `GenerateOAuth2Code` / `ConsumeOAuth2Code` | 生成并消费 SSO OAuth2 授权码 |
| `RevokeOAuth2Code` / `GetOAuth2CodeTTL` | 撤销授权码并查询剩余有效期 |
| `RegisterClientSession` | 记录子系统登录绑定，用于单点注销回调推送 |
| `GetClientSessions` | 查询指定登录主体已登记的子系统会话 |
| `ClearClientSessions` | 清理指定登录主体的子系统会话记录 |

## 注意事项

- Ticket 是一次性凭证，成功消费后会从存储中删除。
- `ConsumeTicket` 会校验客户端密钥、目标客户端、回调地址、过期状态和允许的 SSO 模式。
- `sso.NewServer()` 内置的 `MemoryStorage` 只适合本地调试和测试，进程重启后数据会丢失，也不适合多实例部署。
- 生产环境建议使用 `sso/storage/redis`，Redis 存储已经实现原子读删能力，适合一次性 Ticket 和 OAuth2 Code 消费场景。
- 生产环境建议开启 `CheckSign` 并配置 `SecretKey`，让 Server 与 Client 之间的换票、检查和注销回调都具备防篡改能力。
- 注销回调地址会按客户端注册信息校验来源，不建议把过宽的域名加入 `AllowOrigins`。
- 如果使用自定义存储，Ticket 和 SSO OAuth2 授权码消费需要存储实现 `adapter.AtomicStorage`，保证读取并删除是原子操作。
- `ModeSharedToken` 适合可信系统内部复用短期凭证，默认按客户端维度校验。
- `ModeRemoteSession` 适合子系统不保存完整登录态、每次向统一登录中心远程校验的场景。
- `ModeOAuth2` 是 SSO 场景下的授权码原语，不等同于完整 OAuth2 Token Server。
- `Signer` 默认忽略 `sign` 字段本身，并按参数名和值排序后签名，适合 Server 与 Client 之间做请求防篡改。
- 当前 HTTP 路由已覆盖 `ModeTicket` 重定向换票、OAuth2 Code 换取、凭证检查、凭证撤销、用户信息读取和统一登出回调推送。

## 测试与示例

- [SSO 测试说明](./TESTING_zh.md)
- [Gin SSO Server 示例](../examples/sso_gin_server/README_zh.md)
- [Gin SSO Client 示例](../examples/sso_gin_client/README_zh.md)
- [标准库 SSO Server 示例](../examples/sso_server/README_zh.md)
- [标准库 SSO Client 示例](../examples/sso_client/README_zh.md)
