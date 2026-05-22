# DAIXK.md

## core/manager 文件用途记录

| 文件 | 用处 |
| --- | --- |
| `accessors.go` | 提供 `Manager` 内部组件的只读访问方法，比如配置、存储、序列化器、日志器、协程池、Nonce/OAuth2 管理器等。 |
| `auth_access.go` | 定义权限/角色外部提供器 `AccessProvider`，并处理 Provider 与 Session 缓存权限/角色之间的回退逻辑。 |
| `auth_disable.go` | 账号封禁、服务封禁、设备封禁、具体设备封禁相关逻辑，包括封禁、解封、状态检查、TTL 查询。 |
| `auth_login.go` | 登录核心逻辑，包括登录、续登、登录态校验、token 信息查询、TTL 查询、手动续期。 |
| `auth_logout.go` | 登出、踢下线、顶下线、终端移除、token 状态标记和 token 元数据清理。 |
| `auth_permission.go` | 权限管理和校验逻辑，包括添加/删除权限、按账号或 token 查询权限、AND/OR 权限校验、权限通配符匹配。 |
| `auth_role.go` | 角色管理和校验逻辑，包括添加/删除角色、按账号或 token 查询角色、AND/OR 角色校验。 |
| `auth_session.go` | Session 和终端查询能力，包括按账号/token 获取 Session、获取 token 列表、在线终端数量、终端遍历、搜索 token/session。 |
| `constructor.go` | `Manager` 构造和关闭逻辑，负责注入配置、存储、序列化器、日志器、协程池、Nonce/OAuth2 等组件。 |
| `consts.go` | `manager` 包内部常量定义，包括 key 前缀、Session 字段名、权限分隔符、Token 逻辑状态。 |
| `feature_nonce.go` | Nonce 功能门面，委托 `nonceManager` 完成 nonce 生成、验证、消费、TTL 查询。 |
| `feature_oauth2.go` | OAuth2 功能门面，委托 `oauth2Manager` 完成客户端注册、授权码、token、刷新、校验、撤销等操作。 |
| `internal_concurrency.go` | 登录并发策略处理，包括是否允许并发、是否共享 token、最大登录数限制、超限淘汰旧终端。 |
| `internal_events.go` | 统一事件触发入口，构建事件数据并按配置同步或异步派发。 |
| `internal_keys.go` | 统一生成存储 key，比如 token、session、renew、active、封禁相关 key。 |
| `internal_lifecycle.go` | 管理后台任务生命周期，目前主要用于续期协程池状态日志任务的启动和停止。 |
| `internal_runtime.go` | 运行时辅助逻辑，包括账号级写锁、异步任务提交、有限 TTL 续期、登录失败回滚。 |
| `internal_session.go` | Session 和 TokenInfo 的内部读取、登录态核心校验、过期终端清理。 |
| `internal_storage.go` | 存储辅助逻辑，包括过期时间计算、序列化保存、Session TTL 保护、设备参数解析、分页搜索。 |
| `internal_terminal.go` | `Session` 终端列表和权限/角色集合的内部操作，比如移除终端、查找终端、添加/删除权限角色。 |
| `internal_token.go` | token 逻辑状态和存活性判断，包括 kicked out、replaced、active timeout 状态映射，以及无副作用 token 活性检查。 |
| `manager.go` | 核心数据结构定义，包括 `Manager`、`TokenInfo`、`Session`、`TerminalInfo`、各类封禁信息结构。 |
