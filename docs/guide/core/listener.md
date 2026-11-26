# Event Listener Guide

Fragment aliases: dt = github.com/Zany2/dtoken-go, listener = github.com/Zany2/dtoken-go/event.

[中文文档](listener_zh.md) | English

Independent capabilities publish to an Auth event manager when created with NewService(manager, auth) and called through Service. Direct capability Manager calls bypass this service wrapper. Register application listeners before accepting requests and close the engine through Auth.Close outside listener callbacks.

## Current Status

The project already contains a complete internal event system under `event`.

It is used for authentication and lifecycle events, access checks and changes, refresh tokens, Nonce, Ticket, ShortKey, and OAuth2 flows.

## Accessing the Event Manager

Obtain the event manager through the root registry, a selected Auth, or the advanced `dt.Manager`:

```go
registryEventMgr, err := dt.GetEventManager("user")
if err != nil {
    // Handle the lookup error.
}
instanceEventMgr := auth.EventManager()
coreEventMgr := mgr.GetEventManager()
```

After obtaining `*listener.Manager`, use its registration APIs such as `RegisterFunc` and `RegisterFuncWithConfig`. `dt.AuthType("user").EventManager()` also resolves the selected system; it returns nil if unavailable, while `dt.GetEventManager("user")` reports an error. All paths return the event manager owned by the selected engine.

## Internal Event Types

The built-in events defined in `event/consts.go` are:

| Event | Description |
|------|------|
| `EventLogin` | login |
| `EventLogout` | logout |
| `EventKickout` | kickout |
| `EventActiveTimeout` | inactive timeout |
| `EventReplace` | replace |
| `EventDisable` | account disable |
| `EventUntie` | account untie |
| `EventRenew` | token renew |
| `EventCreateSession` | session create |
| `EventDestroySession` | session destroy |
| `EventPermissionCheck` | permission check |
| `EventPermissionChange` | permission change |
| `EventRoleCheck` | role check |
| `EventRoleChange` | role change |
| `EventDisableService` | service disable |
| `EventUntieService` | service untie |
| `EventDisableDevice` | device disable |
| `EventUntieDevice` | device untie |
| `EventRefreshTokenCreate` | refresh token create |
| `EventRefreshTokenRotate` | refresh token rotate |
| `EventRefreshTokenRevoke` | refresh token revoke |
| `EventNonceGenerate` | Nonce generate |
| `EventNonceVerify` | Nonce verify |
| `EventTicketCreate` | Ticket create |
| `EventTicketValidate` | Ticket validate |
| `EventTicketConsume` | Ticket consume |
| `EventTicketRevoke` | Ticket revoke |
| `EventShortKeyCreate` | ShortKey create |
| `EventShortKeyConfirm` | ShortKey confirm |
| `EventShortKeyValidate` | ShortKey validate |
| `EventShortKeyConsume` | ShortKey consume |
| `EventShortKeyRevoke` | ShortKey revoke |
| `EventOAuth2ClientRegister` | OAuth2 client register |
| `EventOAuth2ClientUnregister` | OAuth2 client unregister |
| `EventOAuth2CodeGenerate` | OAuth2 code generate |
| `EventOAuth2TokenIssue` | OAuth2 token issue |
| `EventOAuth2TokenRefresh` | OAuth2 token refresh |
| `EventOAuth2TokenValidate` | OAuth2 token validate |
| `EventOAuth2TokenRevoke` | OAuth2 token revoke |
| `EventAll` | wildcard |

## EventData Structure

The payload is defined in `event/listener.go`:

```go
type EventData struct {
    Event     Event
    AuthType  string
    LoginID   string
    Device    string
    DeviceID  string
    Token     string
    Extra     map[string]any
    Timestamp int64
}
```

The trigger methods take an event payload snapshot. Each listener receives its own copy of the scalar fields, the top-level `Extra` map, and any values of type `[]string` stored directly in that map. Copying is not recursive: nested maps, slices other than these directly stored `[]string` values, pointers, and other reference values remain shared and must be treated as read-only by filters and listeners.

The manager's diagnostic trigger log contains only the event type, auth type, timestamp, and listener count. It does not write `LoginID`, device fields, tokens, or `Extra`; application listeners and custom panic handlers remain responsible for protecting the payload they receive.

## Extra Field Constants

For Ticket and ShortKey events, `ExtraKeyTTL` is the remaining lifetime when the event is created. It uses the credential's precise `ExpiresAt` when available, rounds positive fractional seconds up, and returns `0` after expiry. Legacy credentials without `ExpiresAt` use `CreateTime + ExpiresIn`. Asynchronous listeners receive this event-time snapshot.

Current extra keys include:

- `ExtraKeyPermission`
- `ExtraKeyPermissions`
- `ExtraKeyRole`
- `ExtraKeyRoles`
- `ExtraKeyLogic`
- `ExtraKeyResult`
- `ExtraKeyAction`
- `ExtraKeyShared`
- `ExtraKeyService`
- `ExtraKeyLevel`
- `ExtraKeyTokenType`
- `ExtraKeyClientID`
- `ExtraKeyUserID`
- `ExtraKeyScopes`
- `ExtraKeySource`
- `ExtraKeySourceApp`
- `ExtraKeyTargetApp`
- `ExtraKeyScene`
- `ExtraKeyStatus`
- `ExtraKeyTTL`
- `ExtraKeyRefreshToken`
- `ExtraKeyGrantType`

## Event Manager APIs

The `event` package is public and can also be used independently.

### Create a Manager

```go
import (
    listener "github.com/Zany2/dtoken-go/event"
)

eventMgr := listener.NewManager()
```

### Register a Listener

```go
id := eventMgr.RegisterFunc(listener.EventLogin, func(data *listener.EventData) {
    println("login:", data.LoginID)
})

_ = id
```

### Register With Config

```go
id := eventMgr.RegisterFuncWithConfig(
    listener.EventLogin,
    func(data *listener.EventData) {
        println("login:", data.LoginID)
    },
    listener.ListenerConfig{
        Async:    true,
        Priority: 100,
        ID:       "login-audit",
    },
)
```

Listener IDs are globally unique within one manager. Registration with a duplicate explicit ID returns an empty string; automatically generated IDs skip values that are already in use.

`Register` and `RegisterFunc` use `Async: true` and priority `0` by default. Use `RegisterFuncWithConfig` when the callback must run inline or needs a different priority.

### Unregister

```go
ok := eventMgr.Unregister("login-audit")
_ = ok
```

### Trigger Events

- `Trigger` dispatches on the current goroutine. Synchronous listeners run inline; accepted pool listeners are not awaited; absent or rejecting pools execute them inline.
- `TriggerAsync` snapshots the payload before returning, fills a zero `Timestamp` with the submission time, and submits the whole dispatch to the event pool. Without a pool or after rejection, direct listeners complete synchronously. Explicit timestamps are preserved and the caller's payload is not modified.
- `TriggerSync` executes direct listeners in priority order on the caller, ignoring their Async setting. It does not wait for unrelated asynchronous work or background descendants started through `Trigger` or `TriggerAsync`; a nested `TriggerSync` remains part of the calling listener's completion.
- `Wait` waits until the manager has no accepted asynchronous dispatch or listener tasks remaining.

Listeners for a concrete event and `EventAll` are merged and sorted by descending priority. For asynchronous listeners, priority controls the start traversal order only; goroutine scheduling does not guarantee callback completion order.

Sorting and merging happen when registrations change. Dispatch reuses immutable lists instead of copying, merging, and sorting them on every event. Equal priorities preserve registration order within each group, with concrete-event listeners before `EventAll`. Registering, unregistering, or clearing listeners inside a callback does not change an already captured dispatch snapshot; subsequent snapshots see the new configuration. Filter lists follow the same rule, and callbacks still run outside the manager lock.

### Skipping Events Without Observers

Disabled events are skipped. An enabled event skips payload copying and asynchronous submission only when it has no concrete or wildcard listeners, no filters, statistics are disabled, and no logging is needed. Filter-only, statistics-only, and logging-only events still run. The built-in `contract.NopLogger` is recognized as a no-op; custom loggers are always treated as observers.

Use `eventMgr.NeedsDispatch(event)` before constructing expensive payloads to check all these conditions. `HasListeners` continues to check direct registrations for the specified event only. The engine uses the new check before constructing EventData, copying Extra, and submitting event work. Payloads prepared by callers before invocation still incur their own construction cost.

This check reflects current configuration and does not reserve listeners. Events skipped at invocation are not replayed when listeners or statistics are enabled later. Accepted asynchronous dispatch still captures configuration when the worker runs, while payloads and default timestamps are captured at submission. Configure observers before accepting requests.

Merged lists are maintained only for events with registered concrete listeners; triggering arbitrary custom event names does not grow the cache. Registration changes pay the copying and merging cost so frequent dispatches can read prepared lists. Per-listener payload isolation is retained.

```go
eventMgr.Trigger(data)
eventMgr.TriggerAsync(data)
eventMgr.TriggerSync(data)
```

## Advanced Features

### Optional Independent Event Pool

Without an event pool, event tasks run synchronously. `SetAsyncPool(pool contract.Pool)` optionally reuses the existing pool contract and implementations. Create a separate instance for events; do not pass the engine's renewal pool. Configure it before accepting requests:

