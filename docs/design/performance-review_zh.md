# DToken-Go 性能优先重构对比报告

日期：2026-11-26。状态：重构前的源码对比与改动建议；后续实现进度见实施方案，本文不作为当前代码的完成清单。

后续设计已收敛到[性能优先重构实施方案](performance-refactor-plan_zh.md)。本文保留对比依据和候选取舍；实施时以方案中明确选择的行为为准。

## 1. 范围、依据与结论

本轮先对比、形成报告，再确定实现方案。允许调整内部实现、公开语义、默认配置和存储布局，不把保留上一轮重构设计作为前提。**SSO 完全不在本轮改造范围内，由独立对话维护。**

对比基线：

- Go：本地 `dtoken-go`，分支 `architecture-refactor`，HEAD `fff8fe8`。审查开始时非 SSO 代码无未提交改动。
- Java：本地 `Sa-Token`，HEAD `832b6517`，重点核对 `StpLogic`、`SaSession`、`sa-token-redis-template`，以及非 SSO 的 OAuth2、Sign、Temp 和事件实现。
- 方法：静态追踪主要公开操作的内部调用链、存储命令、编解码、列表遍历与任务提交。不是对每个框架、生成器、日志后端逐行审计，也不是性能实测。
- 未执行构建、测试、基准测试、生成器或服务。本文中的次数是特定正常路径的源码推导，不是线上请求统计；不提供未经测量的 QPS、延迟降幅或内存收益百分比。

**主要结论：值得重构，但应先收敛语义和数据职责，再优化实现。** 当前额外成本有三类：

1. 可以直接去掉的重复工作，例如 TTL 续期重写整个 Session、锁前锁内重复加载、遍历终端时再次查找同一列表。
2. 由现有功能语义带来的成本，例如每次登录校验自动检查封禁、准确有效终端数量、失败原因保留、刷新令牌轮换。要降低成本，需要明确改变哪些行为。
3. 有必要保留的正确性成本，例如一次性凭据原子消费、Token 归属核对、避免旧维护任务影响新的登录、返回存储故障而非放行。

Java 最值得借鉴的是**基础认证数据轻量化、功能按需执行、仅修改 TTL、在一次操作内复用已取得的数据**。不能把“Java 的基础登录校验通常只读一次 Redis”外推成“Java 的所有操作都只读一次、不反序列化、不需要一致性判断”。

## 2. 操作对比总览

优先级：P0 为重构主线；P1 为直接相关的常见操作；P2 为在主线完成后按测量决定的优化。

