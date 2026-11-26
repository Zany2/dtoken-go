# Memory 方法逻辑验收

本文保留第一轮方法审核时的覆盖快照和行号。后续已补充关键错误分支并删除 4 个无调用方法，最新结果见[分支与遗留方法收尾](2026-11-26-memory-branches_zh.md)。

2026-11-26，按用户要求停止公网 Redis 回归，改用内存验证非 SSO 方法行为。本轮新增 14 个顶层测试，包含终端范围、授权模式、无效配置等子用例；修正一个刷新过期测试的准备时序，没有修改生产代码。

## 结果与统计口径

- `go.work` 中 24 个非 SSO 模块的 vet 和回归全部通过。根模块与 Gin 流程在覆盖运行中验证，其他 22 个模块单独回归；未启动示例应用，未继续连接 Redis。
- 根模块与 Gin 流程合并覆盖后，根模块共有 9,726 条可统计语句，其中 8,659 条执行过，语句覆盖率 **89.0%**。这里包含 `contract/storagetest`，不含独立 Redis 模块和测试应用模块。
- 根模块函数清单含 1,265 项：1,251 项有语句命中，4 个私有辅助函数未命中，10 个 NopLogger 空方法无可执行语句，不能通过覆盖率判断是否调用。
- 921 个导出函数／方法中，除上述 10 个空方法外，**911 个均有语句命中**。这只说明入口已执行，不表示每个方法的全部分支、参数组合或并发时序都已验证。
- 当前编译条件为 Go 1.27.0、Windows/amd64。独立框架模块均跑过测试，但没有纳入本次方法覆盖率合并统计；不能将 911 个入口命中外推为整个工作区每个方法都已覆盖。

测试使用内置 Memory 及已有的内存测试替身。新加的默认门面、OAuth2、Ticket、ShortKey 生命周期使用真实 Memory；既有引擎与 HTTP 单元测试同时保留错误注入、普通存储回退和可控调度替身。没有用替身结果声称 Redis Lua、网络超时或跨进程竞争已经通过。

| 包 | 语句覆盖率 |
| --- | ---: |
| 根包 dtoken | 89.7% |
| internal/engine | 89.8% |
| httpauth / httpauth/context | 95.5% / 79.5% |
| event | 97.6% |
| nonce / oauth2 | 89.1% / 88.8% |
| ticket / shortkey | 80.3% / 83.1% |
| storage/memory | 94.5% |
| migration | 91.8% |

## 本轮补充的行为断言

| 范围 | 验证内容 |
| --- | --- |
| 默认门面与 AuthType | 默认入口、直接实例共享引擎；同一 Memory 上两个命名空间的 Token 和 Session 数据隔离；关闭注册表后拒绝操作 |
| Token 与 Session | 身份、设备、Extra、TTL、手动续期及自省；同账号多 Token 共享数据；撤销 Token 后拒绝读取、写入、删除 Session 数据，存活 Token 不受影响 |
| 下线与查询 | Logout/Kickout/Replace 各 5 种入口，共 15 个范围用例；指定 Token、具体终端、设备与账号的筛选，其他账号保留；终端数量、列表、搜索、遍历提前停止 |
| 角色权限 | 账号和 Token 增删、读取、单项／AND／OR 的布尔和错误接口；授权存在时允许，移除后拒绝 |
| 封禁 | 账号封禁废弃旧凭据且解封不恢复；设备封禁保留凭据、阻止新登录，显式限制按范围生效；服务限制与等级判断不误伤其他业务 |
| 刷新令牌 | 签发、TTL、轮换、旧刷新重放拒绝；主动撤销同时撤销关联访问凭据；自然过期不撤销仍独立有效的访问凭据 |
| OAuth2 | 授权码、S256 PKCE、客户端凭证、密码模式四种入口；错误 secret/password/verifier 拒绝，授权码和刷新重放拒绝；校验、撤销及成功／失败事件结果 |
| Ticket / ShortKey | 定制 TTL、短码确认、绑定不匹配拒绝、只校验不消费、一次性消费、重放拒绝、撤销与状态；失败不得产生消费成功事件 |
| 能力未启用 | OAuth2、Ticket、ShortKey 服务各入口返回 ErrModuleNotEnabled；OAuth2 布尔校验为 false |
| HTTP 分组入口 | Refresh 辅助入口的完整轮换／撤销链；复制请求适配器不改变原请求 Token 与 Manager 绑定 |
| Builder | 输入配置复制、构建后配置隔离、Clone 隔离、nil 恢复默认；非法扩缩容配置、阻塞提交、预分配、非正时间配置被拒绝 |

