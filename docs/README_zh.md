[English](README.md) | 中文文档

# DToken-Go 文档中心

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
- [DToken API](api/dtoken_zh.md) - 全局工具类完整 API

### 设计文档

- [架构设计](design/architecture_zh.md) - 系统架构、数据流转
- [自动续签设计](design/auto-renew_zh.md) - 异步续签原理
- [模块化设计](design/modular_zh.md) - 模块划分策略

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
