# SSO 单点登录

入站查询串与表单默认受大小限制，覆盖中心六个接口、注销回调和浏览器回调。配置、413 响应及中间件兼容说明见[HTTP 请求大小限制](REQUEST_LIMITS_zh.md)。

[完整接入示例](../examples/sso_deployment/README_zh.md) 将中心与多子应用的签名、防重放、PKCE、共享 Auth 会话和注销串联起来；Redis、HTTPS 代理、副本和故障处理见[部署与运维](DEPLOYMENT_zh.md)。

应用启动推荐使用 `NewHTTPServerChecked`、`NewClientAppChecked` 和 `RegisterClientChecked`。默认值、兼容边界和错误处理见[配置与启动校验](CONFIGURATION_zh.md)。

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
defer server.Close() // URL/Config 构造器创建的 Redis Client 由 Server 负责关闭。
```

也可以显式注入已有存储：

```go
server := sso.NewServer(
	sso.WithStorage(storage),
)
```

`server.Close()` 默认不会关闭注入的已有存储；请由创建该存储的调用方负责关闭。

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

// 机密客户端通过 POST 换取身份，客户端密钥不会进入 URL。
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

`ClientApp` 在构建授权地址以及调用 `ExchangeTicket` / `ExchangeOAuth2Code` 时都会携带 `callback` 参数。开启 `EnableSLO` 后，SSO Server 在授权成功或凭证消费成功后记录登录主体与子系统回调地址，因此直接通过 Server API 签发的凭证也能接入单点注销。重复登记会更新同一主体、同一客户端的绑定。后续中心执行 `/sso/logout` 时会向已登记的子系统推送注销回调。

兑换接口在消费凭证前检查回调归属和 `MaxRegisteredClient`。回调地址无效、客户端凭证错误、登录回调地址不匹配或登记数量达到上限时，不会消费凭证。登记身份取自成功消费的凭证，不使用请求传入的 `loginId`。消费后发生存储故障仍可能导致登记失败，此时应重新获取凭证，不应重试已消费的凭证。直接调用 `Server.ConsumeTicket` / `Server.ConsumeOAuth2Code` 仍仅执行凭证操作，不自动登记回调。

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

`ServerURL` 必须是 HTTP(S) 绝对地址，不能包含用户凭据、查询参数或片段；支持并保留网关路径前缀。端点配置必须是路径，不能包含查询参数或片段。

HTTP 请求默认总超时为 10 秒，且不跟随重定向，包括可能转发表单凭证的 307/308。注入的 `HTTPClient` 保留自身超时（包括零值）及显式 `CheckRedirect` 策略；未配置策略时仍拒绝重定向。辅助对象不会修改注入的客户端，请求上下文的取消及更早截止时间仍然有效。

`ClientConfig.MaxResponseBytes` 限制每个协议响应体，包括非 2xx 响应和撤销确认。零或负数使用 `DefaultClientMaxResponseBytes`（1 MiB）；需要返回更大元数据的应用可设置正数字节数，例如 `MaxResponseBytes: 4 << 20`。限制以实际从 `Response.Body` 读取的字节为准（传输层自动解压时为解压后的内容），不依赖 `Content-Length`。恰好达到上限且内容合法的响应可正常接受，超限探测最多额外读取一个字节。超大或不可读响应会返回错误，不返回部分结果；完成请求后的各处理路径都会关闭响应体。这不改变服务端注销回调确认独立的 64 KiB 上限。

响应必须包含数值型 `code`，身份获取和凭证检查还要求非空 `data`。交换凭证和 UserInfo 响应必须包含非空 `loginId`。`Extra` 中的数字解码为 `json.Number`，保留整数精度。`UserInfo` 包含凭证的 `active`、`mode` 和 `expiresIn` 字段。

Ticket 与 OAuth2 Code 换票还会在客户端辅助对象入口拒绝小于 -1 的 `remainTokenTimeout` 或 `remainSessionTimeout`。零仍表示未指定，-1 表示无限，正数秒值保留整数精度。这些检查用于校验响应，应用仍需自行应用本地会话期限策略；响应被拒绝不会恢复中心已经消费的一次性凭证。

### 补充换票结果

标准 HTTP 授权可通过 `AuthorizationExtra` 将可信中心会话标识绑定到四种模式的凭证：

```go
options.AuthorizationExtra = func(r *http.Request, clientID, loginID string) (map[string]any, error) {
	// 应用函数：从已认证的浏览器会话读取标识，并核对登录主体。
	session, err := resolveAuthenticatedCenterSession(r, loginID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"centerSessionID": session.ID}, nil
}
```

`resolveAuthenticatedCenterSession` 由应用实现，应与 `LoginIDResolver` 使用同一登录态来源。钩子在客户端、回调地址、模式和授权范围检查后执行；报错时不签发凭证、不登记回调。不要直接复制浏览器提交的会话 ID。`Extra` 会通过换票或用户信息接口提供给已认证客户端，只应保存允许该客户端读取的元数据，不应放入中心 Cookie 等可直接使用的登录凭据。

`HTTPOptions.EnrichExchangeResult` 可以在 Ticket 或授权码完成认证并消费后，读取应用自己管理的中心会话信息。钩子接收已认证的客户端 ID 和凭证结果，其中包括签发时保存的 `Extra` 数据。它在兑换阶段的注销回调登记前执行，不要求开启 SLO。

```go
options.EnrichExchangeResult = func(r *http.Request, clientID string, result *sso.TicketExchangeResult) error {
	// 应用函数：读取并校验签发凭证时绑定的中心会话。
	session, err := loadCenterSession(r.Context(), clientID, result.LoginID, result.Extra["centerSessionID"])
	if err != nil {
		return err
	}
	result.DeviceID = session.DeviceID
	result.RemainTokenTimeout = session.RemainTokenTimeout
	result.RemainSessionTimeout = session.RemainSessionTimeout
	if result.Extra == nil {
		result.Extra = make(map[string]any)
	}
	result.Extra["displayName"] = session.DisplayName
	return nil
}
```

`loadCenterSession` 由应用实现。如果会话按设备区分，应在签发凭证时通过 `Extra` 绑定会话标识，并在钩子中校验其归属。兑换请求是服务端之间的请求，通常不携带中心浏览器 Cookie；不要通过请求传入且未经校验的 `loginId`、`deviceId` 或有效期参数选择会话。确实需要向客户端提供中心 Token 时，钩子也可显式填写 `TokenValue`。`LoginID`、`CenterID` 和 `Scopes` 必须保持凭证中的值，修改它们会使兑换失败。

有效期字段描述中心会话，不是短期 Ticket 或授权码的有效期：正数为剩余秒数，零为未指定，`-1` 为永不过期，小于 `-1` 的值会被拒绝。中心会话已过期或撤销时，应返回错误，不应使用零表示过期。客户端应结合适用的正数有效期和自身策略限制本地会话期限；SSO 辅助对象不会自动创建或续期本地会话。

标准库和 Gin 客户端示例会取中心正数有效期的最小值，并受本地两小时上限限制；截止时间从换票前计算，保守计入网络耗时。非法或已耗尽的有效期不能替换已有本地会话。两个配套示例中心使用应用持有的 Auth 实例：授权绑定其 Token 的哈希引用，换票检查原会话并返回 Token 剩余期限及设备 ID，中心注销撤销账号的 Auth 会话。中心 Token 不会返回给子应用。这是登录时的快照，不能替代注销回调。

钩子报错或返回无效结果时，兑换失败，不返回部分数据，也不新增兑换阶段的回调绑定。凭证保持已消费状态，不能重放；重试需要获取新凭证。如果此前浏览器授权已登记回调，该绑定会保留。不配置钩子时保留原有结果。直接调用 `Server.ConsumeTicket`、`Server.ConsumeOAuth2Code`、`TicketResult` 和 `OAuth2CodeResult` 不会执行此 HTTP 钩子。

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
| `GET /sso/authorize` | 校验中心登录态，按 Mode 生成凭证，并重定向回子系统 |
| `POST /sso/token` | 子系统使用 Ticket 或 OAuth2 Code 换取登录主体信息 |
| `POST /sso/introspect` | 检查 Ticket、共享 Token、远程会话或 OAuth2 Code 是否有效 |
| `POST /sso/userinfo` | 使用有效凭证读取登录主体、客户端和授权范围 |
| `POST /sso/revoke` | 撤销 Ticket、共享 Token、远程会话或 OAuth2 Code |
| `GET/POST /sso/logout` | 清除共享 Cookie，推送已登记的子系统注销回调，并返回注销结果 |

## 单点注销

`DefaultServerOptions()` 和 `DefaultHTTPOptions()` 默认开启 `EnableSLO`；零值配置结构体不会自动开启。子系统在跳转登录时携带注销回调地址，登录中心会保存一条客户端会话记录：

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

子系统收到回调后应删除自己的本地 Session、Cookie 或 Token。通知成功（或尽力模式允许继续）后，登录中心执行可选的 `CenterLogout` 钩子，再清理该登录主体的客户端会话记录和浏览器 Cookie。

`LogoutCallbackBestEffort` 用于控制失败策略：为 `false` 时任意子系统回调失败会让本次中心注销返回错误；为 `true` 时会尽量推送全部回调，并继续清理中心侧客户端会话记录。默认回调客户端不会跟随重定向；`LogoutHTTPClient` 可用于注入自定义 HTTP Client，例如统一代理、TLS 配置或更细的超时控制。

为了避免恶意 `callback` 造成 SSRF，登录中心只会登记属于当前客户端的注销回调地址：完整匹配 `RedirectURIs`、与某个 `RedirectURIs` 同源，或匹配 `AllowOrigins` 中配置的来源。

### 注销结果确认

注销回调必须返回 2xx HTTP 状态并确认成功。内置 `LogoutCallbackHandler` 返回包含 `code: 0` 的 JSON。非空 JSON 响应必须包含值为零的整数 `code`；HTTP 200 携带 `{"code":1}` 等业务错误会被视为回调失败。缺失或为 null 的状态码、损坏的 JSON、HTML 页面、响应读取失败和超过 64 KiB 的响应体同样会失败。为兼容已有接入，2xx 状态配合空响应、仅含空白的响应或纯文本 `ok` 仍被接受。自定义处理器应在完成本地注销后才返回成功确认；其他非空响应格式需迁移为 `{"code":0}`。

严格模式会报告所有失败客户端 ID，并保留 Cookie 和客户端绑定，不执行 `CenterLogout`，便于重试。所有已登记目标仍会被尝试通知。尽力模式与处理网络故障时一致，即使确认失败，也继续执行 `CenterLogout`。已经成功的通知不会回滚，重试时可能再次发送。

### 撤销中心会话

当 `LoginIDResolver` 读取服务端会话时，应配置 `HTTPOptions.CenterLogout`，通过同一会话存储或 Auth 实例撤销中心登录态：

```go
options := sso.DefaultHTTPOptions()
options.SecretKey = "replace-with-a-strong-signing-secret"
options.LoginIDResolver = resolveCenterSession
options.CenterLogout = func(r *http.Request, loginID string) error {
	return revokeCenterSession(r.Context(), loginID)
}
httpSSO := sso.NewHTTPServer(server, options)
```

`resolveCenterSession` 和 `revokeCenterSession` 是应用接入同一登录态来源的函数。钩子只接收 `LoginIDResolver` 解析出的可信身份；请求中的 `loginId` 与该身份不一致时会被拒绝。严格模式下子系统通知失败不会执行钩子。即使开启尽力模式，钩子报错仍会使注销失败，并保留回调记录和 Cookie 供重试，因此钩子和子系统回调都应允许重复调用。如果会话已撤销、后续凭证撤销或存储清理失败，不会恢复已撤销的中心会话。`EnableSLO` 为 false 时仍执行 `CenterLogout` 和账号凭证撤销，但跳过客户端通知及绑定清理。不配置钩子时不会撤销应用自己管理的中心会话，框架仍撤销账号的 SSO 凭证并清理浏览器 Cookie。

### 注销后关联凭证失效

新签发的 Ticket、共享 Token、远程会话及 OAuth2 Code 均记录账号的 `LogoutVersion`。HTTP 注销在子系统通知和 `CenterLogout` 成功后调用 `Server.RevokeLoginCredentials` 更新账号版本，再清理回调记录和 Cookie。旧版本凭证不能继续校验、兑换、读取用户信息或续期；HTTP introspect 返回 `active: false`。这是账号级撤销，覆盖同一账号的所有客户端和设备，不影响其他账号或其他认证命名空间。

从其他入口注销中心会话时，应用也应调用：

```go
if err := server.RevokeLoginCredentials(ctx, loginID); err != nil {
	return err
}
```

该方法只撤销 SSO 凭证，不发送子系统通知、不删除应用会话或浏览器 Cookie。它适用于管理员注销或应用自己的会话撤销流程；自然过期不会自动触发它。后续合法登录可以签发新版本凭证。中心登录态必须由应用实际撤销，否则复制的无状态 Cookie 等仍可能用于重新申请新凭证。已兑换后建立的子系统本地会话仍通过注销回调或本地有效期管理。

版本标记通过普通 `Get` / `Set` 存储，不要求 CAS、事务或遍历凭证；多个 Server 实例共享存储、前缀和认证命名空间时共享撤销结果。校验读取失败会返回存储错误，不会按有效或普通失效处理。版本标记永久保存，不应在关联凭证仍可能存在时单独删除；旧凭证数据保留到自身 TTL 到期，`Get*TTL` 返回存储剩余期限，不代表凭证仍有效。应用应使用 Validate 或 Introspect 判断有效性。

升级前没有 `LogoutVersion` 的旧记录仍按原有效期处理，不会被追溯撤销。需要完整覆盖时，应等待旧凭证过期或执行应用可控的数据迁移。并发中已完成校验的请求不会被回滚；已经消费的一次性凭证也不会因后续步骤失败而恢复。

### 读取接口检查中心会话

`HTTPOptions.CheckCredentialActive` 为 `/introspect` 和 `/userinfo` 的四种凭证模式增加应用状态检查。只有客户端认证成功，且凭证通过归属、有效期及注销版本等原有检查后，才会调用钩子。参数是包含存储中登录 ID、客户端 ID、模式、权限范围及 `Extra` 的 `CredentialInfo` 值；其中元数据应按只读处理。

```go
options.CheckCredentialActive = func(r *http.Request, credential sso.CredentialInfo) (bool, error) {
	return isCenterSessionActive(r.Context(), credential.LoginID, credential.Extra["centerSessionID"])
}
```

`isCenterSessionActive` 是应用函数，应使用登录与换票共用的会话来源。会话不存在、过期或已撤销时返回 `false, nil`：introspect 仅返回 `active: false`，userinfo 返回 HTTP 401，不包含身份信息。存储或查询故障应返回错误，不能伪装成普通失效。钩子无法重新激活无效凭证；读取不消费或撤销凭证，`/revoke` 也不依赖此钩子。

未配置钩子时保留原有的 SSO 存储校验行为。此选项不改变 Server 直接校验、签发、换票或子应用本地会话；换票仍单独使用 `EnrichExchangeResult`。`expiresIn` 仍表示 SSO 凭证有效期。两个中心示例为换票及上述读取接口共用无续期副作用的 Auth 会话检查，因此中心 Token 失效后，仍存储的 Ticket 也不能继续读取用户信息。

## Redis 存储

生产环境建议把 SSO Server 切换到 Redis 存储。Ticket、OAuth2 Code 这类一次性凭证依赖原子读删，Redis 组件已经提供对应能力；ClientSession 这类单点注销记录也可以跨多实例共享。每个 Server 实例内的注册操作已串行化；如果多个实例会并发注册同一登录主体，应先为存储层增加原子合并能力，再依赖这种写入模式。

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
$env:DTOKEN_REDIS_URL="redis://:password@127.0.0.1:6379/0"
go test ./sso/storage/redis/... -v
```

