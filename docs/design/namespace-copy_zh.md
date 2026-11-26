# 离线命名空间复制

[English](namespace-copy.md) | 中文文档

`migration` 包提供同一 Storage 内、同一 AuthType 的显式离线迁移。当前运行时统一访问 `{编码后的 KeyPrefix}{编码后的 AuthType}v2:`，不会回退到旧的无版本空间。升级部署应使用 `PrepareAuthUpgrade`，将旧凭据、封禁及同空间的独立能力迁入实际版本化目标。工具不会启动应用或删除源数据。SSO 不在范围内。

| 预检入口 | 目标与用途 |
| --- | --- |
| `Prepare` | 按无版本边界原样复制，不转换旧格式 |
| `PrepareTokenUpgrade` | 在无版本目标暂存 Token、Extra 和刷新绑定转换；封禁原样复制 |
| `PrepareAuthUpgrade` | 从无版本源迁入当前 `v2:` 目标，同时规范化封禁；用于当前运行时升级 |

## 使用条件

- Storage 必须支持 `contract.ScannerStorage`。源和目标前缀必须显式提供；AuthType 省略时使用 Builder 默认体系，规范化规则与 Builder 相同。`PrepareAuthUpgrade` 允许两个前缀相同，因为目标多出固定 `v2:` 段；原样复制及 Token 暂存入口仍要求不重叠的空间。
- 从 `Prepare` 之前直到切换结束，停止所有实例的请求写入、自动续期、活跃更新、定时维护，以及独立能力服务的写入。工具不会检测源数据变化，也不提供在线增量迁移。
- 关闭 Auth 可能连带关闭其拥有的存储连接。先排空请求及后台任务，再使用仍有效的、由迁移程序管理的存储连接；不要把已关闭的连接传给工具。工具借用 Storage，不负责关闭。
- 当前实现枚举后端全部键，再按字面命名空间筛选；预检将所选空间的原始值保存在内存中。键列表占用与整个后端键数相关，快照占用与所选数据量相关。应先评估维护窗口和内存预算；它不是流式或分页的大规模迁移器。

## 预检与执行

在停写维护程序中调用，`store` 为该程序已连接的 Storage，`sourceCodec` 必须与源数据一致：

```go
import (
    "context"
    "fmt"

    "github.com/Zany2/dtoken-go/contract"
    "github.com/Zany2/dtoken-go/migration"
)

func upgradeNamespace(ctx context.Context, store contract.Storage, sourceCodec contract.Codec) error {
    plan, err := migration.PrepareAuthUpgrade(ctx, store, migration.Options{
        AuthType:     "user",
        SourcePrefix: "dtoken:",
        TargetPrefix: "dtoken-next:",
    }, sourceCodec)
    if err != nil {
        return err
    }
    fmt.Printf("预检：%+v\n", plan.Summary())

    // Execute only after reviewing the summary during the stopped-writer window. 在停写窗口核对预检摘要后执行。
    result, err := plan.Apply(ctx)
    fmt.Printf("复制：%+v\n", result)
    return err
}
```

所有预检入口均不写入数据；返回的 Plan 持有快照，不应复制或序列化。`Summary` 返回实际源/目标命名空间、计划记录数、永久与有限期记录数，以及预检期间跳过的消失或过期记录数。转换入口执行下文所列的内容校验；原样复制入口不验证载荷内容。

上例源空间为 `dtoken:user:`，目标为 `dtoken-next:user:v2:`。也可以将 `TargetPrefix` 设为 `dtoken:`，迁入 `dtoken:user:v2:`，此时应用无需更改原 KeyPrefix。不要把版本段手工拼到 KeyPrefix 或 AuthType 中。预检拒绝实际目标已有数据，并排除源位置已有的 `v2:` 子树；不会把当前版本数据再次嵌套迁入新目标。

`Apply` 在执行前再次确认整个目标空间为空。支持 `AtomicStorage` 时，每个键使用 `SetIfAbsent`；普通存储使用 Get/Set，仍有竞态，因此停写要求不可省略。单键原子操作不使整个命名空间复制变成事务。

## 数据与有效期

