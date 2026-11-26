# Public API, capability contracts, and lifecycle

English | [中文文档](api-lifecycle_zh.md)

This document owns public API, resource sharing, and shutdown contracts. See [architecture](architecture.md) for dependencies, [source layout](source-layout.md) for code locations, and [modular design](modular.md) for release boundaries.

The public API has two supported levels: applications use the root Auth facade; custom adapters may use root Manager, ManagerOption, and Strategy. Manager is a supported advanced API, not a transitional compatibility alias. Package paths under internal are not public API.

Type aliases exposed through root engine_api.go and their exported methods remain compatibility commitments. Shared models such as TokenInfo, Session, and TerminalInfo are declared in contract and aliased by the facade and engine. Review public additions and signature changes across the instance facade, advanced APIs, capability contracts, and framework exports.

## Entry points and declaration ownership

| Public entry or type | Declaration source | Usage contract |
| --- | --- | --- |
| Auth, Builder, instance operation options | Root auth.go, builder.go, options.go | Preferred application entry points |
| Config | config.Config, aliased by the root | Core configuration is maintained in config |
| TokenInfo, Session, TerminalInfo, disable models, AccessProvider | contract, aliased by root types.go and the engine | Review consumers and serialization compatibility when changing shared models |
| Manager, Strategy, ManagerLoginOptions, RefreshTokenPair, and other advanced types | internal/engine, aliased by root engine_api.go | Applications name these types through the root, not the implementation path |
| Component factories, Components, ComponentFactories | internal/builder, aliased by root engine_api.go | Support custom component assembly |
| Capability service interfaces and data types | contract/nonce, contract/oauth2, contract/ticket, contract/shortkey | HTTP depends on contracts; implementations reuse their data types |
| Framework DTokenContext and Capabilities | httpauth/context, aliased by framework export.go | Frameworks retain native request entry points while sharing request types |

Root LoginOptions and ManagerLoginOptions serve different API levels and are not aliases of one type. Advanced results such as RefreshTokenPair are not basic models declared in contract/model.go either. Follow actual aliases when organizing types; similar names alone do not justify copying, merging, or moving declarations.

## Instances and engines

Root and framework builders register and return Auth through Build. BuildInstance creates unregistered instances. Both return (*Auth, error), leaving failure handling to the caller. Auth.Manager() exposes the shared engine and returns nil once shutdown begins; New(existingManager) creates another facade over that engine, not an independent copy. Starting shutdown through any wrapper makes every wrapper unavailable. IsClosed reports this state, auth operations return ErrManagerNotFound, and Config, Storage, Codec, and EventManager return nil.

A fixed Auth keeps its Manager pointer unchanged after construction; an AuthType selector stores only the namespace and resolves the registry on each call. Manager owns shutdown state, with no separate close flag in Auth. Repeated Close calls, including calls through different fixed facades, wait for the same engine cleanup to finish.

An advanced Manager retained before shutdown follows the same business-call contract: once shutdown starts, login, validation, authorization, Session, terminal, disable, and refresh-token methods return `ErrManagerNotFound`, or `false` for boolean checks. This check precedes argument validation and storage access. Internal maintenance accepted before shutdown still completes, including disable-state rechecks and renewal events, before resources are released. Synchronous requests remain application-managed and must be drained before closing; the entry check does not wait for in-flight calls. Configuration and strategy snapshots remain inspectable, but component getters do not extend resource lifetimes.

Obtain Manager from a built Auth. Its zero value is unsupported. Manager.GetConfig() and Auth.Config() return independent snapshots; AuthType, KeyPrefix, and other configuration are fixed at construction. To change configuration, build a new instance and replace the old one according to the lifecycle contract.

`Manager.GetStrategy()` also returns a snapshot. Changing it cannot replace live Session creation, permission matching, or role matching. Supply custom hooks through `WithStrategy` during construction. Captured closure state is not copied; its concurrency safety remains the caller's responsibility.

## Resource ownership