| 操作 | 当前 Go | 本地 Java 实现 | 建议 | 优先级 |
| --- | --- | --- | --- | --- |
| `IsLogin / CheckLogin / GetLoginID` | 读取完整 Token 记录并解码；自动检查账号、设备封禁；可选活跃检查与异步维护 | 基础映射是 loginId 或状态标记；活跃超时另查；封禁是独立 API | 建立轻量凭据读取路径，明确基础认证与封禁检查的边界 | P0 |
| Token 详情、设备字段查询 | 多个入口复用完整 Token 解码 | 基础身份映射与会话、终端数据分开取得 | 只要身份不加载详情；完整详情按需读取 | P0 |
| 普通登录 | 读取账号 Session，并发处理前全量清理终端 | 常规登录没有等价的全量过期清扫 | 取消无条件清扫；并发规则需要时再检查 | P0 |
| 共享登录 | 查找有效候选，重写 Token/Session，之后再全量清扫 | 按账号和设备尝试复用；不做同样的全量有效性扫描 | 验证命中的候选即可；复用不默认清扫其他终端 | P0 |
| 拒绝新登录、顶替、数量上限 | 清扫后可能再次查同批 Token；超限循环反复筛选、移除 | 按配置分支执行，拒绝新登录使用有效性短路判断 | 把“不允许并发”“数量上限”“共享”拆为最少必要检查 | P0 |
| 终端列表、遍历 | 默认只读 Session，不检查每个 Token | 同样直接读列表/筛选设备 | 保留轻量语义，文档明确不保证实时有效 | 保留 |
| 有效 Token 列表、在线数 | N 次 Token 校验；内部又遍历 Session 查成员 | 普通终端列表没有等价的完整有效性过滤 | 去掉重复成员查找；精确查询才支付 N 个凭据检查成本 | P1 |
| 自动续期 | 默认开启；任务先判断 TTL，需要维护时再读 Token、封禁、Session；Session 全量写回 | 此处 `autoRenew` 更新活跃时间，不等于滑动延长 Token TTL | 区分活跃更新与 TTL 续期；减少无效任务与 Session 写入 | P0 |
| 手动续期 | 多次 Token 读取，写回 Token 元数据和完整 Session | 校验账号/终端后主要调整 TTL | 复用锁内记录；Session 只改 TTL；超时元数据的语义单独决定 | P1 |
| 单 Token 登出、踢出、顶替 | 先完整校验，再进入写流程重新读取身份和 Session | 沿明确的撤销流程清理或修改映射、更新终端 | 撤销操作不先执行完整“是否可访问”校验 | P1 |
| 批量登出 | 已支持批量删除 Token/活动键、单次保存变更后的 Session | `_removeTerminal` 逐个移除，`SaSession.removeTerminal` 会更新 Session | 保留 Go 的批量优势，不照搬 Java 的逐项持久化 | 保留 |
| 批量踢出、顶替 | 多次读同一 Token；逐个删除活跃键、续期键 | 逐项处理 | 复用每项读取，批量删除附属键；保留状态原因 | P1 |
| 角色、权限检查 | 按账号已支持 Provider 优先；部分 Token 入口先加载 Session | 直接调用 `StpInterface` 提供权限和角色 | Token 入口也优先 Provider；只有返回 nil 才加载 Session | P1 |
| Session 数据修改 | 按 Token 写入有锁前完整校验、锁内再次完整校验；保存时 TTL+SET | Session 对象修改后更新；Redis 更新支持 KEEPTTL | 身份定位后锁内一次完整判断；已有记录更新保留 TTL | P1 |
| 账号、设备、服务封禁 | 身份核对、新旧 key 兼容读取；账号封禁还废弃当前凭据 | 以独立服务/等级封禁检查为主 | 收敛封禁语义与新 key 规范，历史兼容离开常规热路径 | P0/P1 |
| 核心 RefreshToken | 读取、锁内重读、原子消费，再进入完整登录与旧凭据撤销 | 普通 StpLogic 无直接等价物；OAuth2 有独立刷新流程 | 提取内部签发步骤供首次登录和轮换复用，保留一次性消费 | P1 |
| Token 自省 | 加载详情、Session、权限、角色及 TTL | OAuth2 检查使用对象模型，不能当作基础登录校验 | 保留为明确的详情操作，不让基础校验复用它 | P1 |
| HTTP 组合检查 | 已一次复用 Token 和 Session；后续 Context 调用仍可再次检查 | 活跃检查有请求内去重 | 保留组合检查；研究仅请求内结果复用与失效规则 | P1 |
| 事件、后台池 | 事件构造、复制、分发排序；可能双层异步；池失败回退 goroutine | 事件中心基本直接循环调用监听器 | 无可观察效果时快速返回，排序移到配置变更阶段，明确过载处理 | P1 |
| Redis/内存存储 | 有 GetDel/SetNX/批量 Delete；无保留 TTL 更新契约；内存修改有外层互斥 | Redis 更新用 SET XX KEEPTTL；常规生命周期也是多 key 顺序执行 | 增加有明确收益的可选能力；普通 Storage 保留合理回退 | P1 |
| 全局搜索分页 | 全量 SCAN、去重、排序后截取 | Redis DAO 也全量扫描后分页 | 不是登录主路径；游标 API 是否新增由管理需求决定 | P2 |
| Nonce | 已原子 GetAndDelete 消费，无 JSON 解码 | Sign nonce 是检查“是否用过”后记录，语义不同 | 保留原子防重放，不为模仿 Java 降级 | 保留 |
| Ticket / ShortKey | 条件验证、原子读删、再次验证、写回消费状态；部分实例锁串行 | 非 SSO Temp 是临时值映射，不是同一状态机 | 是否保留消费结果记录、锁粒度和条件消费分别评估 | P2 |
| OAuth2 | 对象存储、一次性授权码/刷新处理、客户端与范围校验 | 同样有对象读写、索引、刷新模型转换 | 不套用基础 token→loginId 模型；优化同次流程重复读取 | P2 |
| Auth/注册表/Builder | 固定实例或 sync.Map 选择器；生命周期写锁；装配一次 | 按 loginType 的 ConcurrentHashMap | 保留主要结构，优先处理 I/O；不为了“少一层”再拆门面 | 保留 |

## 3. 基础登录校验：先减少“做什么”，再减少“怎么做”

### 3.1 当前有效 Token 的正常路径

`IsLogin / CheckLogin → checkLoginInternal → checkLoginAndGetTokenInfo → inspectLoginToken`：

1. GET Token key，先识别不存在及踢出、顶替等状态。
2. 通过配置的 Codec 解码整个 `tokenRecord`，包括 `TokenInfo.Extra`。
3. 检查账号封禁；有设备字段时检查设备类型及具体设备封禁。
4. 启用不活跃超时时，读取活跃记录并判断。
5. 按配置提交维护任务；正常有效路径不读取 Session。

以下是**有效 Token、无封禁、未启用不活跃超时、普通无转义账号/设备名、新旧 key 相同、无存储故障**时，仅同步部分的 GET 次数：

| Token 中的设备信息 | Go 同步 GET | Token 记录解码 | Session 读取 |
| --- | ---: | ---: | ---: |
| 无设备 | 2：Token + 账号封禁 | 1 | 0 |
| 仅设备类型 | 3：再加设备类型封禁 | 1 | 0 |
| 设备类型和设备 ID | 4：再加具体设备封禁 | 1 | 0 |

新旧封禁 key 不同且需要兼容查找时，读取还会增加。启用活跃超时也有额外读取。Token 不存在时则会提前返回，不做上述后续工作。

