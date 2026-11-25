# 源码目录与职责

[English](source-layout.md) | 中文文档

本文回答“代码在哪里、改动应该放在哪里”。系统分层见[架构设计](architecture_zh.md)，包依赖和发布边界见[模块化设计](modular_zh.md)，公开 API 与资源所有权见[API 与生命周期](api-lifecycle_zh.md)。目录说明描述当前组织方式，不代表各项功能已完成行为验证。

## 目录地图

路径相对于仓库根目录。`go.mod` 决定模块边界，`go.work` 组合本地开发模块；一个目录独立承担职责，并不意味着独立发布。

```text
dtoken-go/
├── auth*.go              实例门面及分类操作
├── builder.go            应用构建入口
├── options.go            实例调用选项
├── types.go              公共领域类型别名
├── engine_api.go         高级引擎 API 与装配类型
├── config.go             根包配置入口
├── registry.go           共享注册表、AuthType 选择与关闭
├── default_*.go          默认体系转发
├── internal/
│   ├── engine/           核心认证实现
│   ├── builder/          配置与组件装配
│   ├── defaults/         内置默认组件工厂
│   ├── utils/            内部共享基础操作
│   └── banner/           启动展示
├── contract/             基础契约、领域模型与请求契约
│   ├── nonce/            Nonce 服务契约
│   ├── oauth2/           OAuth2 服务契约
│   ├── ticket/           Ticket 服务契约
│   ├── shortkey/         ShortKey 服务契约
│   └── storagetest/      可复用存储契约检查
├── config/               核心配置及常量
├── errors/               公共错误及错误码
├── event/                事件契约与监听分发实现
├── nonce/                Nonce 实现与服务封装
├── oauth2/               OAuth2 实现与服务封装
├── ticket/               Ticket 实现与服务封装
├── shortkey/             ShortKey 实现与服务封装
├── httpauth/             共享 HTTP 鉴权检查及错误映射
│   └── context/          请求作用域、能力绑定与认证操作
├── integrations/         各框架的适配实现
├── codec/                编解码组件
├── generator/            Token 生成组件
├── logging/              日志组件
├── pool/                 异步执行组件
├── storage/              存储组件
├── sso/                  独立 SSO 子系统
├── examples/             接入示例应用
├── tests/                应用级组合验证
├── docs/                 教程、指南、参考与设计
└── .github/              CI、仓库检查脚本与协作配置
```

## 应用入口与装配

| 位置 | 职责 | 放置约束 |
| --- | --- | --- |
| 根包 `Auth` 与 `auth_*.go` | 将应用调用转换为引擎操作，提供实例生命周期入口 | 认证规则和存储流程放在引擎中 |
| 根 `builder.go` | 应用配置入口、内置日志和池配置、返回 `Auth` | 使用内部构建器和默认工厂完成装配 |
| `options.go`、`types.go`、`config.go` | 公开选项、共享模型别名与配置入口 | 保持公共模型的唯一声明来源 |
| `engine_api.go` | 暴露受支持的高级引擎类型与装配选项 | 类型别名仍属于公开 API，不能因实现位于 internal 而任意变更 |
| 根 `registry.go`、`default_*.go`  | 共享注册表、默认调用及 AuthType 选择器 | 核心引擎和 HTTP 公共层不反向依赖注册表 |
| `internal/builder` | 配置校验、组件和工厂解析、引擎构建 | 接收组件契约，不选择具体存储、日志或池实现 |
| `internal/defaults` | 提供内置默认组件工厂 | 默认实现的选择集中在这里；应用仍可显式替换组件 |

这三处 Builder 相关代码分别承担应用入口、通用装配、默认接线。新增组件实现放在组件包中；仅当需要应用级配置入口时，才扩展根 Builder。

## 公共基础与独立能力

| 位置 | 内容 |
| --- | --- |
| `contract/storage.go`、`codec.go`、`generator.go`、`log.go`、`pool.go` | 可替换组件契约 |
| `contract/model.go`、`access.go` | 认证领域模型、权限与角色提供器契约 |
| `contract/context.go` | 框架请求、响应与 Cookie 适配契约 |
| `contract/<capability>/service.go` | 独立能力的服务接口和共享数据类型 |
| `contract/storagetest` | 供存储实现复用的契约验证工具，运行时不依赖它 |
| `config`、`errors` | 核心配置和公共错误体系 |
| `event` | 事件载荷、发布契约和监听器运行时实现 |
| `nonce`、`oauth2`、`ticket`、`shortkey` | 能力自身的配置、业务实现及事件服务封装 |

能力包中的 `service.go` 负责实现面向请求的服务契约，并连接事件发布器；具体能力操作留在本包的 Manager 或 Server 实现中。能力专属配置跟随能力包，组件专属配置跟随组件包，避免所有配置集中到核心 `config.Config`。

