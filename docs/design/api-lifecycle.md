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

Root and framework builders register and return Auth through Build; MustBuild panics on failure. BuildInstance and MustBuildInstance create unregistered instances. Auth.Manager() exposes the shared engine; New(existingManager) creates another facade over that engine, not an independent copy. Closing any wrapper makes every wrapper unavailable. IsClosed reports this state, auth operations return ErrManagerNotFound, and Config, Storage, Codec, and EventManager return nil.

Obtain Manager from a built Auth. Its zero value is unsupported. Do not mutate runtime configuration or replace components after serving requests; build another instance instead.

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
- Root and Auth operations ignore the AuthType option field for routing. That field is retained for source compatibility; select the engine through AuthType(name) or a fixed Auth instance.
- Duplicate Build calls return `errors.ErrManagerAlreadyRegistered` and preserve the original instance. Only one concurrent build wins; a losing assembled candidate closes its owned resources. Mark shared components as caller-owned.
- For explicit replacement, use BuildInstance followed by dtoken.SetAuth. SetAuth, SetManager and BuildAndSetManager perform explicit replacement and close the old engine. Setting the same instance again does not close it.
- Instance Close immediately removes entries still pointing to its engine before closing it. Closing an old instance cannot remove its replacement.
- `dtoken.Close()` closes the default namespace; `dtoken.AuthType("user").Close()` closes the current user namespace; `dtoken.CloseAll()` closes all current registered instances. Repeated calls are safe. CloseAll excludes unregistered instances and does not prevent later builds.

WithAuth binds a concrete engine and does not follow replacement. WithManagerResolver(dtoken.GetManager) selects through the registry per request. An established request scope retains its original engine. Coordinate routing and drain requests before closing the old engine.

Migration: remove redundant SetAuth after Build; change builds that require independent instances with the same namespace to BuildInstance. Obtain the advanced Manager with auth.Manager(). Closing instances and the registry follows shared ownership; individual Close calls do not require an additional CloseAll.

The library does not install signal handlers or call os.Exit. The application handles shutdown signals, stops accepting requests, waits for in-flight handlers, and then calls CloseAll. Forced process termination cannot guarantee cleanup.

## Capability contracts

contract/nonce, contract/oauth2, contract/ticket, and contract/shortkey define service interfaces and data types. HTTP contexts and framework adapters depend on these contracts, not on concrete capability implementations.

The bundled capability packages implement these interfaces with compile-time assertions. Their existing public data names alias the contract types, preserving source compatibility and serialized fields.

Capabilities accepts custom implementations. Missing and typed-nil bindings retain disabled-module errors or false for boolean checks. The request-context method `dCtx.WithCapabilities(nil)` preserves bindings; `dCtx.WithCapabilities(&requestcontext.Capabilities{})` clears them. Framework options instead take a value, for example `gindt.WithCapabilities(requestcontext.Capabilities{Nonce: service})`. Rebinding a request adapter preserves services, while switching authentication instances clears capability bindings.

Standalone Service lifetimes are caller-managed. Supplying an Auth as the event emitter establishes event delivery only; callers must keep borrowed storage and codecs available. HTTP middleware additionally validates the bound engine lifecycle.

Contracts and bundled capabilities remain part of the root module release. This separates package dependencies and runtime composition without adding separate versions or a dynamic plugin mechanism.

## Public lifecycle example

<!-- dtoken-compile: lifecycle -->
```go
package main

import (
	"fmt"
	"github.com/Zany2/dtoken-go"
)

func main() {
	auth, err := dtoken.NewBuilder().IsPrintBanner(false).Build()
	if err != nil {
		panic(err)
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
    "context"
    "errors"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/Zany2/dtoken-go"
)

func main() {
    dtoken.NewBuilder().IsPrintBanner(false).MustBuild()
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
        panic(err)
    }
    if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
        panic(serveErr)
    }
}
```
