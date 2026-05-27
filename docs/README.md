English | [中文文档](README_zh.md)

# DToken-Go Documentation Center

## 📚 Documentation Navigation

### 🚀 Quick Start

- [5-Minute Quick Start](tutorial/quick-start.md) - Fastest way to get started

### 📖 User Guides

- [Authentication](guide/authentication.md) - Login, logout, token management
- [Session And Terminal Management](guide/session-terminal.md) - Session, terminals, multi-terminal login, and exit operations
- [Concurrent Login Policy](guide/concurrency-login.md) - Token sharing, max login count, and multi-terminal policy
- [Permission Verification](guide/permission.md) - Permission system, wildcard usage
- [AccessProvider](guide/access-provider.md) - Dynamic permission and role resolution
- [Disable System](guide/disable.md) - Account, service, level, and device disable
- [Multi-Auth Systems](guide/multi-auth.md) - Isolate multiple auth systems with AuthType
- [Token Styles](guide/token-style.md) - UUID, random string, JWT, and other Token generation styles
- [Core API Cheatsheet](guide/core-api-cheatsheet.md) - Common global API calls
- [Configuration Example](guide/configuration.md) - Common Builder configuration options
- [Component Ecosystem](guide/component-ecosystem.md) - Storage, codec, logger, Token generator, and goroutine pool
- [Annotations](guide/annotation.md) - Annotation-style middleware guide
- [Event Listener](guide/listener.md) - Event system usage guide
- [JWT Integration](guide/jwt.md) - JWT token configuration and usage
- [Redis Storage](guide/redis-storage.md) - Redis storage configuration guide
- [Framework Integration](guide/framework-integration.md) - core API plus `integrations/*` middleware usage
- [Core Flow Testing](guide/core-flow-testing.md) - Gin real-flow tests and Redis-backed test notes

### 🔒 Security Features

- [Nonce Anti-Replay](guide/nonce.md) - Prevent replay attacks
- [Advanced Features](guide/advanced-features.md) - SSO, Ticket, short key, Token Introspection, and more
- [Refresh Token](guide/refresh-token.md) - Token refresh mechanism
- [OAuth2 Authorization](guide/oauth2.md) - OAuth2 authorization code flow

### 🔧 API Documentation

- [DToken API](api/dtoken.md) - Complete global utility API reference

### 🏗️ Design Documentation

- [Architecture Design](design/architecture.md) - System architecture and data flow
- [Auto-Renewal Design](design/auto-renew.md) - Asynchronous renewal mechanism
- [Modular Design](design/modular.md) - Module organization strategy

## 📖 Example Projects

- [quick_start](../examples/quick_start/) - Quick start example
- [gin](../examples/gin/) - Gin integration example
- [gin_core_app](../tests/gin_core_app/) - Gin core flow testing app
- [gf](../examples/gf/) - GoFrame integration example
- [echo](../examples/echo/) - Echo integration example
- [fiber](../examples/fiber/) - Fiber integration example
- [chi](../examples/chi/) - Chi integration example
- [hertz](../examples/hertz/) - Hertz integration example
- [kratos](../examples/kratos/) - Kratos integration example

## 🔗 External Resources

- [GitHub Repository](https://github.com/Zany2/dtoken-go)

---

**dtoken-go**