`contract` 当前同时容纳组件、领域和 HTTP 契约。沿用已有分类文件；只有出现独立的依赖或维护需求时才新增子包。契约中不装配具体组件，也不导入能力实现。`event` 含有运行时行为，不能当作纯类型声明包。

## 核心引擎文件索引

以下文件位于同一个 `internal/engine` 包，按状态所有权和共享操作划分。文件名前缀用于阅读导航，不是 Go 可见性边界。

| 文件或文件组 | 职责 |
| --- | --- |
| `manager.go` | Manager 状态组合与组件访问入口 |
| `constructor.go` | 引擎装配选项与组件所有权 |
| `model.go` | 引用 contract 中的共享模型 |
| `strategy.go`、`auth_options.go` | 策略扩展点与引擎操作选项 |
| `auth_login.go`、`auth_logout.go`、`auth_terminate.go` | 登录与终端下线操作 |
| `auth_refresh.go`、`auth_introspection.go` | 刷新凭证与令牌状态查询 |
| `auth_session*.go` | 会话查询与扩展数据操作 |
| `auth_access.go`、`auth_permission.go`、`auth_role.go` | 访问提供器、权限与角色操作 |
| `auth_disable.go` | 封禁操作入口 |
| `internal_account_lock.go` | 账号写锁注册表及锁生命周期 |
| `internal_concurrency.go` | 登录并发策略的内部操作 |
| `internal_maintenance.go` | Token 维护代次及合并的活跃时间状态 |
| `internal_lifecycle.go` | 后台任务接收、关闭与资源释放 |
| `internal_session.go`、`internal_token.go`、`internal_terminal.go` | Session、Token 和终端的共享内部操作 |
| `internal_storage.go`、`internal_keys.go` | 持久化、有效期转换及存储键构造 |
| `internal_access_mutation.go`、`internal_access_values.go` | 访问数据变更流程和权限、角色列表操作 |
| `internal_disable.go`、`internal_events.go` | 封禁基础操作和引擎事件发布 |

Manager 分别持有账号协调、Token 维护和运行时关闭状态。这些文件共享同一套状态与基础操作，当前保持同包组织。新增功能优先归入对应操作文件；仅在状态所有权与依赖能够独立时考虑拆包。

## HTTP 公共层与框架目录

`httpauth` 负责公共检查、Manager 解析和错误映射；`httpauth/context` 负责请求中的 Token 获取、Cookie 操作、认证操作和能力绑定。框架提供请求适配器，不在框架目录中重写领域规则。

八个框架目录为 `beego`、`chi`、`echo`、`fiber`、`gf`、`gin`、`hertz`、`kratos`。共同的文件组织如下：

| 文件 | 职责 |
| --- | --- |
| `context.go` | 适配框架原生请求和响应 |
| `middleware.go` | 接入框架处理链、配置及鉴权失败处理 |
| `instance.go` | 提供 Auth 实例注入入口 |
| `annotation.go` | 将注解配置转换为请求鉴权检查 |
| `context_facade.go`、`context_*_facade.go` | 提供框架原生上下文的操作快捷入口 |
| `export.go`、`error_export.go` | 统一公共类型、常量及错误的便捷导出 |

Gin 的两个导出文件是标准源，`integrations/gin/internal/integrationexportgen` 维护其余框架的对应生成文件。该生成器不生成中间件、请求适配器或全部门面方法。新增共享操作需要同步对应框架入口，并维护现有覆盖检查。

## 示例、验证和文档

包内 `*_test.go` 跟随实现；根目录的 `architecture*_test.go` 约束跨包结构。`tests/gin_core_app` 提供组合测试应用，`tests/gin_core_flow` 验证应用流程；`examples` 优先展示接入方式，避免成为实现内部功能的依赖。

`.github/workflows` 编排 CI，`.github/scripts` 保存模块、文档示例及仓库外消费者检查工具。这些工具和 Gin 导出生成器保留现有位置；新增工具放到其实际维护对象附近。

`docs/tutorial` 负责入门，`docs/guide` 负责用法，`docs/api` 负责 API 参考，`docs/design` 负责设计约定。新增或移动文档时同步中英文入口及相对链接，功能流程说明链接到对应指南。

## 改动放置规则

| 改动 | 优先位置 |
| --- | --- |
| 核心认证规则 | `internal/engine`，必要时补根实例门面 |
| 动态权限或组件扩展接口 | 先核对 `contract` 中已有契约 |
| 新存储、编码或日志实现 | 对应组件目录；模块拆分按依赖和发布需求决定 |
| 独立能力实现 | 对应能力包；需要 HTTP 注入时维护其服务契约 |
| 所有框架共用的请求鉴权 | `httpauth` 或 `httpauth/context` |
| 某个框架的原生行为适配 | 对应 `integrations/<framework>` |
| 全局注册与查找 | 根 `registry.go` |
| 跨组件使用场景 | `examples` 或 `tests`，按教学与验证目的区分 |

SSO 保留为独立模块，本指南仅标示它的仓库位置，内部目录和协议另行梳理。
