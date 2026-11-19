# 模块化设计

[English](modular.md) | 中文文档

本文定义包依赖与 Go 模块发布边界。整体分层见[架构设计](architecture_zh.md)，目录及文件职责见[源码目录](source-layout_zh.md)，升级步骤见[迁移指南](module-migration_zh.md)。

| 职责 | 包 |
| --- | --- |
| 应用入口 | 根实例门面、global 注册表 |
| 公共契约与配置 | contract、config、errors |
| 事件服务 | event：发布契约及监听分发实现 |
| 内部实现与装配 | internal/engine、internal/builder、internal/defaults |
| 独立能力 | nonce、oauth2、ticket、shortkey |
| HTTP 公共层 | httpauth、httpauth/context |
| 框架适配 | integrations/<framework> |
| 组件实现 | codec、generator、logging、pool、storage |
| 单点登录 | sso、sso/storage/redis |

## 依赖规则

- 核心引擎依赖 contract、config、errors、event 和内部基础工具，不依赖框架、独立能力或具体组件实现。
- 能力实现依赖公共契约及基础工具，通过构造器接收组件和可选事件发布器，不反向依赖根门面、global 或核心引擎。
- contract 保持共享接口和数据类型，不导入能力实现、框架或默认装配。event 自身包含运行时实现。
- config 可依赖 contract 和 errors；contract 不反向依赖 config。errors 与内部基础工具不依赖上层运行时，event 不依赖引擎或装配层。
- 根包通过内部构建器和默认工厂装配基础组件，实例调用无需全局注册。global 依赖根实例门面并管理自己的注册表。
- internal/builder 依赖引擎和组件契约，不直接选择组件实现。internal/defaults 明确选择内置 JSON、生成器、日志、池和内存存储；更换内置默认组件时同步更新装配依赖白名单。
- HTTP 公共层调用引擎和能力服务契约，不导入能力实现或 global。注册表查找通过显式 ManagerResolver 注入。
- 框架适配器调用 HTTP 公共层，处理原生请求、响应和处理链。框架 Builder 转发根构建器并返回 Auth，不转导出全局 API 或可选组件构造器。
- 框架适配器可以引用自身目录下的辅助子包，不直接导入具体组件、内部构建器、其他框架适配器、SSO 或示例与测试应用。根 Builder 的默认组件依赖属于有意的间接装配关系。
- 组件实现可依赖自身组件目录、公共契约、配置、错误和内部基础工具，不反向依赖引擎、HTTP 或装配层。contract/storagetest 仅用于测试，不作为运行时依赖。
- 新组件通过组件包接入。Gin 标准源和生成器统一维护 export.go、error_export.go；其余适配代码按框架维护。

这些规则区分实现调用和装配依赖：根 Builder 选择内置组件是有意的装配职责，核心引擎不应因此依赖具体组件。

## Go 模块边界

| 发布单元 | 目录范围 | 拆分目的 |
| --- | --- | --- |
| 根模块 | 根门面、global、internal、contract、config、errors、event、四个能力包、httpauth，以及下列子模块之外的基础组件 | 保持认证核心及基础能力的统一版本 |
| 可选组件 | codec/jsonv2、codec/msgpack、logging/gf、storage/redis | 隔离可选依赖或工具链要求 |
| 框架集成 | integrations 下的八个框架目录 | 隔离框架依赖并单独发布 |
| SSO | sso、sso/storage/redis | 独立子系统和存储扩展 |
| 示例与应用测试 | examples 下的应用、tests/gin_core_app、tests/gin_core_flow | 隔离应用依赖，不作为核心运行时依赖 |

每个 go.mod 定义实际模块边界，go.work 仅组合本地开发模块。nonce、oauth2、ticket、shortkey 是独立组合的包，仍随根模块发布；contract 下的能力接口也不单独发布。

新增目录优先使用现有模块。仅在需要隔离第三方依赖、工具链或发布节奏时新增 go.mod，并同步工作区、CI、示例和发布说明。公开包路径变动会影响使用方，不能视为单纯的目录整理。

## 发布与所有权

根模块和扩展的 tag 命名、依赖发布顺序及消费者验证步骤见[迁移指南](module-migration_zh.md)。源码中的版本声明不能证明对应版本已经发布。

运行时资源所有权独立于模块发布边界：引擎关闭其自有组件，能力服务借用组件，注册表与 Auth 共享引擎。完整约定见 [API 与生命周期](api-lifecycle_zh.md)。

## 现有结构检查

- 根目录 architecture_test.go 通过仓库内直接导入白名单约束契约、配置、错误、事件、内部工具、引擎、能力、HTTP 公共层、组件及两层内部装配代码；HTTP 与框架另有禁止依赖规则。
- 检查扫描非 _test.go 的 Go 文件，源码仓库中存在的独立组件和框架子模块也参与扫描。发布后的根模块没有 integrations 目录时跳过框架扫描；组件父目录仍检查其保留的内置实现。
- architecture_integrations_test.go 检查各框架请求操作名称的覆盖范围，不替代签名或行为验证。
- Gin 导出生成器的检查模式检查两个共享导出文件的一致性。
- CI 中的独立模块及仓库外消费者检查验证工作区之外的依赖解析和公开使用方式。

导入检查不分析第三方依赖的传递关系，不替代公开 API、编译或功能行为验证。功能校验按对应模块另行进行。