`Prepare` 原始复制覆盖所选命名空间下的所有键，包括 Token、Extra、Session、活跃/续期状态、刷新凭据及反向映射、封禁、同空间的独立能力和应用自定义后缀。原始字符串、字节及基础数值/布尔值原样复制，字节数组会分离缓冲区；其他类型在预检阶段报错。此入口不解析或补造 AccessID，不改写值中的路径或前缀引用，也不修复损坏记录或历史封禁键的歧义。自定义负载内嵌旧前缀时，需要应用另外转换。

每个有限期记录在读取 TTL 前记录时间基准；执行时扣除已耗费时间，已过期记录跳过，绝不以零 TTL 写成永久。只有 `TTLNoExpire` 对应永久记录，`TTLNotFound` 与零 TTL 跳过，其他非法负 TTL 报错。相对 TTL 仍受后端精度、网络与写入耗时影响，不保证跨键的严格绝对到期时刻一致。

## 转换 Token 存储格式

仅需在无版本目标暂存凭据格式转换、尚不进行运行时升级时，可使用以下入口：

```go
plan, err := migration.PrepareTokenUpgrade(ctx, store, migration.Options{
    AuthType:     "user",
    SourcePrefix: "dtoken:",
    TargetPrefix: "dtoken-next:",
}, sourceCodec)
```

`sourceCodec` 必须与源数据使用的 Codec 一致；此入口不猜测 JSON、Base64 或其他格式。成功后仍需检查 `err`、审阅 `plan.Summary()`，再显式调用 `Apply`。暂存结果位于无版本空间，不能直接作为新运行时目标；实际升级应使用 `PrepareAuthUpgrade`。以下凭据转换规则由两个转换入口共用，目标应用继续使用相同 Codec 读取 Session、Extra、刷新凭据和独立能力数据。

- Token 主记录转换为 `dtk:2:`。已有 AccessID、TerminalIndex、Revoked 和逻辑下线状态保持不变；没有 AccessID 的凭据分配新标识。
- 内联非 nil Extra 分离到 `token-extra:<accessID>`，继承对应 Token 的截止时间；源格式仍可区分的空 map 与 nil 保持区别。若历史编码已经省略空 map，则无法恢复该区别。已有独立 Extra 会检查存在性和可解码性。
- 使用引擎原有的 `refreshMatchesAccessRecord` 规则核对转换前的绑定，匹配的匿名刷新记录使用 Token 的新 AccessID；无匹配 Token 的匿名刷新记录使用独立标识，避免附着到已复用的 Token。刷新记录保留自身 TTL、Extra 和 TerminalExtra；Token 自然到期后仍可按原规则刷新。
- 预检拒绝跨 AuthType 记录、重复 Token 生命周期标识、非法主记录、缺失/损坏的必要 Extra，以及反向映射与仍存在的刷新记录不一致等情况。匿名旧记录的匹配仍受历史字段精度限制，无法重建源数据未记录的真实生命周期。
- Session 仅检查可解码性、账号归属和基本字段；不重建缺失 Session，不整理历史终端，也不改变终端序号。封禁、独立能力及自定义记录继续原样复制，本入口不宣称完成整个存储空间的语义校验。

摘要额外提供 `UpgradedTokens`、`UpdatedRefreshTokens`、`AddedTokenExtras`；`Records`、`Permanent` 和 `Expiring` 是计划写入的目标记录数，包含新增 Extra。转换期间不写存储，任何转换错误都不会返回可执行 Plan；转换耗时计入原快照的 TTL 扣减。已符合当前格式且身份完整的凭据不会重复改写快照载荷。

## 转换封禁与归属

需要同时迁移凭据和封禁时，使用以下预检入口，再沿用相同的摘要审阅与 `Apply` 流程：

```go
plan, err := migration.PrepareAuthUpgrade(ctx, store, migration.Options{
    AuthType:     "user",
    SourcePrefix: "dtoken:",
    TargetPrefix: "dtoken-next:",
}, sourceCodec)
```

此入口在同一份快照内处理账号、服务、设备类型和具体设备封禁，不重复扫描源存储，并将最终键放入 `v2:` 目标空间。每条封禁转换为当前转义键，并补齐 `loginId` 与 `kind`；原因、时间、服务等级和剩余 TTL 保持原意，过期记录仍由 `Apply` 跳过。`Summary.UpgradedRestrictions` 统计业务键或归属发生变化的记录，不包含统一增加的版本段；已规范化的记录不重复编码。

