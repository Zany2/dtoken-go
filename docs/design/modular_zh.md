[English](modular.md) | 中文文档

# 模块化设计

## 设计目标

将项目拆分为多个独立模块，实现：

- 按需导入
- 最小依赖
- 多模块独立维护
- 清晰的职责划分
- 方便扩展新的组件和框架集成

## 模块划分

### 核心模块 (core)

```text
github.com/Zany2/dtoken-go/core
```

**职责**：
- 配置、上下文、Manager、Builder
- 权限、角色、封禁、Session 核心逻辑
- Listener、Nonce、OAuth2 等能力

**特点**：
- 不依赖 Web 框架
- 不依赖具体存储实现
- 通过 `adapter` 接口解耦组件

### 全局工具模块 (dtoken)

```text
github.com/Zany2/dtoken-go/dtoken
```

**职责**：
- 对外暴露全局便捷 API
- 管理全局 `Manager`
- 提供统一的 `context.Context` 风格调用方式

### SSO 模块

```text
github.com/Zany2/dtoken-go/sso
github.com/Zany2/dtoken-go/sso/storage/redis
```

**职责**：
- 协调服务端与客户端单点登录流程
- 签发并消费一次性 Ticket 和 OAuth2 Code
- 将 SSO 存储所有权与核心 Manager 隔离

Redis 子模块只提供生产环境存储构造器，不会让基础 `sso` 模块引入 Redis 依赖。

### 存储模块

#### Memory 存储

```text
github.com/Zany2/dtoken-go/com/storage/memory
```

**特点**：
- 零外部依赖
- 适合开发和测试

#### Redis 存储

```text
github.com/Zany2/dtoken-go/com/storage/redis
```

**特点**：
- 适合生产环境
- 支持分布式部署
- 依赖 `github.com/redis/go-redis/v9`

### 组件模块

当前除了存储外，还拆分了这些可替换组件：

- `com/codec/base64`
- `com/codec/json`
- `com/codec/jsonv2` - 基于 Go 1.27 `encoding/json/v2` 的 JSON v2 编解码器（需要启用 `GOEXPERIMENT=jsonv2`）
- `com/codec/msgpack`
- `com/generator/dgenerator`
- `com/log/gf`
- `com/log/nop`
- `com/log/dlog`
- `com/pool/ants`

### 框架集成模块

当前已提供这些独立集成：

- `integrations/gin`
- `integrations/gf`
- `integrations/echo`
- `integrations/fiber`
- `integrations/chi`
- `integrations/hertz`
- `integrations/kratos`
- `integrations/beego`

每个集成模块都负责：

- 请求上下文适配
- `DTokenContext` 注入
- 中间件封装
- 注解式校验封装
- `export.go` 统一导出层

Beego 遵循相同的模块边界，但通过 `RouteAccessHandlerFromAnnotations` 将注解规则映射到 Beego 过滤器。

### 集成组件导出维护

各框架包有意暴露相同的内置组件别名、构造器、强类型 API、错误和门面操作。Gin 版本的 `component_export.go`、`api_export.go`、`error_export.go` 和 `facade_export.go` 分别是它们的标准源，其他集成包中的对应文件由这些标准源生成；GoFrame 专用日志导出只在生成 `component_export.go` 时自动注入。

修改标准导出清单后，在 `integrations/gin` 目录重新生成其他框架文件：

```bash
go generate
```

不要直接修改生成的导出文件。
如果导出清单引入了新的组件模块，还需要单独更新受影响集成包的 `go.mod`。

`export.go` 暂时继续独立维护：所有集成包暴露的符号一致，但部分包按业务领域分组，其他包使用单个声明块。保留两种布局可以避免大量只涉及排版的改写。

## 依赖关系

```text
应用代码
  ↓
框架集成 (integrations/*)    或    dtoken    或    sso
  ↓
core
  ↓
com/storage/* / com/codec/* / com/log/* / com/pool/* / com/generator/*
```

## 按需导入

### 场景1：只使用核心能力

```bash
go get github.com/Zany2/dtoken-go/core
go get github.com/Zany2/dtoken-go/dtoken
go get github.com/Zany2/dtoken-go/com/storage/memory
```

**适合场景**：
- 非 Web 场景
- 自己封装请求上下文
- 希望完全掌控初始化过程

### 场景2：使用框架集成包