Java 的 `getLoginIdNotHandle` 直接读字符串映射；`getLoginId` 识别状态，并按配置检查活跃超时；`checkDisable` 独立存在。未启用活跃超时的普通模式，基础凭据查询是一条 GET，既不读取账号 Session，也不解码完整 TokenInfo。

因此，**仅把 JSON 换成另一种序列化方式，不能把我们的 2～4 次读取变成一次。**

### 3.2 推荐目标

- 基础凭据检查只判断 Token 是否存在、是否处于可用状态，并取得必要身份。
- 不读取完整 TokenInfo、Extra 或 Session；不为了查有效期额外执行 TTL。
- 存储 TTL 是绝对到期的主要依据；TTL 缺失/状态失效不通过认证。自定义 Storage 必须遵守过期后 Get 返回不存在的契约。
- 活跃超时作为启用后才执行的同步检查。已超时的当前请求不能先通过，再异步判断。
- 将封禁检查从基础凭据检查中分离，是本报告推荐的性能优先方向；HTTP 组合检查按声明执行相应策略。
- 详情查询、自省、精确终端管理按需取得额外数据。

**语义变化必须显式发布：** 如果 `CheckLogin` 不再自动检查所有封禁，它只表示凭据有效，不再同时表示业务访问未受限制。登录签发仍需拒绝被禁止登录的账号/设备。路由声明、现有中间件默认值、错误映射、示例与迁移文档必须一起调整，不能静默少检查一项。

另一个选择是继续保持“登录检查包含封禁”，使用多 key 批量读取降低往返。这可以保留语义，但不能等同于 Java 的单 key 基础检查，也不会消除服务端读多个 key 的成本。

### 3.3 数据布局建议及取舍

推荐把认证所需的轻量信息与低频详情分离，但不在未确定功能边界前固定字段和编码格式：

| 数据 | 职责 | 热路径是否读取 |
| --- | --- | --- |
| Token 主记录 | 状态、身份，以及实际需要的最小生命周期信息 | 是 |
| Token 详情（确有需求时） | 创建信息、Extra 等完整查询需要的数据 | 否 |
| 账号 Session | 账号数据、终端索引、没有 Provider 时的权限角色 | 否 |
| 可选活跃记录 | 启用不活跃超时后的有效性依据 | 按需 |
| 刷新凭据记录 | 独立生命周期、轮换和撤销信息 | 仅刷新相关操作 |

这是 Go 的候选设计，**不是 Java 固定采用的三 key 布局**。Java 的 Token-Session 是可选的 Token 范围数据容器，不等于完整 TokenInfo；普通 StpLogic 的 Extra API 还与 JWT 插件有关。

推荐先确定是否需要在主记录保留设备、活跃超时覆盖值和生命周期 ID。如果只保存 loginId，却又让每次设备限制检查或异步维护读取详情，可能把开销挪到另一处。轻量记录仍需要解析必要字段，“不反 JSON”不等于完全没有解析。

若拆出详情 key，需要同时设计签发顺序、共同续期、注销清理、无期限记录回收、Token 同值复用的归属绑定，以及详情缺失时各 API 的行为。基础凭据成功与详情 API 成功不应隐式捆绑。不能只优化 CheckLogin，然后让续期遗漏详情 TTL。

## 4. Session 终端：取消无条件全量清扫

### 4.1 当前实际成本

`handleConcurrency` 开始就调用 `cleanExpiredTerminals`；共享登录返回前也清理；刷新轮换有相关清理路径。

`cleanExpiredTerminals` 对账号的每个终端读取 Token，正常记录需要解码；活跃超时启用时另读活跃 key；一些失效分支还会重读、清理附属 key。之后并发判断可能再次访问已查过的 Token。

所以一次普通登录会受该账号历史终端数量影响。共享命中也不等于只处理共享候选。它们是在登录调用返回前执行的同步工作。

Java 普通账号终端列表不自动清扫自然过期项，普通列表查询也不逐个检查。它以更少读取换取索引中暂时保留过期项。Java 的 Temp/OAuth2 索引另有按到期时间整理的实现，不能与账号终端列表混为一谈。

### 4.2 推荐处理规则

| 场景 | 建议 |
| --- | --- |
| 无上限并发登录 | 读取 Session、追加本次终端；不逐个读取历史 Token |
| 共享登录 | 从匹配范围寻找有效候选，命中后停止；不清扫其他设备 |
| 拒绝新登录 | 只查限制范围，找到一个有效占位即停止 |
| 顶替旧终端 | 直接进入相应范围的撤销流程，不先对整个账号清扫一遍 |
| 有限登录数量 | 未达到候选条目阈值时不全查；达到阈值才核实影响决策的记录并清理 |
| 普通列表/遍历 | 只返回索引，不验证也不写回 |
| 精确有效列表/在线数 | 明确逐个或批量检查，结果过滤与持久化清理分开 |
| 主动退出/踢出 | 更新对应索引，不推迟凭据失效本身 |

