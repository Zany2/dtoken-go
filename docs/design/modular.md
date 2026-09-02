English | [中文文档](modular_zh.md)

# Modular Design

## Design Goals

Split the project into multiple independent modules in order to achieve:

- on-demand imports
- minimal dependencies
- independent multi-module maintenance
- clear separation of responsibilities
- easier extension for new components and framework integrations

## Module Division

### Core Module (core)

```text
github.com/Zany2/dtoken-go/core
```

**Responsibilities**:
- config, context, manager, builder
- core logic for permissions, roles, disable, session
- listener, nonce, and OAuth2 capability

**Features**:
- no web framework dependency
- no concrete storage dependency
- component abstraction through `adapter`

### Global Utility Module (dtoken)

```text
github.com/Zany2/dtoken-go/dtoken
```

**Responsibilities**:
- expose global convenience APIs
- manage the global `Manager`
- provide unified `context.Context`-based calling style

### SSO Modules

```text
github.com/Zany2/dtoken-go/sso
github.com/Zany2/dtoken-go/sso/storage/redis
```

**Responsibilities**:
- coordinate server and client single sign-on flows
- issue and consume one-time Ticket and OAuth2 Code values
- isolate SSO storage ownership from the core manager

The Redis submodule adds a production storage constructor without making Redis a dependency of the base `sso` module.

### Storage Modules

#### Memory Storage

```text
github.com/Zany2/dtoken-go/com/storage/memory
```

**Features**:
- zero external dependencies
- suitable for development and testing

#### Redis Storage

```text
github.com/Zany2/dtoken-go/com/storage/redis
```

**Features**:
- suitable for production
- supports distributed deployment
- depends on `github.com/redis/go-redis/v9`

### Component Modules

Besides storage, the current repository also splits out these replaceable components:

- `com/codec/base64`
- `com/codec/json`
- `com/codec/jsonv2` - JSON v2 codec backed by Go 1.27 `encoding/json/v2` (requires `GOEXPERIMENT=jsonv2`)
- `com/codec/msgpack`
- `com/generator/dgenerator`
- `com/log/gf`
- `com/log/nop`
- `com/log/dlog`
- `com/pool/ants`

### Framework Integration Modules

The current repository provides these independent integrations:

- `integrations/gin`
- `integrations/gf`
- `integrations/echo`
- `integrations/fiber`
- `integrations/chi`
- `integrations/hertz`
- `integrations/kratos`
- `integrations/beego`

Each integration module is responsible for:

- request context adaptation
- `DTokenContext` injection
- middleware wrappers
- annotation-style check wrappers
- unified export layer in `export.go`

Beego follows the same module boundary but maps annotation rules to Beego filters through `RouteAccessHandlerFromAnnotations`.

### Integration Component Export Maintenance

The framework packages intentionally expose the same bundled component aliases, constructors, typed APIs, errors, and facade operations. The Gin versions of `component_export.go`, `api_export.go`, `error_export.go`, and `facade_export.go` are their canonical sources; the corresponding files in the other integration packages are generated from them. GoFrame-specific logger exports are injected only while generating `component_export.go`.

After changing the canonical export list, regenerate the other framework files from `integrations/gin`:

```bash
go generate
```

Generated export files should not be edited directly.
If the export list introduces a new component module, update the affected integration `go.mod` files separately.

`export.go` remains independently maintained for now: all integration packages expose the same symbols, but some group operations by domain while others use one consolidated declaration. Keeping both layouts avoids a large formatting-only rewrite.

## Dependency Relationships

```text
Application Code
  ↓
Framework Integration (integrations/*)    or    dtoken    or    sso
  ↓
core
  ↓
com/storage/* / com/codec/* / com/log/* / com/pool/* / com/generator/*
```

## On-Demand Imports

### Scenario 1: Core Capability Only

```bash
go get github.com/Zany2/dtoken-go/core
go get github.com/Zany2/dtoken-go/dtoken
go get github.com/Zany2/dtoken-go/com/storage/memory
```

**Suitable for**:
- non-web usage
- custom request/context integration
- full control over initialization

### Scenario 2: Using a Framework Integration Package

```bash
go get github.com/Zany2/dtoken-go/integrations/gin
go get github.com/Zany2/dtoken-go/com/storage/redis
```

**Suitable for**:
- direct integration with Gin / Echo / Fiber / Chi / GoFrame / Hertz / Kratos / Beego
- using the integration package as the unified export surface

### Scenario 3: Replacing Components

```bash
go get github.com/Zany2/dtoken-go/core
go get github.com/Zany2/dtoken-go/dtoken
go get github.com/Zany2/dtoken-go/com/storage/redis
go get github.com/Zany2/dtoken-go/com/log/dlog
go get github.com/Zany2/dtoken-go/com/pool/ants
```

**Suitable for**:
- production environments
- replacing logging, storage, pool, or other components

## Module Independence

### Each Module Has Its Own go.mod

Core, SSO, component, integration, and example modules all have independent `go.mod` files where they are published separately.

For example:

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

### Local Development Is Unified by go.work

The current repository uses `go.work` to connect all modules.

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

**Advantages**:
- seamless local debugging
- direct linkage between modules
- no need to publish versions for local integration testing

## Version Management

### Version Synchronization Principle

For a multi-module repository, public versions should stay aligned as much as possible across:

- `core`
- `dtoken`
- `com/storage/*`
- `integrations/*`

Using the same released version helps reduce cross-module compatibility issues.

### Compatibility Guarantee

- core interface changes should be propagated to related modules
- integration exports should stay aligned with `dtoken` and `core`
- docs, examples, and export layers should be updated together

## Extending New Modules

### Add a New Storage Module

1. Create directory: `com/storage/mysql/`
2. Create `go.mod`
3. Implement `adapter.Storage`
4. Add it to `go.work`
5. Write documentation and examples

### Add a New Framework Integration

1. Create directory: `integrations/iris/`
2. Create `go.mod`
3. Implement context adaptation
4. Add middleware, annotation wrappers, and `export.go`
5. Add it to `go.work`
6. Write examples and documentation

### Add a New Component

1. Create directory: `com/log/xxx/` or `com/codec/xxx/`
2. Implement the corresponding `adapter` interface
3. Add it to `go.work`
4. Write tests and docs

## Advantages Summary

| Feature | Monolithic Style | Current Modular Design | Benefit |
|---------|------------------|------------------------|---------|
| Dependency control | Weak | Strong | On-demand import |
| Framework isolation | Weak | Strong | Integrations do not affect each other |
| Component replacement | Average | Strong | Storage / Codec / Log / Pool / Generator are replaceable |
| Local integration | Average | Strong | Unified by `go.work` |
| Extensibility | Average | Strong | Easier to add components and integrations |

## Next Steps

- [Architecture Design](architecture.md)
- [Auto-Renew Design](auto-renew.md)
- [DToken API Documentation](../api/dtoken.md)