## 共享 Cookie

同主域部署时，可以使用共享 Cookie 作为登录中心会话来源。它适合 `sso.example.com`、`app-a.example.com`、`app-b.example.com` 这类场景。必须配置强随机 `SecretKey`；未配置时 Cookie 不会被解析为可信登录身份。

签名 Cookie 包含精确过期时间，服务端会独立于浏览器的 `Max-Age` 检查有效期。升级后，缺少签名截止时间的旧 Cookie 会被拒绝，用户需要重新登录。清除 Cookie 只删除浏览器副本，不能提前撤销这种无状态凭证的其他副本。如需立即撤销，请通过 `LoginIDResolver` 查询服务端会话，并在应用注销流程中使该会话失效。

```go
cookie := sso.CookieOptions{
	Name:     "dtoken_sso",
	Domain:   ".example.com",
	Path:     "/",
	MaxAge:   2 * time.Hour,
	HTTPOnly: true,
	Secure:   true,
	SameSite: http.SameSiteLaxMode,
	SecretKey: "replace-with-a-strong-cookie-signing-secret",
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

### 客户端独立签名密钥

在中心配置 `Client.SecretKey`，在对应应用配置相同的 `ClientConfig.SecretKey`，即可按客户端隔离签名：

```go
err := server.RegisterClient(&sso.Client{
	ClientID:     "app-a",
	ClientSecret: "app-a-authentication-secret",
	SecretKey:    "app-a-signing-secret",
	RedirectURIs: []string{"https://app.example.com/sso/callback"},
	Modes:        []sso.Mode{sso.ModeTicket},
})
if err != nil {
	return err
}