推荐保留“有效登录数量上限”的语义，不直接把残留记录当成在线登录。否则虽然省读，却会造成有效终端被提前踢出或新登录被错误拒绝。若产品选择与记录数量一致的上限，应明确改名/改文档，作为另一种策略。

取消默认清扫也有代价：账号 Session 被长期续期、允许无限终端时，索引可能持续增长。需要明确历史保留策略，优先考虑在管理操作、数量阈值处整理，或显式维护接口；不默认启动全库扫描任务。若要增加索引到期时间，必须评估每次续期维护这个时间的写放大，不能认为它是免费优化。

### 4.3 终端精确查询中的重复遍历

`filterTokens/countAliveTokens → checkTerminalEntryAlive → checkTerminalTokenAliveWithOptions` 已有终端项和 Session，后者仍调用 `sessionHasTerminalToken` 遍历 Session 查成员。

对 N 个有效终端逐个查询，成员检查最坏可累积为 O(N²) 的本地遍历，另有 O(N) 个 Token 读取。这与 Redis 延迟是两种不同的成本。

建议让已核对身份的终端项直接进入后续检查，避免回头扫描；保留现有聚合查询内的账号/设备封禁缓存。需要任意 Token 查成员时再建立一次索引或单次查找，不为每个小列表固定分配 map。

## 5. 登录签发、共享和续期

### 5.1 共享登录不重写不变数据

当前 `getTokenAndShare` 找到可用 Token 后调用 `saveSessionWithMinTTL` 保存整个 Session，并写回整个 Token 记录，然后外层清扫终端。

建议：共享候选有效且数据未变时，仅执行约定的 TTL/活跃维护。不因“复用”而重复编解码、写回详情。是否共享登录必定重置 Token TTL也应明确；Java 复用行为不能直接假定与我们一致。

### 5.2 续期与保存对象分离

当前自动续期通过 `renewTokenAndSessionLocked` 调整 Token TTL，但仍调用 `saveSessionWithMinTTL` 序列化并 SET 整个 Session。手动续期也保存整个 Session。

Java `renewTimeout` 仍会取得 Session 并校验终端，并非零读取；但持久化侧主要使用 `updateTimeout/updateMinTimeout`，不会仅为延长生命周期而重写所有会话数据。

建议分为：

1. **内容修改**：序列化一次，更新已有记录并保留 TTL。
2. **生命周期修改**：仅调整 TTL；账号 Session 只延长、不缩短，也不把无限期改成有限期。

`expireTokenIfLimited` 当前是 Exists 再 Expire；Storage.Expire 已要求缺失时返回错误，Redis PEXPIRE 和内存实现也提供该判断。可以用一次 Expire 的结果判定缺失并保持公开错误语义，不需要先 Exists。

手动续期当前还更新 `TokenInfo.Timeout`，这关系到下一次自动续期的周期。若保留该语义，仍需更新相应轻量元数据；不能一概用 EXPIRE 代替而让后续续期回到旧周期。无限期切换也必须单独定义：当前 `Expire <= 0` 表示删除，不能当作 Redis PERSIST 使用。

### 5.3 异步并不等于节省总成本

当前默认开启 AutoRenew。`prepareLoginMaintenance` 只按配置接收任务，`runLoginMaintenance` 才查 TTL。任务执行完就释放合并状态，因此合并的是同时在途的工作，不是给连续请求提供长期节流。

需要关注：

- 远未到续期阈值的请求，仍可能反复产生任务、锁和 TTL 查询。
- 只更新活跃时间的任务，也会读取账号 Session。
- 整个 `tokenRecord` 被用于维护上下文，Extra 等低频数据可能随任务滞留。
- 维护任务在判断是否需要工作之前，已经取得账号写锁。

建议：区分“是否需要调度”“何时读 TTL”“如何续期”三个阶段；只携带身份与生命周期字段。纯活跃更新不依赖 Session；只有实际需要修改账号会话时进入相应写流程。评估有界、短时的维护节流，必须说明对活跃时间精度、多实例行为的影响。

不能为减少读取而默认缓存登录成功结果跨请求使用；否则撤销可能不立即生效。维护调度的节流与认证结果缓存必须分开。

## 6. 撤销、封禁与会话写入

### 6.1 撤销不应先要求“当前可访问”

当前按 Token Logout/Kickout/Replace 会先读取 Token，再执行包含 Session/封禁/活跃状态的校验，然后进入撤销流程再次核对绑定、读取 Session。批量踢人还可能在归属检查后再次读取同一记录检查结构性有效性。

建议流程：**定位凭据身份 → 进入必要的写同步范围 → 核对当前归属 → 撤销凭据与关联刷新令牌 → 更新一次 Session → 发布一次对应事件。**

封禁或不活跃不应阻止撤销，也不需要先模拟一次业务访问。保留“旧快照不能撤销被复用 Token 的新生命周期”的约束；减少同次流程重复读取，而不是删除所有归属检查。

当前批量 Logout 已批量删除 Token 和活动键，并只保存一次变更后的 Session，值得保留。`processTerminalsIf` 中续期 key、活跃 key 的逐项 Delete 可归并；需要不同状态值/TTL的更新可以评估可选批处理，pipeline 只减少往返，不等于事务。

