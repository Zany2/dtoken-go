# Event Listener Guide

[中文文档](../core/listener_zh.md) | English

## Current Status

The project already contains a complete internal event system under `core/listener`.

It is used for authentication and lifecycle events, access checks and changes, refresh tokens, Nonce, Ticket, ShortKey, and OAuth2 flows.

## Accessing the Event Manager

The current version exposes the event manager through the global facade, the instance facade, and `manager.Manager`:

```go
globalEventMgr, err := dtoken.GetEventManager("user")
if err != nil {
    // Handle the lookup error.
}
instanceEventMgr := auth.EventManager()
coreEventMgr := mgr.GetEventManager()
```

After obtaining `*listener.Manager`, use its registration APIs such as `RegisterFunc` and `RegisterFuncWithConfig`. The global facade, instance facade, and core manager are alternative access paths; choose the manager that belongs to your application.

## Internal Event Types

The built-in events defined in `core/listener/consts.go` are:

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

The payload is defined in `core/listener/listener.go`:

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

The trigger methods take an event payload snapshot. Each listener receives its own copy of the scalar fields and the top-level `Extra` map. This is a shallow copy: nested maps, slices, pointers, and other reference values inside `Extra` remain shared and must be treated as read-only by filters and listeners.

The manager's diagnostic trigger log contains only the event type, auth type, timestamp, and listener count. It does not write `LoginID`, device fields, tokens, or `Extra`; application listeners and custom panic handlers remain responsible for protecting the payload they receive.

## Extra Field Constants

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

The `core/listener` package is public and can also be used independently.

### Create a Manager

```go
import (
    "github.com/Zany2/dtoken-go/core/listener"
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

- `Trigger` dispatches on the current goroutine. Synchronous listeners run inline; asynchronous listeners are started without being awaited.
- `TriggerAsync` snapshots the payload before returning, schedules the whole dispatch asynchronously, and returns immediately.
- `TriggerSync` dispatches on the current goroutine and waits for the listeners started directly by that dispatch. It does not wait for unrelated asynchronous work or background descendants started through `Trigger` or `TriggerAsync`; a nested `TriggerSync` remains part of the calling listener's completion.
- `Wait` waits until the manager has no accepted asynchronous dispatch or listener tasks remaining.

Listeners for a concrete event and `EventAll` are merged and sorted by descending priority. For asynchronous listeners, priority controls the start traversal order only; goroutine scheduling does not guarantee callback completion order.

```go
eventMgr.Trigger(data)
eventMgr.TriggerAsync(data)
eventMgr.TriggerSync(data)
```

## Advanced Features

### Global Filters

```go
eventMgr.AddFilter(func(data *listener.EventData) bool {
    return data.AuthType == "dtoken:"
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

## Related Documentation

- [Authentication Guide](../core/authentication.md)
- [Permission Management](../core/permission.md)
- [OAuth2 Guide](../security/oauth2.md)
