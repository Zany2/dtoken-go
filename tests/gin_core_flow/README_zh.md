# Gin 核心流程测试

本目录包含 `tests/gin_core_app` 的 HTTP 流程测试。

这些测试不会手动请求外部端口，而是通过 `httptest.NewServer` 在测试进程内启动 Gin 应用，然后按照真实 HTTP 流程调用接口。

## 测试列表

- `TestAuthFlow`：测试认证流程。
  - 未携带 token 请求 `/api/me`，期望未授权。
  - 通过 `/login` 登录，期望返回 token。
  - 使用 token 请求 `/api/me`，期望返回当前用户。
  - 通过 `/api/logout` 登出，期望旧 token 被拒绝。

- `TestTokenMetadataAndStatusFlow`：测试 token 元信息和状态接口。
  - 检查有无 token 时的 `IsLogin` 行为。
  - 读取 token 信息、设备、设备 ID、创建时间和超时时间。
  - 使用自定义超时时间登录并验证 TTL。
  - 使用已有 token 调用 LoginByToken。

- `TestPermissionFlow`：测试权限校验。
  - 登录时不授予 `article:read`。
  - 请求 `/api/articles`，期望禁止访问。
  - 通过 `/api/permissions` 授权。
  - 再次请求 `/api/articles`，期望成功。

- `TestPermissionMutationAndLogicFlow`：测试权限变更和逻辑校验。
  - 移除已授权限并验证访问被撤销。
  - 验证 AND 权限校验需要全部权限。
  - 验证 OR 权限校验和通配符权限。

- `TestAccessStatusFlow`：测试布尔型权限和角色校验。
  - 通过 login ID 验证 HasPermission 和 HasRole。
  - 通过 token 验证 HasPermission 和 HasRole。

- `TestAccessListFlow`：测试权限和角色列表接口。
  - 通过 login ID 验证 GetPermissions/GetRoles。
  - 验证 GetPermissionsByToken/GetRolesByToken。

- `TestAccessProviderFlow`：测试外部访问提供者行为。
  - 验证 provider 权限和角色会覆盖 session 中存储的值。
  - 验证按终端区分的 provider 数据可以因设备不同而不同。

- `TestRoleFlow`：测试角色校验。
  - 登录时不授予 `admin`。
  - 请求 `/api/admin`，期望禁止访问。
  - 通过 `/api/roles` 授予角色。
  - 再次请求 `/api/admin`，期望成功。

- `TestRoleMutationAndLogicFlow`：测试角色变更和逻辑校验。
  - 移除已授予角色并验证访问被撤销。
  - 验证 AND 角色校验需要全部角色。
  - 验证 OR 角色校验只需要任意一个角色。

- `TestRenewFlow`：测试 token 续期。
  - 使用较短 token 超时时间登录。
  - 通过 `/api/token/ttl` 读取初始 TTL。
  - 等待 TTL 减少。
  - 通过 `/api/token/renew` 续期，期望 TTL 延长。

- `TestAutoRenewFlow`：测试自动续期。
  - 启用 AutoRenew，并设置刷新阈值和续期间隔。
  - 验证登录后的初始续期间隔会阻止已经满足阈值的请求续期。
  - 间隔结束后访问受保护接口，等待实际 TTL 增长和续期事件。
  - 立即再次访问，验证间隔内不会产生额外的续期事件。
  - 通过 Manager 只读接口观察 TTL，避免测量动作触发续期。

- `TestRenewConfigurationMatrixFlow`：测试关闭续期、刷新阈值和无间隔时的重复续期。
  - 使用有超时限制的存储 TTL 轮询，替代异步续期后的固定等待。

- `TestRenewBoundaryFlow`：测试手动续期边界。
  - 在 HTTP 层拒绝 0 和负数续期值。
  - 接受有效续期值并验证新的 TTL。

- `TestTokenExpiredFlow`：测试 token 过期。
  - 使用 1 秒超时时间登录。
  - 等待超时。
  - 请求受保护接口，期望未授权。

- `TestActiveTimeoutFlow`：测试无操作超时。
  - 使用较长绝对 TTL 和较短 active timeout 登录。
  - 等待无操作超时。
  - 请求受保护接口，期望返回 active-timeout 代码。

- `TestKickoutAndReplaceFlow`：测试 token 状态变更。
  - 踢出当前 token，期望旧 token 被拒绝。
  - 替换当前 token，期望旧 token 被拒绝。

- `TestSessionFlow`：测试 session 查询。
  - 登录。
  - 请求 `/api/session`。
  - 断言 login ID 和终端数量。

- `TestMultiTerminalSessionFlow`：测试多终端。
  - 同一账号分别从 web 和 mobile 登录。
  - 使用任一 token 请求 `/api/session`。
  - 断言终端数量为 2。

- `TestTerminalInspectionFlow`：测试终端元数据查询。
  - 从 web 和 mobile 登录。
  - 请求 `/api/terminal`。
  - 断言终端设备信息和在线数量。

- `TestSessionQueryFlow`：测试 session 查询接口。
  - 按 login ID、设备和具体设备 ID 查询 token 列表。
  - 查询终端列表并遍历终端。
  - 搜索 token 值和 session ID。

- `TestSessionAliveFilterFlow`：测试存活 token 过滤。
  - 将一个终端标记为离线。
  - 验证 token 列表查询只返回存活 session token。
  - 验证离线 token 保留准确的失败原因。

- `TestSessionExpiredTokenFilterFlow`：测试自然过期过滤。
  - 保持移动端登录有效，等待短有效期的 web token 过期。
  - 验证 `alive=false` 保留两个终端条目，`alive=true` 仅返回移动端 token。

