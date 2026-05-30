# Gin SSO Client 示例

这个示例演示使用 Gin 接入统一登录中心的业务系统。

- `/protected`：受保护资源，未登录时跳转 SSO Server。
- `/sso/callback`：接收 Ticket，调用 SSO Server `/sso/token` 换取 `loginId`。
- `/sso/logout-callback`：接收统一登出回调，并清理本地会话。
- `/logout`：只清理当前子系统本地登录态。

示例客户端把本地登录态保存在进程内存，Cookie 中只保存本地 `sessionId`。`/sso/logout-callback` 使用 `ClientApp.LogoutCallbackHandler` 处理，实际项目开启签名后会在这里完成验签。

## 启动

先启动 Gin SSO Server：

```powershell
go run ./examples/sso_gin_server
```

再启动 Gin SSO Client：

```powershell
go run ./examples/sso_gin_client
```

访问：

```text
http://localhost:9101/protected
```

浏览器会跳转到 SSO Server 登录页，登录后带 Ticket 回到 Client，并创建子系统本地登录态。

## 验证统一登出

登录成功后访问：

```text
http://localhost:9100/sso/logout?loginId=user-1001
```

SSO Server 会推送 `/sso/logout-callback`，Client 收到回调后会删除 `user-1001` 的本地会话。