```go
import eventpool "github.com/Zany2/dtoken-go/pool/ants"

cfg := eventpool.DefaultRenewPoolConfig()
cfg.SetNonBlocking(true)
eventPool, err := eventpool.NewRenewPoolManagerWithConfig(cfg)
if err != nil {
    panic(err)
}
auth.EventManager().SetAsyncPool(eventPool)
```

The existing constructor name includes `Renew`, but its implementation accepts general tasks and can be instantiated separately for events. A standalone `listener.NewManager()` supports the same setter. No new pool is created automatically, and the engine's `AsyncEvent` dispatch configuration and renewal pool wiring remain unchanged.

- `Trigger` submits asynchronous listeners to the event pool; synchronous listeners still run inline.
- `TriggerAsync` submits its dispatch to the event pool, and its asynchronous listeners also use that pool.
- `TriggerSync` executes direct listeners inline without submitting or waiting for pool tasks, avoiding nested waits on its own worker.
- A rejected submission runs inline, preserving delivery without spawning extra goroutines. Callback time adds to caller latency; multiple callers may still execute fallbacks concurrently.
- `SetAsyncPool(nil)` switches subsequent submissions to synchronous execution; already accepted tasks stay with their original pool and remain tracked by `Wait`.

The injected pool must provide concurrency-safe, nonblocking submission: accept a task for asynchronous execution exactly once, or return an error without executing it. Do not use blocking submission, silently discard accepted tasks, or run tasks inline. Nested dispatch can submit more work from a pool worker, so waiting for capacity can deadlock. The bundled ants pool requires `NonBlocking: true`; setting it to false returns a configuration error during construction.

The application owns the event pool; the event manager never calls its `Stop`. At shutdown, stop and drain request/event producers, call `auth.Close()` outside callbacks, then call `eventPool.Stop()`. For a standalone event manager, use `eventMgr.Wait()` before stopping the pool. Keep old pools alive until their accepted tasks drain when replacing a pool.

### Global Filters

```go
eventMgr.AddFilter(func(data *listener.EventData) bool {
    return data.AuthType == "auth:"
})
```

If the filter returns `false`, the event will not be dispatched any further.

Panics from filters are recovered, passed to the panic handler, and reject the current event.

### Statistics

```go
eventMgr.EnableStats(true)

stats := eventMgr.GetStats()
println(stats.TotalTriggered)
```

### Panic Handling

```go
eventMgr.SetPanicHandler(func(event listener.Event, data *listener.EventData, recovered any) {
    println("listener panic:", event, recovered)
})
```

Panics from listeners and filters are recovered and passed to this handler. A panic in a custom panic handler is contained and reported by the manager logger.

### Enable or Disable Events

```go
eventMgr.DisableEvent(listener.EventRenew, listener.EventPermissionCheck)
eventMgr.EnableEvent(listener.EventLogin, listener.EventLogout)
eventMgr.EnableEvent() // no args means enable all
```

`EnableEvent(events...)` replaces the current allow-list. `DisableEvent(events...)` disables only the selected events without changing unrelated built-in or custom events. `EventAll` can be used with either method to target all events.

### Wait For Async Listeners

```go
eventMgr.Wait()
```

Use `Wait` as an external lifecycle barrier: first stop callers from submitting new events, then call `Wait` during shutdown before closing listener dependencies. Do not call it from inside a listener, because that listener may itself be part of the tracked asynchronous work. New work submitted after `Wait` returns is not covered by the completed wait.

## Logic Constants

```go
listener.LogicAnd
listener.LogicOr
```

These usually appear in the `Extra` payload of permission and role check events.

## Practical Recommendation

Obtain the event manager from the facade or Manager instance and register application listeners before serving traffic. During shutdown, stop event producers before using `Wait` to drain accepted asynchronous work.

The public `Auth.Emit` and `Manager.Emit` entry points ignore new events once the authentication engine starts closing, regardless of `AsyncEvent`. Already accepted asynchronous events and internal renewal completion events still drain before resource release. This does not close a separately held `event.Manager`; callers that use its `Trigger` methods directly must stop their own event producers before shutdown.

Before calling `Manager.CloseManager`, stop and drain application requests, then call it from outside tasks and listeners. It rejects new asynchronous tasks, stops background status logging, drains accepted tasks and asynchronous listeners, and finally releases its owned Pool, Storage, and Logger. Caller-owned components remain open. Accepted renewal tasks dispatch their renewal event within the same task, avoiding both nested pool submission and event loss from rejected submissions during shutdown. Each listener's own `Async` setting still applies.

Manager recovers and logs task panics, including the synchronous fallback used when no pool is available or submission fails. Recovery isolates the failure; it does not retry the task or roll back completed storage operations.

## Related Documentation

- [Authentication Guide](../core/authentication.md)
- [Permission Management](../core/permission.md)
- [OAuth2 Guide](../security/oauth2.md)