| Situation | Ownership |
| --- | --- |
| Successful build | Storage, logger, and pool are engine-owned by default, including explicitly supplied components |
| Failed assembly | Factory-created components are cleaned up; explicitly supplied components remain caller-owned |
| Lost registration race | The assembled candidate closes resources according to ComponentOwnership; the registered instance remains unchanged |
| WithComponentOwnership | Sets the complete policy; false fields remain caller-owned |
| Builder.Clone | Copies configuration but shares injected components, providers, and callback state |
| New(existingManager) | Shares the original engine lifetime |
| Capability Service | Borrows components; the event emitter does not transfer ownership |

Closing an engine is idempotent. There is no cross-engine reference counting for shared components. When several Auth instances share storage, a logger, or a pool, set its ownership flag to false and close it from the application after all users finish. Use factories returning fresh components for independent ownership.

This fragment uses `dtoken` for the root package and `memory` for `storage/memory`:

```go
shared := memory.NewStorage()
builder := dtoken.NewBuilder().
    Storage(shared).
    UseManagerOption(dtoken.WithComponentOwnership(dtoken.ComponentOwnership{
        Storage: false,
        Logger: true,
        Pool: true,
    }))
first, err := builder.AuthType("user").Build()
if err != nil { panic(err) }
defer first.Close()
second, err := builder.Clone().AuthType("admin").Build()
if err != nil { panic(err) }
defer second.Close()
```

Auth.Close stops maintenance, waits for accepted asynchronous work, and releases owned resources. It does not drain HTTP handlers. Stop accepting requests, wait for handlers, close Auth instances, and finally close caller-owned resources.

## Global registration

The root owns the manager registry. Build registers automatically and needs no separate SetAuth:

```go
userAuth, err := dtoken.NewBuilder().AuthType("user").Build()
if err != nil { panic(err) }
defer userAuth.Close()

token, err := userAuth.LoginID(ctx, "10001")
if err != nil { panic(err) }
err = dtoken.AuthType("user").CheckLogin(ctx, token)
```

- A builder without AuthType registers the default `auth:` namespace. Package functions such as `dtoken.Login` and `dtoken.IsLogin` always use that namespace and match the Auth method signatures.
- `dtoken.AuthType("user")` is a selector that resolves the registry on every operation, including after explicit replacement. Missing names return ErrManagerNotFound or false for boolean checks; they never fall back to the default.
- Operation options contain business parameters and have no AuthType field. Select the engine through root default calls, AuthType(name), or a fixed Auth instance.
- Duplicate Build calls return `errors.ErrManagerAlreadyRegistered` and preserve the original instance. Only one concurrent build wins; a losing assembled candidate closes its owned resources. Mark shared components as caller-owned.
- For explicit replacement, use BuildInstance followed by dtoken.SetAuth. SetAuth and SetManager return an error for invalid registration; successful replacement closes the old engine. Setting the same instance again does not close it.
- Instance Close marks its shared engine as closing and removes entries still pointing to it, then waits for resource cleanup. Closing an old instance cannot remove its replacement.
- `dtoken.Close()` closes the default namespace; `dtoken.AuthType("user").Close()` closes the current user namespace; `dtoken.CloseAll()` closes all current registered instances. Repeated calls are safe. CloseAll excludes unregistered instances and does not prevent later builds.

The registry retains a write mutex for registration, replacement, and removal; ordinary lookups do not acquire it. Replacement and removal mark old managers as closing and reject new async tasks before releasing the registry lock. Accepted tasks and listeners are drained and resources released outside that lock. There is no separate retiring set: each engine's shutdown state prevents its registration again.

WithAuth binds a concrete engine and does not follow replacement. WithManagerResolver(dtoken.GetManager) selects through the registry per request. An established request scope retains its original engine. Coordinate routing and drain requests before replacing or closing the old engine; registry locking and shutdown flags do not wait for in-flight synchronous operations.

Migration: remove redundant SetAuth after Build; change builds that require independent instances with the same namespace to BuildInstance. Obtain the advanced Manager with auth.Manager(). Closing instances and the registry follows shared ownership; individual Close calls do not require an additional CloseAll.