### 6.2 封禁语义应先统一

Go 的账号封禁会使关联凭据失效，解封不恢复已废弃凭据；设备封禁与登录占位还存在可解除限制的处理。Java 的服务/等级封禁是独立检查，与踢出并不等价。

推荐明确两个动作：

- 禁止某类访问/登录：独立限制，在要求该规则的入口检查。
- 让已有凭据失效：通过撤销/踢出明确执行。

若保留“封禁同时撤销”的便捷语义，应在写路径执行，不让每次凭据校验承担所有管理职责。但仅依赖终端索引撤销也有条件：索引必须完整，遗漏的有效 Token 不能绕过禁令；否则需要额外的账号状态/版本检查，不能再承诺单 key 校验。

新布局应使用无歧义的命名空间和身份编码。当前新旧封禁 key 兼容、身份核对有历史原因；若这轮允许存储格式变化，建议版本化迁移，把旧键兼容从长期热路径移走，而不是不安全地删除归属判断。

### 6.3 会话数据写入只做必要复核

`SetSessionValueByToken/DeleteSessionValueByToken` 当前先取 Token 定位账号，再做一次完整锁前校验，锁内再次完整校验。

正常不超时路径，仅定位和校验就涉及 **3 次 Token 读取、2 次 Session 读取**，另有两轮封禁/可选活跃检查；最后 Session 更新还要 TTL+SET。

建议定位身份后进入账号写锁，在锁内做一次所需校验，直接使用该 Session 修改并保存。锁前的身份快照与锁内生命周期复核仍保留；删除的是重复加载整份上下文。权限/角色修改也沿同一原则核对。

## 7. 权限、HTTP 请求与自省

### 7.1 Provider 命中时不要先读 Session

按账号的 `loadPermissionsByLoginID/loadRolesByLoginID` 已优先查询 Provider；而部分按 Token 入口和 `CheckTokenAccess` 先取得 Session，再调用 Provider。

Java 的 `StpInterface` 可直接提供权限/角色，不要求先加载账号 Session。建议把 Go 的优先顺序统一为：

1. 校验轻量凭据，建立 AccessSubject。
2. 请求确实需要的 Provider 数据；非 nil 结果直接使用。
3. 只有某组数据返回 nil、需要本地回退时才加载 Session；权限和角色共享同一次加载。

必须保留 nil 表示回退、非 nil 空列表表示明确无权限的区别。不默认跨请求缓存 Provider 结果，否则权限回收时效会改变。

权限模式匹配和角色匹配仍应支持现有 Strategy。不能无条件改成 map 查找而绕过自定义匹配器；小集合使用线性查找可能更合适，通配符优化以测量为准。

### 7.2 请求内复用，边界明确

`httpauth.Check` 已通过 `CheckTokenAccess` 合并登录、权限、角色检查，保留这个方向。独立 Context 方法多次调用，仍可能重复提取 Token、查询凭据。

Java 的 `checkActiveTimeoutByConfig` 使用请求 storage 避免同一请求重复执行活跃检查。建议优先复用 HTTP 中间件已取得的结果，缓存只属于本请求，按 Manager 身份、Token、检查种类区分。

同一请求发生登录、登出、换 Token、换 AuthType、权限或封禁变更时必须使相关结果失效。不要把“已检查一次”作为全局布尔值；也不要把所有公共 API 的跨调用缓存当作默认行为。

### 7.3 自省保留详情语义

`IntrospectToken` 返回身份、设备、TTL、权限角色、Extra，合理地比 IsLogin 昂贵。建议与轻量凭据校验明确分层、Provider 命中时延迟 Session 读取，但不把丰富返回值伪装成单 GET 操作。

## 8. 存储、生命周期和事件

### 8.1 最值得补充的存储能力

当前 `saveToStorage` 在未显式指定过期时间时先 TTL、再 SET；Java Redis DAO 的更新使用 `SET XX KEEPTTL`。

推荐引入可选的“更新已有值并保留 TTL”能力：Redis 可用一条命令，内存可在已有原子范围内完成，普通自定义 Storage 保留合理的 TTL+SET 回退。结果要能区分 key 不存在；不要更新时重建已过期凭据。

第二优先才是批量读取/写入能力，主要用于用户明确要求精确的终端查询或批量状态变更。批量读只能减少往返，不能消除返回数据量和解码；Redis Cluster 跨 slot 也不是任意 MGET 都可用。不要为抽象完整性强制所有存储实现事务、Lua 或批处理。

Java 常规登录、多 key 退出也是顺序 DAO 调用，没有统一跨 key 事务和自动回滚。Go 可保持正常生命周期正确、局部原子操作优先的策略，不建设数据库式补偿框架。

### 8.2 锁只保留在需要的范围

当前按账号的锁保护同一 Manager 内的写操作，不能当作跨进程锁。它避免本地正常并发写覆盖，有保留价值。优先缩短锁内 I/O、去掉无须写入时的锁，而不是为了追求“无锁”移除所有同步。