```bash
go get github.com/Zany2/dtoken-go/integrations/gin
go get github.com/Zany2/dtoken-go/com/storage/redis
```

**适合场景**：
- 直接接入 Gin / Echo / Fiber / Chi / GoFrame / Hertz / Kratos / Beego
- 希望使用集成包统一导出的构建器、中间件和工具函数

### 场景3：替换组件实现

```bash
go get github.com/Zany2/dtoken-go/core
go get github.com/Zany2/dtoken-go/dtoken
go get github.com/Zany2/dtoken-go/com/storage/redis
go get github.com/Zany2/dtoken-go/com/log/dlog
go get github.com/Zany2/dtoken-go/com/pool/ants
```

**适合场景**：
- 生产环境
- 需要替换日志、存储、协程池等组件

## 模块独立性

### 每个模块都有独立 go.mod

当前仓库中核心、SSO、组件、集成和示例模块在独立发布时都具备独立 `go.mod`。

例如：

```text
core/go.mod
dtoken/go.mod
com/storage/memory/go.mod
com/storage/redis/go.mod
com/codec/json/go.mod
integrations/gin/go.mod
integrations/gf/go.mod
integrations/beego/go.mod
examples/quick_start/go.mod
sso/go.mod
sso/storage/redis/go.mod
```

### 本地开发通过 go.work 统一管理

当前仓库使用 `go.work` 将全部模块串联起来。

```go
go 1.27.0

use (
    ./com/codec/base64
    ./com/codec/json
    ./com/codec/jsonv2
    ./com/codec/msgpack
    ./com/generator/dgenerator
    ./com/log/dlog
    ./com/log/gf
    ./com/log/nop
    ./com/pool/ants
    ./com/storage/memory
    ./com/storage/redis
    ./core
    ./defaults
    ./dtoken
    ./examples/beego
    ./examples/chi
    ./examples/echo
    ./examples/fiber
    ./examples/gf
    ./examples/gin
    ./examples/hertz
    ./examples/kratos
    ./examples/quick_start
    ./examples/sso_client
    ./examples/sso_gin_client
    ./examples/sso_gin_server
    ./examples/sso_server
    ./integrations/authcheck
    ./integrations/beego
    ./integrations/chi
    ./integrations/echo
    ./integrations/fiber
    ./integrations/gf
    ./integrations/gin
    ./integrations/hertz
    ./integrations/kratos
    ./sso
    ./sso/storage/redis
    ./tests/gin_core_app
    ./tests/gin_core_flow
)
```

**优势**：
- 本地开发无缝调试
- 模块之间直接联动
- 无需频繁发布版本做本地联调

## 版本管理

### 版本同步原则

多模块仓库建议保持对外发布版本同步，例如：

- `core`
- `dtoken`
- `com/storage/*`
- `integrations/*`

尽量使用相同发布版本，减少跨模块组合时的兼容性问题。

### 兼容性保证

- 核心接口变更要同步更新相关模块
- 集成层导出能力应与 `dtoken` / `core` 保持一致
- 文档、示例、导出层需要一起更新

## 扩展新模块

### 添加新存储

1. 创建目录：`com/storage/mysql/`
2. 创建 `go.mod`
3. 实现 `adapter.Storage`
4. 添加到 `go.work`
5. 编写文档和示例

### 添加新框架集成

1. 创建目录：`integrations/iris/`
2. 创建 `go.mod`
3. 实现上下文适配
4. 增加中间件、注解封装、`export.go`
5. 添加到 `go.work`
6. 编写示例和文档

### 添加新组件

1. 创建目录：`com/log/xxx/` 或 `com/codec/xxx/`
2. 实现对应 `adapter` 接口
3. 添加到 `go.work`
4. 编写测试与文档

## 优势总结

| 特性 | 单体模式 | 当前模块化设计 | 优势 |
|------|----------|----------------|------|
| 依赖控制 | 弱 | 强 | 按需导入 |
| 框架隔离 | 弱 | 强 | 集成互不影响 |
| 组件替换 | 一般 | 强 | Storage / Codec / Log / Pool / Generator 可替换 |
| 本地联调 | 一般 | 强 | `go.work` 统一管理 |
| 扩展性 | 一般 | 强 | 易于新增组件与集成 |

## 下一步

- [架构设计](architecture_zh.md)
- [自动续签设计](auto-renew_zh.md)
- [DToken API 文档](../api/dtoken_zh.md)
