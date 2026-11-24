# Auth And Root Facade API

English | [中文文档](auth_zh.md)

The root package `github.com/Zany2/dtoken-go` is the preferred application entry point. `NewBuilder().Build()` automatically registers and returns `*dtoken.Auth`; `BuildInstance()` creates an unregistered instance. See [quick start](../tutorial/quick-start.md) for complete initialization.

## Three Calling Styles

| Call | Selected engine |
| --- | --- |
| `auth.CheckLogin(ctx, token)` | The fixed instance returned by the builder |
| `dtoken.CheckLogin(ctx, token)` | The currently registered default `auth:` instance |
| `dtoken.AuthType("user").CheckLogin(ctx, token)` | The currently registered user instance |

Root business functions match Auth method signatures, for example `dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "10001"})`. Omitting Builder.AuthType registers the default namespace. Missing explicit namespaces return ErrManagerNotFound, or false for boolean checks, without falling back. Selectors resolve each operation; fixed instances do not follow replacement.

Build / MustBuild register automatically. Duplicate Build returns errors.ErrManagerAlreadyRegistered; MustBuild panics on failure. BuildInstance / MustBuildInstance do not register and allow independent instances with the same namespace. For explicit replacement, use BuildInstance then dtoken.SetAuth; replacement closes the original engine. Ordinary Build needs no SetAuth.

## Entry Points And Options

| Operation | Instance call |
| --- | --- |
| Login by account | `auth.LoginID(ctx, loginID)` |
| Login with device, lifetime, and extra data | `auth.Login(ctx, dtoken.LoginOptions{...})` |
| Validate login | `auth.CheckLogin(ctx, token)` / `auth.IsLogin(ctx, token)` |
| Resolve account | `auth.GetLoginID(ctx, token)` |
| Inspect without renewal | `auth.IntrospectToken(ctx, token)` |
| Logout, kick out, or replace by token | `auth.LogoutByToken` / `auth.KickoutByToken` / `auth.ReplaceByToken` |
| Terminate by criteria | `auth.Logout` / `auth.Kickout` / `auth.Replace` with `dtoken.LogoutOptions` |
| Mutate permissions and check one permission | `auth.AddPermissions` / `auth.RemovePermissions` / `auth.CheckPermission` with `dtoken.PermissionOptions` |
| Mutate roles and check one role | `auth.AddRoles` / `auth.RemoveRoles` / `auth.CheckRole` with `dtoken.RoleOptions` |
| Boolean permission and role checks | `auth.HasPermission(ctx, loginID, permission)` / `auth.HasRole(ctx, loginID, role)` |
| Disable an account, service, or device | `auth.Disable` / `auth.DisableService` / `auth.DisableDevice` with their typed options |
| Query and persist session data | `auth.GetSession` / `auth.SetSessionValue` / `auth.GetSessionValue` |
| Refresh tokens | `auth.LoginWithRefreshToken` / `auth.LoginWithRefreshTokenOptions` / `auth.RefreshToken` |

Instance methods use their own authentication namespace; an option's `AuthType` does not switch the instance to another engine. Use another Auth or dtoken.AuthType(name) for another namespace. That option field remains for legacy global option calls.

## Login And Permission Example

This fragment assumes auth is already built and the user identity has been verified. ctx is the current request context.

```go
token, err := auth.Login(ctx, dtoken.LoginOptions{
    LoginID: "10001",
    Device: "web",
    DeviceID: "browser-1",
})
if err != nil {
    return err
}
if err := auth.AddPermissions(ctx, dtoken.PermissionOptions{
    LoginID: "10001", Permissions: []string{"user:read"},
}); err != nil {
    return err
}
if err := auth.CheckPermission(ctx, dtoken.PermissionOptions{
    Token: token, Permission: "user:read",
}); err != nil {
    return err
}
```

## Differences From Legacy global APIs

`global` resolves instances through a registry; many functions accept an optional trailing authType. Instance method names and parameters can differ, so changing only the call prefix is insufficient.

| Global call | Corresponding instance call |
| --- | --- |
| `global.Login(ctx, id, device, deviceID, authType)` | `auth.Login(ctx, dtoken.LoginOptions{LoginID: id, Device: device, DeviceID: deviceID})` |
| `global.Logout(ctx, token)` | `auth.LogoutByToken(ctx, token)` |
| `global.AddPermissions(ctx, id, values)` | `auth.AddPermissions(ctx, dtoken.PermissionOptions{LoginID: id, Permissions: values})` |
| `global.CheckPermission(ctx, id, value)` | `auth.CheckPermission(ctx, dtoken.PermissionOptions{LoginID: id, Permission: value})` |

See the [global API reference](dtoken.md). HTTP middleware uses instances through `WithAuth(auth)`. Grouped request-context APIs use underlying Manager option types; distinguish root LoginOptions from ManagerLoginOptions when passing options.

## Lifecycle And Extensions

- `auth.Config()` returns a configuration snapshot; editing it does not change the engine.
- `auth.Storage()` and `auth.Codec()` lend components for independent services; keep Auth open while using them.
- `auth.Manager()` exposes the supported advanced API. Name its types through the root package, never internal imports.
- `dtoken.New(auth.Manager())` wraps the same engine. Closing either facade affects every sharer.
- `auth.Close()` unregisters entries still pointing to its engine and releases owned resources. dtoken.Close closes the default, dtoken.AuthType(name).Close closes the selected namespace, and dtoken.CloseAll closes all current registrations. Close unregistered instances separately and drain HTTP requests first.
- Call Nonce, OAuth2, Ticket, and ShortKey through their Service implementations and bind them with `WithCapabilities` for HTTP; they are not Auth methods.

See [API and lifecycle](../design/api-lifecycle.md), [independent capabilities](../guide/security/advanced-features.md), and [framework integration](../guide/integration/framework-integration.md).
