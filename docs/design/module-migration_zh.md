# 模块组织与迁移

[English](module-migration.md) | 中文文档

## 模块归属

仓库共 30 个模块，包含示例与测试应用。根模块 `github.com/Zany2/dtoken-go` 统一发布以下包，共用一个版本。

- `core`
- `dtoken`
- `defaults`
- `com/codec/base64`、`com/codec/json`
- `com/generator/dgenerator`
- `com/log/dlog`、`com/log/nop`
- `com/pool/ants`
- `com/storage/memory`
- `integrations/authcheck`

独立库模块包括 `com/codec/jsonv2`、`com/codec/msgpack`、`com/log/gf`、`com/storage/redis`、八个框架集成、`sso` 和 `sso/storage/redis`。示例和应用流程测试继续使用独立模块，隔离各自的框架依赖。

包导入路径继续适用。例如 `github.com/Zany2/dtoken-go/core/manager` 和 `github.com/Zany2/dtoken-go/dtoken` 现在属于根模块，统一使用仓库根目录的 `go.mod`。

## 构建入口

| 入口 | 使用场景 |
| --- | --- |
| `dtoken.NewBuilder()` | 应用首选入口，提供默认组件和可选功能的高层配置 |
| `defaults.NewBuilder()` | 使用默认组件的底层核心构建器 |
| `core/builder` 的 `builder.NewBuilder()` | 显式配置组件与工厂 |
| 框架包的 `NewBuilder()` | 现有框架便捷入口，返回默认核心构建器 |

应用通过 `dtoken.NewBuilder().Build()` 创建 Manager，通过 `dtoken.New(mgr)` 使用实例 API，或通过 `dtoken.SetManager(mgr)` 注册全局调用入口。各 Builder 返回类型和现有 API 按当前契约使用。

快速开始示例采用应用首选构建器。框架集成通过 `integrations/authcheck` 复用鉴权规则。组件实现 `core/adapter` 的扩展契约，由 `defaults` 装配默认实现。

## 使用方迁移

从独立核心模块升级时，需要清理应用 `go.mod` 中上述十一个合并模块的直接、间接依赖和本地替换规则，再添加目标版本的根模块依赖。独立扩展继续保留自己的依赖声明，并选择已改为依赖根模块的扩展版本。

旧子模块与根模块可能同时提供同一包，导致导入歧义。升级时需要检查应用及其传递依赖的模块声明。现有代码中的包导入路径可以继续使用。

选定兼容的发布版本后，执行 `go mod tidy` 和应用验证命令。本分支源码开发使用仓库工作区，独立模块 CI 通过临时本地替换映射当前源码。

## 发布协调

本分支的根模块依赖暂沿用原有源码基线 `v0.0.3`。使用方通过远程解析新结构前，需要先发布包含根模块的版本，分支中的依赖声明本身不能证明远程版本已经可用。

正式发布时，选择可用的根版本，更新扩展模块的根依赖，使用 `vX.Y.Z` 形式的仓库 tag 发布根模块。独立模块使用目录前缀 tag，例如 `integrations/gin/vX.Y.Z` 和 `com/storage/redis/vX.Y.Z`。先发布被依赖模块，再发布扩展与示例所依赖的版本。

对外公布前，在仓库之外创建干净的使用方模块，设置 `GOWORK=off`，安装目标版本的门面包与框架集成，并验证登录、受保护访问和退出。此步骤用于确认远程发布内容与版本依赖。

## 验证

在仓库根目录执行以下命令，检查合并后的主模块。

```bash
go vet ./...
go test -count=1 -timeout=5m ./...
```

独立扩展从各自模块目录验证，`com/codec/jsonv2` 需要启用 `GOEXPERIMENT=jsonv2`。CI 会遍历工作区模块，并运行核心、存储、SSO 和 HTTP 流程的 Redis 场景。

以下 Bash 命令使用 `GOWORK=off` 与临时本地替换检查全部模块，执行 vet 和测试，依赖解析使用临时模块文件。

```bash
bash .github/scripts/check-modules.sh
```

该检查验证当前源码的独立依赖关系，仓库外的使用方验证用于确认远程发布。
