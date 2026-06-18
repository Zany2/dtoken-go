English | [中文文档](README_zh.md)

# DToken-Go Documentation Center

## Navigation

### Quick Start

- [5-Minute Quick Start](tutorial/quick-start.md) - Fastest way to get started

### Core Capabilities

- [Authentication](guide/core/authentication.md) - Login, logout, token management
- [Session And Terminal Management](guide/core/session-terminal.md) - Session, terminals, multi-terminal login, and exit operations
- [Concurrent Login Policy](guide/core/concurrency-login.md) - Token sharing, max login count, and multi-terminal policy
- [Permission Verification](guide/core/permission.md) - Permission system, wildcard usage
- [AccessProvider](guide/core/access-provider.md) - Dynamic permission and role resolution
- [Disable System](guide/core/disable.md) - Account, service, level, and device disable
- [Multi-Auth Systems](guide/core/multi-auth.md) - Isolate multiple auth systems with AuthType
- [Token Styles](guide/core/token-style.md) - UUID, random string, JWT, and other Token generation styles
- [Event Listener](guide/core/listener.md) - Event system usage guide

### Security And Protocols

- [Nonce Anti-Replay](guide/security/nonce.md) - Prevent replay attacks
- [OAuth2 Authorization](guide/security/oauth2.md) - OAuth2 authorization code flow
- [SSO](../sso/README.md) - Optional SSO module, Ticket, shared token, and remote session modes
- [Refresh Token](guide/security/refresh-token.md) - Token refresh mechanism
- [JWT Integration](guide/security/jwt.md) - JWT token configuration and usage
- [Advanced Features](guide/security/advanced-features.md) - SSO, Ticket, short key, Token Introspection, and more

### Integration And Components

- [Framework Integration](guide/integration/framework-integration.md) - core API plus `integrations/*` middleware usage
- [Annotations](guide/integration/annotation.md) - Annotation-style middleware guide
- [Component Ecosystem](guide/integration/component-ecosystem.md) - Storage, codec, logger, Token generator, and goroutine pool
- [Redis Storage](guide/integration/redis-storage.md) - Redis storage configuration guide
- [Core Flow Testing](guide/integration/core-flow-testing.md) - Gin real-flow tests and Redis-backed test notes

### Reference

- [Core API Cheatsheet](guide/reference/core-api-cheatsheet.md) - Common global API calls
- [Configuration Example](guide/reference/configuration.md) - Common Builder configuration options
- [API Stability](guide/reference/api-stability.md) - Public API compatibility and versioning notes
- [DToken API](api/dtoken.md) - Complete global utility API reference

### Design Documentation

- [Architecture Design](design/architecture.md) - System architecture and data flow
- [Auto-Renewal Design](design/auto-renew.md) - Asynchronous renewal mechanism
- [Modular Design](design/modular.md) - Module organization strategy

## Example Projects

- [quick_start](../examples/quick_start/) - Quick start example
- [gin](../examples/gin/) - Gin integration example
- [gin_core_app](../tests/gin_core_app/) - Gin core flow testing app
- [gf](../examples/gf/) - GoFrame integration example
- [beego](../examples/beego/) - Beego integration example
- [echo](../examples/echo/) - Echo integration example
- [fiber](../examples/fiber/) - Fiber integration example
- [chi](../examples/chi/) - Chi integration example
- [hertz](../examples/hertz/) - Hertz integration example
- [kratos](../examples/kratos/) - Kratos integration example

## External Resources

- [GitHub Repository](https://github.com/Zany2/dtoken-go)

---

**dtoken-go**