- 类型依据载荷中的服务、设备字段及已声明的 `kind` 校验，不能只按键前缀分类。例如账号 `service:account` 的账号封禁不会被误当成服务封禁。
- 记录已有 `loginId` 时，用完整身份重建新旧两种键，源键必须匹配其中之一。旧记录同时缺少 `loginId` 和 `kind` 且键中不含反斜杠时，才允许依据已存字段及完整键恢复账号。
- 带反斜杠而无账号归属、类型与字段矛盾、空账号、非法字段或损坏载荷会使预检失败。工具不猜测反斜杠是字面字符还是转义字符，也不从 Session 列表猜测封禁所属账号。
- 多条新旧记录指向同一规范键时，即使内容相似也拒绝迁移。它们可能有不同的剩余 TTL、等级或原读取优先级，不能自动挑选一条或合并期限。不同身份的旧键相互占位，但规范化后目标不同的情况可以迁移。
- 遇到歧义，应用应在停写窗口依据可信业务信息补齐源记录的 `loginId`、`kind`（`account`、`service` 或 `device`），保留实际服务/设备字段、原 Codec 与剩余 TTL。遇到重复目标，应明确选定要保留的业务规则并处理源记录，再重新预检。不要删除未知封禁以跳过错误；修复源数据不是工具自动执行的步骤。

封禁预检失败时，整个调用返回错误且不返回 Plan，先前的凭据转换也只存在于内存中，不会产生任何目标写入。原 `Prepare`、`PrepareTokenUpgrade` 的封禁原样复制语义保持不变；需要规范化限制的升级应选择 `PrepareAuthUpgrade`。

## 失败与切换

1. 目标已有任意键时返回 `ErrDestinationNotEmpty`，不主动覆盖。预检不写入；执行中途失败则可能已经写入部分目标数据。
2. `Result.Written` 仅统计已确认成功的写入，`Expired` 统计执行前到期的快照记录。网络错误下，失败的那次写入可能实际已生效，需要检查后端。
3. Plan 在首次执行尝试后不可重用，包括取消或失败；再次执行返回 `ErrPlanUsed`。没有自动回滚或清理，重新预检前应由应用明确处理部分目标数据。
4. 使用 `PrepareAuthUpgrade` 完整迁移并核验后，所有应用统一使用目标前缀，例如 `NewBuilder().KeyPrefix("dtoken-next:").AuthType("user")`；相同前缀升级时保留原配置即可。Manager、Nonce、OAuth2、Ticket、ShortKey 自动追加 `v2:`，须同步升级版本并保持相同前缀和体系。保留其他配置及 Codec；复制期间到期的凭据不会恢复有效。不能只要求重新登录而丢弃封禁和持久账号数据。
5. 切换后先验证登录校验、刷新、退出、封禁、Session 和独立能力，再决定何时单独清理旧空间。目标开始接受写入后，源与目标会分叉，不能直接退回源空间当作无损回滚。

同前缀升级时，新空间是旧边界下的 `v2:` 子树；后续清理旧数据必须排除该子树，不能按整个旧前缀清空。迁移预检也将源位置的 `v2:` 视为保留子树并排除；旧应用若曾自定义该后缀，需要另外核对其数据。

默认 KeyPrefix 仍为 `dtoken:`，实际运行时空间已经增加固定 `v2:` 段，旧空间不会自动被读取或删除。运行时访问凭据对旧载荷及缺少 AuthType 或 AccessID 的紧凑记录返回 ErrInvalidToken，不尝试 Codec 解码；PrepareAuthUpgrade 会补齐必要身份并转换为紧凑格式。运行时封禁不再回退旧业务键或推断缺失的归属；PrepareAuthUpgrade 转换键并补齐 loginId、kind。即使旧键与规范键的后缀恰好相同，也不能将旧封禁载荷原样复制到新空间后直接使用。本工具未执行实际部署数据迁移；同前缀切换、命名空间隔离、能力消费及迁移后生命周期单元测试已随 2026-11-26 根模块非 Redis 回归通过。真实 Redis 与部署迁移演练仍未执行，详见[验收记录](performance-acceptance_zh.md)。