内存 Storage 的外层修改锁保证组合操作与普通写入协调，但同时串行不同 key 的修改。先减少上层写放大，再测不同账号并发写入；如确认存在瓶颈，再考虑分片，而不是直接删除保证 GetAndDelete/SetIfAbsent 语义的锁。

Auth 注册表读不持有 `globalManagerMu`；该锁主要协调注册/替换/关闭。Builder 在初始化阶段执行。相比远程存储往返，这些不是本轮的优先重构对象。

### 8.3 事件和过载必须算入成本

当前引擎在确认是否存在实际监听需求之前就可能构造事件、复制 Extra、提交任务。事件管理器每次分发合并专属和全局监听器并排序；异步事件再遇到异步监听器会形成两次调度。

建议：

- 无监听器、无过滤器、无统计与日志观察需求时，构造载荷前快速返回。不能只看某类监听器数量，遗漏 EventAll、统计或过滤器语义。
- 把稳定排序/合并工作移到注册或配置变化阶段，分发读取稳定快照；保留监听器变更和回调安全语义。
- 明确异步只在哪一层发生，减少重复调度。Java 的同步循环简单，但直接改成同步会让慢监听器拖长业务请求，不能称为普遍加速。
- `submitAsync` 在池不存在或拒绝时启动 goroutine，单 Token 合并不能限制大量不同 Token 的总任务数。应区分可合并/可跳过的维护任务与不能默默丢弃的事件，定义有界的过载行为。

限流、丢弃、阻塞、同步执行都有语义代价，需要在实现前确定；不能一边承诺“校验立即返回”，一边用无界 goroutine 把压力转移出去。

## 9. RefreshToken 与独立能力

### 9.1 核心刷新令牌

当前刷新先定位账号、锁内重读、检查访问凭据、消费刷新凭据，再通过完整登录路径签发，最后撤销旧访问凭据。复用了行为，但也重复进入登录清扫、并发控制、持久化和撤销调用链。

建议把“解析登录策略”和“内部签发凭据/更新一个终端”分开，首次登录与轮换使用共同的内部步骤，不让轮换绕经多层公开操作。轮换是否占用一个原有名额需要保持明确，不将其简单视为另一台设备登录。

保留以下约束：访问 Token 自然过期不等于刷新凭据过期；退出/撤销后不能刷新复活；同一个一次性刷新凭据不能成功消费两次；旧访问 Token 值被复用后不能误删新登录。

Java 普通 StpLogic 没有与这套核心轮换直接等价的流程。OAuth2 的刷新生成新访问模型，是否生成新刷新令牌由客户端配置控制。不能拿不同的刷新/撤销语义直接比较总命令数。

### 9.2 Nonce

Go 已用 `AtomicStorage.GetAndDelete` 一次完成消费，不经过 JSON；这符合目标，应保留。Java Sign 是验证外部 nonce 未被使用后登记，Go 是先签发再消费，不能照搬其 GET+SET 作为同等防重放保证。

### 9.3 Ticket / ShortKey

当前原子存储路径仍会先 Validate，再 GetAndDelete 并验证实际取出的对象，最后写回消费状态。它承担约束检查、明确的 consumed 状态和并发消费语义。

可评估两项独立取舍：

- 若不需要长期查询“已消费”状态，删除即可，省掉状态写回；但原有 Status/错误原因会改变。
- 缩小 ShortKey 实例级锁范围或按 key 协调，避免不同短 key 的确认、消费相互等待；不能仅因支持 GetAndDelete 就去掉与 Confirm/Revoke 的协调。

直接改成“先 GETDEL 再检查所有约束”可能让错误的调用者耗尽别人的有效凭据。保留先验证，或提供明确的条件原子消费能力，是安全语义取舍，不应只按少一次读取决定。

Java 非 SSO Temp 是值映射和可选索引，不是 Ticket/ShortKey 的完整状态机。本报告未找到可直接替代这两个独立 Go 模块的同名、同语义非 SSO 实现；这部分属于本地设计优化，而不是照搬 Java。

### 9.4 OAuth2

Go `ValidateAccessTokenAndGetInfo` 与 Java `SaOAuth2Dao.getAccessToken` 都读取对象信息。OAuth2 需要 client、scope、过期和授权关系，不能只看 key 存在就视为授权完成。

重点评估同次授权/刷新过程中客户端、凭据与 scope 的重复读取，复用已校验的数据；保留 PKCE、redirect URI、client 归属和一次性授权码消费。Go 目前把同一 AccessToken 载荷保存到访问/刷新两个 key，Java 则有各自模型，可借鉴职责拆分，但收益应与索引、撤销代价一起比较。

OAuth2、Ticket、ShortKey、Nonce 继续作为独立服务组合，不因本轮优化并入认证引擎。

## 10. 建议明确采用与保留讨论的决定

