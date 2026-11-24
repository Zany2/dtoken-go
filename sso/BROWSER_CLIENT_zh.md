# 浏览器客户端完整接入

`BrowserClient` 在 `ClientApp` 上组合登录发起、浏览器状态绑定、回调校验、凭证兑换／查询、会话期限计算和退出处理器。应用仍管理本地会话，可接入根包 `dtoken.Auth` 或已有会话系统，无需全局注册。已有 `ClientApp` 接口不变，浏览器组件为可选接入。

## 组装与路由

退出 Origin 校验接受等价的主机大小写、IP 写法及默认端口，拒绝路径（包括 `/`）、查询串、片段、用户信息、多来源及 `null`。这不会放宽授权回调地址的精确注册规则。未知状态 Cookie `SameSite` 值和非法回调端口在构造时拒绝。

客户端协议请求在发送前及限量读取响应体后检查取消状态。自定义传输在取消后返回成功，也不能继续建立浏览器会话。已消费的登录状态保持已消费，应重新发起登录，不要重放该回调。

浏览器处理器还在创建状态前、持久化返回后及应用登录／退出钩子返回后检查取消。状态持久化完成时已取消，不再设置新的状态 Cookie 或授权跳转；钩子取消后不再发送框架的成功跳转。钩子自身可能已完成业务操作或写入 Cookie，这些副作用不回滚；已存储状态按 TTL 过期，已消费状态保持已消费。

下面的 `stateStore` 是应用初始化的 `contract.Storage`；`establishSession` 与 `clearSession` 是应用会话函数，契约见下一节。`clientApp` 应已配置客户端 ID、密钥、中心地址及所需的注销回调登记；生产环境同时配置签名与 [防重放](./REPLAY_PROTECTION_zh.md)。

```go
browser, err := sso.NewBrowserClient(clientApp, sso.BrowserClientOptions{
    Storage: stateStore,
    CallbackURL: "https://app.example/sso/callback",
    StateCookie: sso.CookieOptions{
        Name: "app_login_state",
        SecretKey: stateCookieSecret,
        MaxAge: 5 * time.Minute,
    },
    SessionMaxAge: 2 * time.Hour,
})
if err != nil {
    return err
}

mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.Header().Set("Allow", "GET")
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    if err := browser.BeginLogin(w, r, "/orders", nil); err != nil {
        http.Error(w, "cannot start login", http.StatusInternalServerError)
    }
})
mux.HandleFunc("/sso/callback", browser.CallbackHandler(establishSession))
mux.HandleFunc("/logout", browser.LogoutHandler(clearSession, false, "/"))
mux.HandleFunc("/logout-all", browser.LogoutHandler(clearSession, true, "/"))
mux.HandleFunc("/sso/logout-callback", clientApp.LogoutCallbackHandler(
    func(r *http.Request, callback sso.LogoutCallback) error {
        return revokeLocalSessionsByLoginID(r.Context(), callback.LoginID)
    },
))
```

受保护页面先查询应用本地登录态，未登录时调用 `BeginLogin`，已登录时照常处理。可把经过校验的当前应用路径传入 `returnTo`，空值默认为 `/`；额外授权参数通过 `extra url.Values` 传入。组件会用随机状态覆盖 `Params.Back`，业务返回地址只保存在服务端状态中。

Gin 可直接用 `gin.WrapF(browser.CallbackHandler(...))`、`gin.WrapF(browser.LogoutHandler(...))`，或在适配器中传入 `c.Writer` 和 `c.Request`。无需为每个框架复制状态校验和兑换逻辑。

## 应用会话契约

```go
func establishSession(w http.ResponseWriter, r *http.Request, result *sso.BrowserLoginResult) error
func clearSession(w http.ResponseWriter, r *http.Request) error
```

- `establishSession` 使用 `result.Identity.LoginID`、授权范围及元数据建立本地会话；服务端会话截止时间与 Cookie 截止时间都不得超过 `result.ExpiresAt`。不要仅通过 Cookie 控制过期，也不要在后续自动续期时越过该截止时间。
- 新会话建立成功后，再撤销该浏览器旧会话。若建立失败，保持旧会话不变并返回错误。设置 Cookie 可以在钩子中完成，响应体及跳转由处理器负责。
- `clearSession` 撤销当前浏览器的服务端会话并清除本地 Cookie，应可重复调用。中心注销回调则按 `LoginID` 撤销该账号的本地会话，继续复用 `ClientApp.LogoutCallbackHandler` 的签名校验。
- 回调处理器成功后以 303 跳转到保存的返回路径。会话钩子失败返回 500，不执行成功跳转。
- Ticket／OAuth2 Code 使用现有兑换接口；共享 Token／远程会话通过 `UserInfo` 校验有效状态、模式、客户端归属和身份后才创建本地会话。共享凭证不会被组件另存为浏览器会话。
- 本地期限取 `SessionMaxAge` 和中心提供的有效期限中的最短值，并计入中心请求耗时。Ticket 兑换的 `0` 表示未提供期限，`-1` 表示不限期，均受本地上限约束；共享 Token／远程会话查询的剩余秒数为 `0` 时拒绝登录。