- `TestTerminalOperationFlow`：测试终端范围操作。
  - 登出一个具体设备，同时保持另一个终端在线。
  - 踢出某个设备类型的所有终端。
  - 替换某个账号的所有终端。

- `TestTerminalOperationMatrixFlow`：测试账号、设备类型和具体设备操作矩阵。
  - 登出某账号的所有终端。
  - 登出某设备类型的所有终端。
  - 踢出账号终端和具体设备终端。
  - 替换设备类型终端和具体设备终端。

- `TestConcurrencyPolicyFlow`：测试登录并发策略。
  - 同一设备复用共享 token。
  - 不同具体设备 ID 创建新 token。
  - 未提供设备维度时复用账号级 token。
  - 账号级最大登录数量溢出。
  - 溢出模式：logout、kickout、replaced。
  - 设备级最大登录数量溢出。
  - 账号和设备并发范围。
  - 非并发替换，以及账号/设备范围的新设备拒绝。

- `TestDisableFlow`：测试账号禁用和服务禁用。
  - 账号禁用会拒绝旧 token 和新登录。
  - 可以查询账号禁用信息和 TTL。
  - 服务禁用只拒绝 `/api/payment`。
  - 可以查询服务禁用信息、级别和 TTL。

- `TestServiceDisableLevelFlow`：测试服务禁用级别。
  - 以级别 3 禁用某服务。
  - 验证较低或相等级别会被阻止，较高级别允许访问。
  - 解绑服务并验证级别校验被清除。

- `TestUntieFlow`：测试移除禁用状态。
  - 解绑账号禁用并再次登录。
  - 解绑服务禁用并再次访问服务。
  - 解绑设备禁用并从该设备再次登录。

- `TestDeviceDisableFlow`：测试设备类型禁用。
  - 禁用当前账号的 `web` 设备。
  - 可以查询设备禁用信息和 TTL。
  - 验证 web 登录被拒绝。
  - 验证 mobile 登录仍然成功。

- `TestConcreteDeviceDisableFlow`：测试具体设备 ID 禁用。
  - 只禁用 `web/browser-1`。
  - 可以查询具体设备禁用信息和 TTL。
  - 验证 `web/browser-1` 被拒绝。
  - 验证 `web/browser-2` 仍然允许。

- `TestNonceFlow`：测试 nonce。
  - 通过 `/nonce` 生成 nonce。
  - 在不消费 nonce 的情况下检查有效性和 TTL。
  - 通过 `/nonce/verify` 验证一次，期望成功。
  - 再次验证同一个 nonce，期望失败。

- `TestNonceTimeoutFlow`：测试 nonce 自定义 TTL。
  - 生成 1 秒 TTL 的 nonce。
  - 验证它最初有效。
  - 等待过期后验证无法再消费。

- `TestOAuth2AuthorizationCodeFlow`：测试 OAuth2 授权码流程。
  - 生成授权码。
  - 使用授权码换取 access token 和 refresh token。
  - 验证授权码只能使用一次。
  - introspect access token。
  - 刷新后验证旧 access token 和 refresh token 均不可再用，并在撤销前确认新 access token 有效。
  - 撤销刷新后的 token 并验证其无效。

- `TestOAuth2AuthorizationCodeBindingFlow`：测试授权码与客户端、回调地址的绑定。
  - 拒绝其他已注册客户端换码，以及不匹配的回调地址。
  - 验证上述请求被拒绝后，原客户端仍可使用同一授权码合法换码。

- `TestOAuth2PasswordAndClientCredentialsFlow`：测试其他 OAuth2 授权方式。
  - password grant 返回用户 token。
  - client credentials grant 返回客户端 token。
  - 错误 client secret 会被拒绝。

- `TestOAuth2ClientManagementFlow`：测试 OAuth2 客户端管理。
  - 注册并查询客户端。
  - 使用注册后的客户端执行 client credentials。
  - 拒绝未允许的 scope。
  - 注销客户端并验证它不能再使用。

- `TestMultiAuthIsolationFlow`：测试多认证体系隔离。
  - 将同一个 ID 分别登录到 user-auth 和 admin-auth。
  - 验证 token 不能跨认证体系使用。
  - 读取两侧身份，对比权限列表和角色列表，验证 AuthType 隔离。

- `TestDecodeFlowData`：拒绝缺失、null、格式错误及类型错误的数据，并在复用解码目标时清除旧字段。
- `TestFlowClientDefaultsToMemory`：在未设置 Redis 环境变量时执行真实登录和受保护 HTTP 请求。
- `TestFlowStorageCleanup`：删除真实测试实例的键，检查认证和 nonce 失效，并保留同一后端的无关键。

## 运行

自动化流程测试默认使用内存存储。设置 `DTOKEN_REDIS_URL` 后，流程场景会使用 Redis，覆盖其存储和扫描行为。默认命令无需外部服务，也会实际执行流程测试。

每个测试实例使用随机的 `dt-gcf-<hex>:` 键前缀，不同测试进程之间也保持隔离。清理时只扫描并删除该实例前缀下的键。建议使用专门的测试 Redis 数据库。

Redis URL 示例：

```text
redis://localhost:6379/0
redis://:password@localhost:6379/0
```

未设置 `DTOKEN_REDIS_URL` 时，流程测试使用内存存储。配置 Redis 后若初始化失败，或应用配置、创建失败，测试会明确失败，不再静默跳过。

在当前目录运行：

```powershell
go test ./...
```

如需启用 Redis 流程测试，请先设置地址：

```powershell
$env:DTOKEN_REDIS_URL='redis://:password@localhost:6379/0'
go test ./...
```

如果你的本地 Go 环境在 Windows 上设置了 `GOOS=linux`，请切换为 Windows 目标后再运行：

```powershell
$env:GOOS='windows'
$env:GOARCH='amd64'
go test ./...
```