既有测试仍负责轻量 Token 记录解析、过期与活跃时间、异步续期、Session 懒清理、共享组件关闭、一次性消费并发、存储故障等行为；本轮没有用新增门面测试替代这些底层测试。

`TestManagerRefreshTokenExpires` 原来使用 20 ms 刷新寿命完成整个签发流程，公网多次往返可能在准备阶段耗尽寿命。现在先签发一分钟凭据，再将刷新主键 TTL 缩为 20 ms，等待过期，验证刷新失败、TTL=-2、访问凭据仍有效。内存下单独重复 20 次及完整回归通过；修正后没有重跑 Redis。

## 原始证据与复现

- [完整根模块与 Gin 流程回归](2026-11-26-memory-methods-final.txt)
- [24 个模块的 vet 与其余模块回归](2026-11-26-memory-modules-final.txt)
- [逐函数覆盖原始输出](2026-11-26-memory-methods-functions.txt)
- [逐方法清单 CSV](2026-11-26-memory-methods.csv)：含路径、行号、接收者、方法名、导出标识、语句覆盖率与统计范围；`hit` 为有语句命中，`unhit` 为无命中，`no-statements` 为不可用覆盖率判断的空方法。
- [包语句统计 CSV](2026-11-26-memory-packages.csv)

根模块与 Gin 流程的合并命令如下；独立模块遍历方式见[验收记录](../performance-acceptance_zh.md)。

```powershell
$env:GOCACHE = Join-Path $env:TEMP 'dtoken-nonsso-go-cache'
$env:GOTOOLCHAIN = 'local'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:GOEXPERIMENT = ''
$env:DTOKEN_REDIS_URL = ''
$env:DTOKEN_MANAGER_TEST_STORAGE = ''
$env:DTOKEN_BENCH_STORAGE = 'memory'
$taskCoverage = Join-Path $env:TEMP 'dtoken-memory-methods-final.cover'
go test ./... ./tests/gin_core_flow/... -count=1 '-coverpkg=./...' -covermode=atomic "-coverprofile=$taskCoverage" -timeout=5m
go tool cover "-func=$taskCoverage"
```

`-coverpkg` 每个包后显示的百分比是该测试进程对整个所选包集合的覆盖，不能当作该包自身覆盖率。原始函数输出的总计为 85.8%，还包含 Gin 流程导入的 Redis 适配器与测试应用代码；本报告的 89.0% 根据 profile 按源文件位置合并命中后，仅汇总根模块语句，二者分母不同。

## Redis 的实际进度

先前获准连接专用测试数据库，`storage/redis` 全套测试（含 8 个存储契约子用例）通过，见 [Redis 适配器记录](2026-11-26-redis-storage.txt)。随后引擎回归因公网往返耗时较长，按用户要求停止；运行前段有完成用例，但整套引擎没有完成，不能计为通过。中途观测到上述 20 ms 刷新测试在准备阶段失败，已改测试时序并只在内存验证。

分组进程的非零退出来自主动中断，不单独视为产品失败；已出现的测试失败仍保留在原始记录。没有执行 Redis HTTP 流程、Redis 基准或部署迁移。停止后未继续访问数据库，临时前缀数据可能尚未清理。本报告与仓库文件不保存连接密码。

## 尚未证明的范围

- 语句覆盖不等于分支或业务组合完整性，尚未覆盖的语句仍在逐方法清单中体现；普通 go test 只执行 fuzz 种子，没有持续模糊测试。
- race 仍受本机 CGO/C 编译器缺失限制；普通并发测试通过不能替代 race 检测。
- 四个未命中的私有辅助函数为 `checkLoginAndGetContext`、`getCheckedTokenSession`、`ensureTerminalTokenAlive`、`isTerminalTokenAlive`。当前源码检索未发现从公开路径进入这组辅助函数的调用链；本轮按仓库约定保留，未为提高覆盖率人为调用或删除。
- 真实 Redis 生命周期、网络错误、分布式并发、部署尾延迟和实际迁移仍需相应环境验证。本轮 Memory 方法审核及回归已完成，不宣称框架在任意输入和运行环境下都无错误。