| 决定 | 推荐方向 | 代价或需要同步变更的部分 |
| --- | --- | --- |
| 基础登录校验是否包含封禁 | 分离；HTTP 按显式组合策略执行 | API 语义、路由默认值、错误和迁移文档 |
| 普通登录是否整理所有过期终端 | 不整理；策略决策/管理场景按需处理 | 索引允许过期项，需明确增长与整理策略 |
| 登录数量上限怎么算 | 推荐继续按有效占位；达到阈值再核实 | 部分边界登录仍有读取，不能做到所有登录 O(1) |
| Token 是否能用相同值重新分配 | 默认生成值不复用；保留显式复用时的生命周期绑定 | 若取消复用支持，需要公开声明，不能只删保护代码 |
| 原始存储格式是否兼容 | 新版本布局；迁移兼容尽量不驻留热路径 | 旧 Token 的迁移/失效方案、灰度与回滚策略 |
| AutoRenew 默认行为 | 区分绝对 TTL 滑动续期与活跃更新，重新评估默认开启 | 默认会话有效期行为会改变，不能仅改注释 |
| 异步过载行为 | 有界；维护和事件分别定义 | 续期精度、事件可靠性和响应延迟取舍 |
| 详情/索引缺失是否使凭据无效 | 基础凭据以主记录为准；详情 API 独立报错 | 不再让所有 API 共享同一“全量校验”入口 |
| 一次性凭据是否保留消费状态 | 按业务需求决定，不默认一刀切 | Status API、审计/排错需求与写入成本 |

允许调整任何逻辑意味着可以重新选择这些语义，不意味着在报告阶段已经批准每一种破坏性迁移。实现阶段应以最终选定的一组一致行为为准，避免一半 Java 语义、一半旧 Go 语义。

## 11. 实施顺序与验收标准

### 第一阶段：确定语义和数据职责

确定基础认证、封禁、活跃超时、自动续期、终端数量、显式 Token 复用的契约。列出每个字段被哪些操作消费，选定主记录与详情结构，再确定数据迁移方式。

验收：能用同一套规则解释登录、校验、退出、封禁、续期和刷新；明确哪些 API 只读凭据、哪些会读取 Session/详情。

### 第二阶段：实现 P0 热路径

优先轻量凭据校验、取消无条件清扫、共享候选短路、后台维护调度。签发、注销、续期和详情读取必须同时适配新记录。

验收：

- 未启用附加检查的有效 Token 基础校验：1 次主记录 GET，0 次完整详情/Session 解码，0 次同步写入。
- 无效或已标记失效的 Token：一次读取后立即返回，不安排有效登录维护。
- N 个历史终端的普通无限制登录/共享命中：不发生 N 次历史 Token 读取；仍需读取和处理 Session，其字节量和序列化成本仍随列表大小增长，不能宣称整体 O(1)。
- 终端精确查询不再有逐项回扫 Session 的 O(N²) 成员检查。
- 统计同步与异步的总命令数，不能只优化返回时间。

### 第三阶段：收敛 P1 操作

TTL 更新独立化、会话写入复用、Provider 延迟回退、撤销批处理、刷新内部签发、事件快速返回及过载规则。

验收：未修改内容的续期不序列化和 SET 整个 Session；Provider 完全命中时不读 Session；会话写入不做锁前锁内两轮完整上下文加载；批量操作对 Session 最多一次最终写回或删除。

### 第四阶段：按数据优化 P2

独立能力、精确管理查询批处理、存储锁竞争和搜索分页在前述基线稳定后评估。公开游标分页、分片锁或缓存只有在需求/测量支持时再增加。

### 测量与回归矩阵（待用户允许运行后执行）

| 维度 | 用例 |
| --- | --- |
| 存储 | 内存、真实 Redis；最小 Storage 回退、可选能力路径 |
| 校验 | 有效、缺失、踢出、顶替；活跃超时开/关；封禁组合开/关 |
| 载荷 | 无 Extra、1 KiB、64 KiB；小 Session 与大 Session |
| 终端规模 | 单账号 1、10、100、1000 个终端；0%、50%、100% 过期残留 |
| 登录策略 | 无上限、有限上限、拒绝新登录、共享命中/未命中、设备范围 |
| 权限 | 无 Provider、完全命中、nil 回退、非 nil 空列表、组合权限角色 |
| 写操作 | Session 修改、手动续期、自动续期、退出、批量踢人、刷新轮换 |
| 并发 | 同 Token、多 Token 同账号、多账号；慢 Redis、池饱和 |
| 指标 | GET/TTL/SET/EXPIRE/DEL/原子操作次数、传输字节、编解码次数、ns/op、B/op、allocs/op、p50/p95/p99、后台任务数与锁等待 |

应先记录旧实现基线，再在同等语义/同等配置下比较新实现；行为有意变化的场景单独列出，不能把少执行的认证规则包装成纯实现加速。

必须回归：即时撤销、不活跃边界、临时限制解除、刷新凭据不复活、Token 同值新生命周期、无限期/缩短 TTL、错误传播、跨认证体系隔离、请求内身份变化、关闭等待与事件次数。新增可选存储能力要覆盖语义一致的普通存储回退。

## 12. 源码核对索引

下面的行号对应本报告基线，后续改造可能变化。