若需要应用自定义错误页面或日志，可直接调用 `CompleteLogin`，自行执行会话钩子和跳转。默认处理器返回固定错误消息，不把中心响应体、Ticket 或内部存储错误写入浏览器响应。不要将直接返回的详细错误原样展示给用户。

## 状态、并发与部署

每次登录生成独立随机状态和浏览器绑定值。状态位于授权 URL 中，绑定值位于带签名的 HttpOnly Cookie 中，服务端只保存绑定值的哈希、返回路径及截止时间。回调必须同时具备匹配的状态记录和 Cookie。

每次流程使用独立 Cookie 名，回调只清理自己的 Cookie，因此不同标签页不会互相覆盖待完成状态。并行登录最终仍使用应用共用的本地会话 Cookie；若同时登录不同账号，由应用决定最终会话策略，组件不保证标签页账号隔离。

先验证回调参数与浏览器绑定，再消费状态。支持 `contract.AtomicStorage` 时使用 `GetAndDelete`，同一状态跨副本只允许一个回调进入兑换；普通存储在同一 `BrowserClient` 实例内通过锁保护 `Get`／`Delete`，不保证跨实例并发唯一消费。

状态消费后，无论兑换失败、响应丢失还是本地会话建立失败，都需要重新发起登录；刷新原回调不会再次建立会话。错误浏览器绑定和重复参数不会消费有效状态。存储操作失败不会降级放行。

组件借用存储，不负责关闭。默认键前缀按客户端 ID 和回调 URL 派生，不同环境建议显式设置独立 `KeyPrefix`。多副本使用同一存储、前缀、Cookie 签名密钥和配置。记录使用 JSON 字符串存储，兼容现有 Redis 组件；内存存储重启后原登录流程失效。状态过期由 TTL 和记录内截止时间共同约束，无需后台清理任务。登录入口的请求频率限制由宿主应用负责。

Cookie 默认仅限当前主机、路径为回调路径、SameSite=Lax、HttpOnly=true；HTTPS 回调自动使用 Secure。Cookie 必须能覆盖回调路径，禁止跨域 Cookie 与 SameSite=Strict 配置，避免跨站中心返回时丢失绑定。不同应用在同一主机上应使用不同 Cookie 名前缀。`StateCookie.SecretKey` 应作为独立、稳定的应用密钥配置。

## 返回地址和退出

返回地址只允许 `/orders?id=1` 这样的应用内路径，拒绝绝对 URL、`//host`、反斜杠和控制字符，包括对应转义形式。回调参数中的额外返回地址不会覆盖服务端保存值。

`LogoutHandler` 仅接受 POST，并要求唯一 `Origin` 与配置的 `CallbackURL` 源一致；缺少或不匹配时返回 403。通过浏览器同源表单或请求调用，反向代理部署时使用实际外部回调地址。GET 退出链接应替换为 POST 按钮。直接 API 客户端的认证与 CSRF 策略由应用另行组合。

- 本地退出：调用本地撤销钩子后，303 跳转到提供的应用内 `returnTo`。
- 全局退出：先构造中心注销 URL，再撤销本地会话，303 跳转到中心。中心仍依据自己的登录 Cookie 解析可信账号；最终注销结果由中心报告，当前接口不提供中心完成后自动返回子应用的协议。
- 本地退出成功不代表所有子应用已退出；中心失败不会恢复已退出的本地会话。退出不会取消此前其他标签页尚未完成的授权流程。

## 示例与验证

标准库客户端增加可选完整示例，保留默认旧入口。从工作区根目录手动运行：

```powershell
go run ./examples/sso_client -browser-flow
```

配合现有中心示例，访问 `/protected` 完成登录，页面提供本地／全局退出 POST 按钮。该示例复用现有内存会话和开发环境配置，生产环境需要应用自己的持久化会话及签名配置。实现见 [browser.go](../examples/sso_client/browser.go)。

回归测试包含四种模式、签名防重放组合、自定义协议参数、多标签页、跨浏览器拒绝、重复回调、原子与普通存储并发、存储故障、会话期限和退出来源校验。命令见 [测试指南](./TESTING_zh.md)。

## PKCE 自动接入

为 Ticket 或 OAuth2 授权码设置 `BrowserClientOptions.PKCE: true`，组件会按流程生成并保存私有 verifier，授权时发送 S256 挑战、回调时从已绑定的服务端状态取出证明并通过 POST 换票。中心可用 `Client.RequirePKCE: true` 强制要求证明。两项配置默认关闭，共享 Token 和远程会话不支持此选项。切换选项后旧待完成流程必须重新发起；详见 [PKCE 指南](PKCE_zh.md)。