options := sso.DefaultHTTPOptions() // 默认开启 CheckSign。
options.SecretKey = "legacy-client-signing-secret" // 可选的默认密钥。
options.Cookie.SecretKey = "separate-center-cookie-secret"
httpSSO := sso.NewHTTPServer(server, options)

app := sso.NewClientApp(sso.ClientConfig{
	ClientID:     "app-a",
	ClientSecret: "app-a-authentication-secret",
	ServerURL:    "https://sso.example.com",
	CheckSign:    true,
	SecretKey:    "app-a-signing-secret",
})
```

密钥选择顺序为：优先使用 `Client.SecretKey`，仅在该字段为空时使用 `ServerOptions.SecretKey`。授权、兑换、凭证检查、用户信息、撤销、注销请求及发往子系统的注销回调都遵循这一规则。验签可额外接受同组的 VerificationKey，但失败时不会再尝试其他组的默认密钥。`ClientSecret` 仍是客户端认证凭据，不会被隐式用作签名密钥。`CheckSign: false` 保持原有行为。

带客户端 ID 的签名请求会读取该客户端当前的注册配置；未知客户端和存储读取失败会被拒绝。同名客户端参数重复时会作为歧义请求拒绝。未提供客户端 ID 时使用全局密钥；如果开启 `AllowAnonymousClient`，则按匿名客户端配置选择密钥。注销仍要求解析出可信的中心登录身份。全部客户端配置独立密钥后可以不设置全局密钥，但中心登录态仍需配置 `Cookie.SecretKey` 或自定义 `LoginIDResolver`；客户端签名密钥不会用于签发中心 Cookie。

通过 `RegisterClient` 更新客户端配置后，后续请求和注销回调使用新密钥。轮换时需同步调整该应用的 `ClientConfig.SecretKey`，旧密钥签发的请求将失效。未设置 `SecretKey` 的旧客户端继续使用全局密钥。各应用应配置不同密钥，才能获得客户端之间的签名隔离。

## 单点注销可靠性

注销流程的逐客户端结果、实例并发限制、失败重试及结果观察钩子，见 [单点注销可靠性指南](./LOGOUT_RELIABILITY_zh.md)。默认仍只尝试一次；启用更多尝试后，同一轮只重试可恢复的失败目标。

## 防重放保护

通过显式配置，可为请求与注销回调整体启用时间戳检查、nonce 去重和操作绑定签名，见 [签名通信与防重放](./REPLAY_PROTECTION_zh.md)。中心与子应用应协调启用，多副本应共用防重放存储。

## 浏览器客户端接入

`BrowserClient` 组合登录发起、独立浏览器状态、四种模式回调校验、本地会话期限和本地／全局退出处理器。本地会话通过应用钩子接入；详见 [浏览器完整接入指南](./BROWSER_CLIENT_zh.md)。标准库客户端可通过 `-browser-flow` 选择完整示例，默认入口保持原行为。

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
| `RevokeLoginCredentials` | 使账号在所有客户端此前签发的带版本凭证失效 |
| `RegisterClientSession` | 记录子系统登录绑定，用于单点注销回调推送 |
| `GetClientSessions` | 查询指定登录主体已登记的子系统会话 |
| `ClearClientSessions` | 清理指定登录主体的子系统会话记录 |

## 注意事项

- Ticket 是一次性凭证，成功消费后会从存储中删除。
- `ConsumeTicket` 会校验客户端密钥、目标客户端、回调地址、过期状态和允许的 SSO 模式。
- `sso.NewServer()` 内置的 `MemoryStorage` 只适合本地调试和测试，进程重启后数据会丢失，也不适合多实例部署。
- 生产环境建议使用 `sso/storage/redis`，Redis 存储已经实现原子读删能力，适合一次性 Ticket 和 OAuth2 Code 消费场景。
- 生产环境建议开启 `CheckSign` 并配置 `SecretKey`，让 Server 与 Client 之间的换票、检查和注销回调都具备防篡改能力。
- 注销回调地址必须是无用户信息、无片段的 HTTP(S) 绝对地址。登记和发送前都会校验当前客户端白名单，已注销客户端会被跳过。不建议把过宽的域名加入 `AllowOrigins`。
- 如果使用自定义存储，框架会优先使用 `adapter.AtomicStorage` 原子消费 Ticket 和 SSO OAuth2 授权码；普通 `Storage` 使用单实例串行回退，不保证跨实例原子性。
- `ModeSharedToken` 适合可信系统内部复用短期凭证，默认按客户端维度校验。
- `ModeRemoteSession` 适合子系统不保存完整登录态、每次向统一登录中心远程校验的场景。
- `ModeOAuth2` 是 SSO 场景下的授权码原语，不等同于完整 OAuth2 Token Server。
- `Signer` 忽略签名字段本身，按参数名排序，并保留同名参数值的顺序，与取首值的读取语义保持一致。验签拒绝空密钥和多个签名值。单值参数签名不变；升级时，同名多值参数需要重新签名。
- 当前 HTTP 授权路由支持按 Mode 签发 Ticket、共享 Token、远程会话或 OAuth2 Code，并提供凭证交换、检查、撤销、用户信息和统一登出回调推送。

## 测试与示例

- [SSO 测试说明](./TESTING_zh.md)
- [Gin SSO Server 示例](../examples/sso_gin_server/README_zh.md)
- [Gin SSO Client 示例](../examples/sso_gin_client/README_zh.md)
- [标准库 SSO Server 示例](../examples/sso_server/README_zh.md)
- [标准库 SSO Client 示例](../examples/sso_client/README_zh.md)

## 授权凭证绑定（PKCE）

Ticket 和 OAuth2 授权码支持 PKCE S256：中心通过 `Client.RequirePKCE` 强制绑定，子应用通过 `BrowserClientOptions.PKCE` 自动管理每次登录的证明，也可使用 `AuthURLWithPKCE` 与 `CredentialRequest.CodeVerifier` 手动接入。签发、兑换及 HTTP 身份查询统一校验，错误证明不会在正常消费流程中销毁凭证。两个开关默认关闭，旧接口保留；完整流程、迁移顺序和边界见 [PKCE 指南](PKCE_zh.md)。

## 逐请求远程会话鉴权

`NewRemoteAuth` 为远程会话和共享 Token 提供统一鉴权入口，`Middleware` 可直接保护标准 HTTP 业务处理器，`Authenticate` 可用于框架适配。每次请求重新查询中心，检查客户端归属、模式、有效期及全部所需 Scopes；失败不放行，成功后可从请求上下文读取身份。支持默认 Bearer 输入、自定义会话解析、整体超时和错误回调，不创建本地会话、不缓存成功结果。完整接入和状态边界见 [远程鉴权指南](REMOTE_AUTH_zh.md)。

## 管理员账号下线

管理员强制下线另见[账号下线指南](ACCOUNT_LOGOUT_zh.md)：`HTTPServer.LogoutAccount(r, loginID)` 为可信后台串联中心会话撤销、旧 SSO 凭证失效、子应用通知及可重试的绑定清理，返回完整阶段进度。不新增管理 HTTP 路由；操作者鉴权和账号禁用由应用负责。

## 签名密钥轮换

客户端注册、中心默认组和 ClientApp 可用可选 `VerificationKey` 完成签名密钥过渡，出站仍仅使用 `SecretKey`。三阶段切换及信任边界见[密钥轮换](SIGNING_ROTATION_zh.md)。

## 请求参数一致性

协议字段必须是单值，凭证字段须与所选模式匹配。重复 Query／Body 参数、拒绝顺序及旧歧义输入迁移见[请求参数一致性](REQUEST_VALIDATION_zh.md)。

协调升级、独立模块验证及行为变化见[升级与发布指南](MIGRATION_zh.md)。编译、运行回归和部署验收分别记录结果。

## 动态授权策略

`HTTPOptions.AuthorizationPolicy` 在签发、兑换和身份检查时统一评估当前用户的子应用访问权限。策略可以拒绝访问或缩减 Scopes，不能扩大请求或原始凭证中的授权范围。兑换拒绝不消费凭证；检查拒绝返回 inactive，配合 `RemoteAuth` 可将策略变化反映到后续业务请求。默认不配置时保留现有行为，可信底层 Server API 不自动执行 HTTP 策略。完整接入、错误分类与生命周期说明见 [动态授权策略指南](AUTHORIZATION_POLICY_zh.md)。
