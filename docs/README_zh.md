[English](README.md) | 中文文档

# DToken-Go 文档中心

新项目从[快速开始](tutorial/quick-start_zh.md)和 [Auth 实例 API](api/auth_zh.md)入手。已有全局调用使用 `global` 包；框架请求显式绑定 Auth，独立能力通过 Service 和 WithCapabilities 组合。

文档约定：`dt` 或 `dtoken` 表示根包，`global` 只表示注册表入口，`auth` 是已构建实例，`mgr` 是其底层 Manager。带 package main 的示例提供完整程序或带标记的组合片段；其他片段需放在持有实例的函数内，补齐对应导入、业务处理器和错误处理。根模块要求 Go 1.25+，仓库工作区要求 Go 1.27+。

## 文档导航

### 快速上手

- [5 分钟快速开始](tutorial/quick-start_zh.md) - 最快上手方式

### 核心能力

- [登录认证](guide/core/authentication_zh.md) - 登录、登出、Token 管理
- [Session 与终端管理](guide/core/session-terminal_zh.md) - Session、终端、多端和下线管理
- [并发登录策略](guide/core/concurrency-login_zh.md) - Token 复用、最大登录数和多端策略
- [权限验证](guide/core/permission_zh.md) - 权限系统详解、通配符使用
- [AccessProvider](guide/core/access-provider_zh.md) - 动态权限和角色解析
- [封禁体系](guide/core/disable_zh.md) - 账号、服务、等级、设备封禁
- [多认证体系](guide/core/multi-auth_zh.md) - AuthType 隔离多套认证系统
- [Token 风格](guide/core/token-style_zh.md) - UUID、随机串、JWT 等 Token 生成方式
- [事件监听](guide/core/listener_zh.md) - 事件系统使用指南

### 安全与协议

- [Nonce 防重放](guide/security/nonce_zh.md) - 防止请求重放攻击
- [OAuth2 授权](guide/security/oauth2_zh.md) - OAuth2 授权码模式
- [SSO 单点登录](../sso/README_zh.md) - 独立 SSO 模块、Ticket、共享 Token 和远程会话模式说明
- [Refresh Token](guide/security/refresh-token_zh.md) - 刷新令牌机制
- [JWT 使用](guide/security/jwt_zh.md) - JWT Token 配置和使用
- [高级能力](guide/security/advanced-features_zh.md) - SSO、Ticket、短 Key、Token Introspection 等能力入口

### 集成与组件

- [框架集成](guide/integration/framework-integration_zh.md) - 核心 API 与 integrations 中间件用法
- [注解使用](guide/integration/annotation_zh.md) - 注解式中间件使用说明
- [组件生态](guide/integration/component-ecosystem_zh.md) - 存储、编解码、日志、Token 生成器和协程池
- [Redis 存储](guide/integration/redis-storage_zh.md) - Redis 存储配置详解
- [核心流程测试](guide/integration/core-flow-testing_zh.md) - Gin 真实流程测试与 Redis 测试说明

### 参考资料

- [Core API 速查](guide/reference/core-api-cheatsheet_zh.md) - 常用全局 API 调用方式
- [配置示例](guide/reference/configuration_zh.md) - Builder 常用配置说明
- [API 稳定性](guide/reference/api-stability_zh.md) - 公开 API 兼容性与版本规则
- [Auth 实例 API](api/auth_zh.md) - 应用入口、选项与全局调用对照
- [全局 API](api/dtoken_zh.md) - 全局认证 API 参考

### 设计文档

- [架构设计](design/architecture_zh.md) - 职责分层、依赖方向与装配路径
- [源码目录](design/source-layout_zh.md) - 目录职责、引擎文件索引与代码放置规则
- [模块化设计](design/modular_zh.md) - 包依赖、Go 模块边界与结构检查
- [API 与生命周期](design/api-lifecycle_zh.md) - 公开 API、组件所有权与实例共享
- [迁移指南](design/module-migration_zh.md) - 包路径升级、发布顺序与消费者验证
- [自动续签设计](design/auto-renew_zh.md) - 异步续签原理

## 示例项目

- [quick_start](../examples/quick_start/) - 快速开始示例
- [gin](../examples/gin/) - Gin 集成示例
- [gin_core_app](../tests/gin_core_app/) - Gin 核心流程测试应用
- [gf](../examples/gf/) - GoFrame 集成示例
- [beego](../examples/beego/) - Beego 集成示例
- [echo](../examples/echo/) - Echo 集成示例
- [fiber](../examples/fiber/) - Fiber 集成示例
- [chi](../examples/chi/) - Chi 集成示例
- [hertz](../examples/hertz/) - Hertz 集成示例
- [kratos](../examples/kratos/) - Kratos 集成示例

## 外部资源

- [GitHub 仓库](https://github.com/Zany2/dtoken-go)

---

**dtoken-go**
