# SSO Client 示例

这个示例演示一个接入 SSO 的业务系统：

- `/protected`：受保护资源，未登录时跳转 SSO Server。
- `/sso/callback`：接收 Ticket，调用 SSO Server `/sso/token` 换取 `loginId`。
- `/sso/logout-callback`：接收 SSO Server 的统一登出回调，并清理本地会话。
- `/logout`：清除子系统本地登录 Cookie。

示例客户端会把本地登录态存到进程内存，并在 Cookie 中只保存本地 `sessionId`。这样 SSO Server 推送统一登出回调时，可以根据 `loginId` 删除该用户在子系统内的所有本地会话。`/sso/logout-callback` 使用 `ClientApp.LogoutCallbackHandler` 处理，实际项目开启签名后也会在这里完成验签。

## 启动

先启动 SSO Server：

```powershell
go run ./examples/sso_server
```

再启动 SSO Client：

```powershell
go run ./examples/sso_client
```

访问：

```text
http://localhost:9001/protected
```

浏览器会自动跳转到 SSO Server 登录页，登录后带 Ticket 回到 Client，并创建子系统本地登录态。

在 SSO Server 执行 `/sso/logout` 时，Server 会调用本示例的 `/sso/logout-callback`，Client 收到回调后会删除对应用户的本地会话。
