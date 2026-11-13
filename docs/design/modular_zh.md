# 模块化设计

[English](modular.md) | 中文文档

包边界按职责划分，Go 模块边界用于隔离可选依赖与独立发布。路径和发布规则见[迁移指南](module-migration_zh.md)。

| Layer | Packages |
| --- | --- |
| Application entry | Root instance facade, global registry |
| Public contracts | contract, config, errors, event |
| Implementation | internal/engine, internal/builder, internal/defaults |
| Independent capabilities | nonce, oauth2, ticket, shortkey |
| Shared HTTP | httpauth, httpauth/context |
| Framework adapters | integrations/<framework> |
| Components | codec, generator, logging, pool, storage |
| Single sign-on | sso, sso/storage/redis |

## 依赖规则

- 核心引擎依赖公共契约，不依赖框架或独立能力包。
- 能力通过构造器接收存储、编解码器和可选事件发布器。
- 根包装配基础组件，实例调用无需全局注册。global 依赖根实例门面并管理自己的注册表。
- HTTP 上下文显式接收服务，框架只负责请求适配和中间件。
- 新组件通过组件包接入；共用框架导出由 Gin 标准源和生成器维护。

## 发布与所有权

基础包和能力包共用根模块版本。可选组件、框架与 SSO 独立发布。工作区、独立模块及仓库外消费者检查共同验证依赖边界。

引擎关闭其拥有的 Storage、Logger、Pool 和后台任务。共享组件通过根包 WithComponentOwnership 声明为调用方持有。global 替换同 AuthType 的 Manager 时关闭旧实例；应用负责安排停止请求和关闭资源的顺序。能力服务借用组件，不独立关闭它们。

HTTP 包不依赖 global；注册表查找通过显式 ManagerResolver 注入。框架 Builder 统一返回根包 Auth，框架包不再转导出全局调用或可选组件构造器。

公开 API 定位、组件所有权及注册表替换规则见 [API 与生命周期](api-lifecycle_zh.md)。HTTP 能力绑定依赖 contract 下的服务接口，不依赖具体实现。