| 主题 | Go 入口 | Java 对照 |
| --- | --- | --- |
| 凭据读取 | [internal_session.go:80](G:/code/go/my_project/dtoken-go/internal/engine/internal_session.go:80)、[校验:328](G:/code/go/my_project/dtoken-go/internal/engine/internal_session.go:328) | [StpLogic.getLoginId:1126](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:1126)、[getLoginIdNotHandle:1302](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:1302) |
| 登录与共享 | [auth_login.go:145](G:/code/go/my_project/dtoken-go/internal/engine/auth_login.go:145)、[getTokenAndShare:208](G:/code/go/my_project/dtoken-go/internal/engine/internal_concurrency.go:208) | [createLoginSession:488](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:488)、[distUsableToken:542](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:542) |
| 清理与查询 | [cleanExpiredTerminals:556](G:/code/go/my_project/dtoken-go/internal/engine/internal_session.go:556)、[终端列表:281](G:/code/go/my_project/dtoken-go/internal/engine/auth_session.go:281)、[逐项校验:191](G:/code/go/my_project/dtoken-go/internal/engine/internal_token.go:191) | [终端列表:2406](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:2406)、[SaSession 筛选:283](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/session/SaSession.java:283) |
| 续期与后台任务 | [RenewTimeout:653](G:/code/go/my_project/dtoken-go/internal/engine/auth_login.go:653)、[自动续期:784](G:/code/go/my_project/dtoken-go/internal/engine/auth_login.go:784)、[调度:388](G:/code/go/my_project/dtoken-go/internal/engine/internal_session.go:388) | [renewTimeout:2012](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:2012)、[请求内活跃检查:1760](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:1760) |
| 撤销与封禁 | [logoutTerminalsIf:460](G:/code/go/my_project/dtoken-go/internal/engine/auth_logout.go:460)、[processTerminalsIf:706](G:/code/go/my_project/dtoken-go/internal/engine/auth_logout.go:706)、[封禁检查:1143](G:/code/go/my_project/dtoken-go/internal/engine/auth_disable.go:1143) | [_logout:968](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:968)、[checkDisable:2614](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:2614) |
| 角色权限 | [CheckTokenAccess:12](G:/code/go/my_project/dtoken-go/internal/engine/internal_access_check.go:12)、[Provider 回退:15](G:/code/go/my_project/dtoken-go/internal/engine/auth_access.go:15) | [getRoleList:2069](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:2069)、[getPermissionList:2207](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/stp/StpLogic.java:2207) |
| 数据更新与存储 | [会话写入:101](G:/code/go/my_project/dtoken-go/internal/engine/auth_session_data.go:101)、[保存:129](G:/code/go/my_project/dtoken-go/internal/engine/internal_storage.go:129)、[Expire:283](G:/code/go/my_project/dtoken-go/storage/redis/storage.go:283) | [SaSession.update:411](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/session/SaSession.java:411)、[KEEPTTL:175](G:/code/java/git_project/Sa-Token/sa-token-plugin/sa-token-redis-template/src/main/java/cn/dev33/satoken/dao/SaTokenDaoForRedisTemplate.java:175) |
| 事件与池 | [事件提交:19](G:/code/go/my_project/dtoken-go/internal/engine/internal_events.go:19)、[分发快照:464](G:/code/go/my_project/dtoken-go/event/listener.go:464)、[异步接收:158](G:/code/go/my_project/dtoken-go/internal/engine/internal_lifecycle.go:158) | [SaTokenEventCenter](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/listener/SaTokenEventCenter.java) |
| 刷新与 OAuth2 | [核心刷新:163](G:/code/go/my_project/dtoken-go/internal/engine/auth_refresh.go:163)、[OAuth2:254](G:/code/go/my_project/dtoken-go/oauth2/oauth2.go:254) | [OAuth2 刷新:122](G:/code/java/git_project/Sa-Token/sa-token-plugin/sa-token-oauth2/src/main/java/cn/dev33/satoken/oauth2/data/generate/SaOAuth2DataGenerateDefaultImpl.java:122)、[对象读取:352](G:/code/java/git_project/Sa-Token/sa-token-plugin/sa-token-oauth2/src/main/java/cn/dev33/satoken/oauth2/dao/SaOAuth2Dao.java:352) |
| 独立凭据 | [Nonce:145](G:/code/go/my_project/dtoken-go/nonce/nonce.go:145)、[Ticket:171](G:/code/go/my_project/dtoken-go/ticket/ticket.go:171)、[ShortKey:249](G:/code/go/my_project/dtoken-go/shortkey/shortkey.go:249) | [Sign nonce:267](G:/code/java/git_project/Sa-Token/sa-token-plugin/sa-token-sign/src/main/java/cn/dev33/satoken/sign/template/SaSignTemplate.java:267)、[Temp 索引:254](G:/code/java/git_project/Sa-Token/sa-token-core/src/main/java/cn/dev33/satoken/temp/SaTempTemplate.java:254) |

报告编写当时的交付仅为源码对比与建议，尚未修改生产代码。此后已按[实施方案](performance-refactor-plan_zh.md)推进重构；当前实现进度及待验收事项以该方案为准，本文保留历史对比基线。SSO 始终不在此次性能重构范围内。
