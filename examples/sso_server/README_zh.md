# SSO Server 示例

这个示例演示一个最小统一登录中心：

- `/login`：模拟 SSO 登录页。
- `/sso/authorize`：生成 Ticket 并重定向回子系统。
- `/sso/token`：让子系统使用 Ticket 换取登录主体信息。
- `/sso/logout`：清除 SSO 中心 Cookie，并推送子系统统一登出回调。

子系统在跳转登录时会携带 `callback` 参数，SSO Server 授权成功后会记录该回调地址。用户从登录中心退出时，Server 会通知已登记的子系统清理本地登录态。

这是本地流程演示：`/login` 接受任意登录 ID，不校验密码；ID 为空时默认使用 `user-1001`。服务仅监听 `localhost:9000`。登录提交只读取 POST 表单正文，并使用 Go 标准库的跨源请求保护。协议端点仍支持子系统到认证中心的请求；配套示例关闭了请求签名，但 Ticket 交换仍校验已登记的客户端密钥和回调地址。

认证中心使用内存存储，并在每次进程启动时随机生成 Cookie 签名密钥。重启会清空已登记的会话记录，并使旧的中心 Cookie 失效。Cookie 有效期为两小时。注销会清除浏览器中的 Cookie，但此前复制的 Cookie 在签名有效期内仍可使用；需要立即吊销时，应改用服务端会话解析器并同步删除会话。

注销采用严格回调策略：已登记子系统离线或拒绝回调时，中心注销返回错误，保留中心 Cookie 和回调记录以便重试；成功后才清理两者。体验 `/sso/logout` 时请保持子系统运行。

## 启动

```powershell
go run ./examples/sso_server
```

在仓库根目录使用 Go 1.25 或更高版本执行。用于部署时，应替换模拟登录、配置共享存储和受管理的密钥，并使用 HTTPS 与 Secure Cookie。

默认监听：

```text
http://localhost:9000
```

需要同时启动 `examples/sso_client`，然后访问：

```text
http://localhost:9001/protected
```
