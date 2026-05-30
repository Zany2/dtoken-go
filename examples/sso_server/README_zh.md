# SSO Server 示例

这个示例演示一个最小统一登录中心：

- `/login`：模拟 SSO 登录页。
- `/sso/authorize`：生成 Ticket 并重定向回子系统。
- `/sso/token`：让子系统使用 Ticket 换取登录主体信息。
- `/sso/logout`：清除 SSO 中心 Cookie，并推送子系统统一登出回调。

子系统在跳转登录时会携带 `callback` 参数，SSO Server 授权成功后会记录该回调地址。用户从登录中心退出时，Server 会通知已登记的子系统清理本地登录态。

## 启动

```powershell
go run ./examples/sso_server
```

默认监听：

```text
http://localhost:9000
```

需要同时启动 `examples/sso_client`，然后访问：

```text
http://localhost:9001/protected
```