The library does not install signal handlers or call os.Exit. The application handles shutdown signals, stops accepting requests, waits for in-flight handlers, and then calls CloseAll. Forced process termination cannot guarantee cleanup.

## Capability contracts

contract/nonce, contract/oauth2, contract/ticket, and contract/shortkey define service interfaces and data types. HTTP contexts and framework adapters depend on these contracts, not on concrete capability implementations.

The bundled capability packages implement these interfaces with compile-time assertions. Their existing public data names alias the contract types, preserving source compatibility and serialized fields.

Capabilities accepts custom implementations. Missing and typed-nil bindings retain disabled-module errors or false for boolean checks. The request-context method `dCtx.WithCapabilities(nil)` preserves bindings; `dCtx.WithCapabilities(&requestcontext.Capabilities{})` clears them. Framework options instead take a value, for example `gindt.WithCapabilities(requestcontext.Capabilities{Nonce: service})`. Rebinding a request adapter preserves services, while switching authentication instances clears capability bindings.

Standalone Service lifetimes are caller-managed. Supplying an Auth as the event emitter establishes event delivery only; callers must keep borrowed storage and codecs available. When DTokenContext has a bound Manager, direct Nonce, OAuth2, Ticket, and ShortKey operations also return ErrManagerNotFound once shutdown starts, or false for boolean checks. A context without a Manager can still use explicitly bound standalone capability services. Capability operations requiring the current login identity still need valid authentication bindings.

Operations that read the current request token or access cookies require an open Manager and a valid request adapter: a missing or closed Manager returns ErrManagerNotFound; a missing adapter, including a typed nil, returns ErrInvalidParam. Token extraction without an error result returns an empty string, boolean authentication checks return false, and direct cookie writes do nothing. Cookie login validates bindings before writing authentication state. Operations using explicit parameters, such as Auth.Login and Refresh, only require an open Manager; they do not require a request adapter.

Ticket and ShortKey compare AuthType using the same identity as storage keys: `user` and `user:` are equivalent, and existing payloads need no migration. Creation results retain the caller's AuthType spelling. Comparison ignores only one trailing colon, preserving whitespace and other distinct names. When composing capabilities, prefer AuthType and KeyPrefix from Auth.Config().

Contracts and bundled capabilities remain part of the root module release. This separates package dependencies and runtime composition without adding separate versions or a dynamic plugin mechanism.

## Public lifecycle example

<!-- dtoken-compile: lifecycle -->
```go
package main

import (
    "log"
	"fmt"
	"github.com/Zany2/dtoken-go"
)

func main() {
	auth, err := dtoken.NewBuilder().IsPrintBanner(false).Build()
	if err != nil {
		log.Print(err)
		return
	}
	defer auth.Close()
	shared := dtoken.New(auth.Manager())
	shared.Close()
	fmt.Println(auth.IsClosed()) // true
}
```

## Application-managed shutdown signals

This example drains HTTP handlers before deferred CloseAll runs. Applications with hijacked connections such as WebSockets must also wait for those handlers. Do not close an engine synchronously from its own async task or event listener.

<!-- dtoken-compile: graceful-shutdown -->
```go
package main

import (
    "log"
    "context"
    "errors"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/Zany2/dtoken-go"
)

func main() {
    if _, err := dtoken.NewBuilder().IsPrintBanner(false).Build(); err != nil {
        log.Print(err)
        return
    }
    defer dtoken.CloseAll()

    stopped, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    server := &http.Server{Addr: ":8080", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if err := dtoken.CheckLogin(r.Context(), r.Header.Get("Authorization")); err != nil {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    })}
    done := make(chan error, 1)
    go func() { done <- server.ListenAndServe() }()
    var serveErr error
    select {
    case serveErr = <-done:
    case <-stopped.Done():
    }
    if err := server.Shutdown(context.Background()); err != nil {
        log.Print(err)
        return
    }
    if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
        log.Print(serveErr)
        return
    }
}
```
