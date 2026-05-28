# AccessProvider

**[中文文档](../core/access-provider_zh.md)**

`AccessProvider` resolves permissions and roles dynamically from business systems. It fits projects where access data lives in a database, RBAC service, configuration center, or third-party permission platform.

## Relationship With Session Access

DToken-Go supports two access sources:

1. Permissions and roles stored in Session through APIs such as `AddPermissions` and `AddRoles`.
2. Permissions and roles returned dynamically by `AccessProvider` or compatible callbacks.

When `AccessProvider` is configured, non-`nil` data returned by the provider takes priority over Session data. If the provider returns `nil`, DToken-Go falls back to Session data.

## AccessSubject

The provider receives an `AccessSubject`:

```go
type AccessSubject struct {
    AuthType string
    LoginID  string
    Device   string
    DeviceID string
    Token    string
}
```

These fields allow access resolution by auth system, account, device type, concrete device, or Token.

## Basic Usage

```go
provider := manager.AccessProviderFunc{
    PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
        if subject.AuthType == "admin:" {
            return []string{"admin:*"}, nil
        }
        return []string{"user:read"}, nil
    },
    RoleFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
        if subject.LoginID == "10001" {
            return []string{"admin"}, nil
        }
        return []string{"user"}, nil
    },
}

mgr, err := defaults.NewBuilder().
    SetStorage(storage).
    SetAccessProvider(provider).
    Build()
```

## Fallback To Session

Returning `nil` means the provider does not handle this query, so DToken-Go should use Session data:

```go
provider := manager.AccessProviderFunc{
    PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
        if subject.LoginID == "" {
            return nil, nil
        }
        return nil, nil // Fall back to Session permissions.
    },
}
```

Returning an empty slice `[]string{}` means the subject explicitly has no permissions, so DToken-Go does not fall back.

## Error Handling

If the provider returns an error, permission or role resolution fails. Boolean `Has*` APIs fail closed so requests are not allowed when the permission service is unavailable.

```go
PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
    return nil, errors.New("permission service unavailable")
}
```

## Compatible Callbacks

Builder also keeps simpler callback APIs:

```go
mgr, err := defaults.NewBuilder().
    SetStorage(storage).
    SetCustomPermissionListFunc(func(loginID, authType string) ([]string, error) {
        return []string{"user:read"}, nil
    }).
    SetCustomRoleListFunc(func(loginID, authType string) ([]string, error) {
        return []string{"user"}, nil
    }).
    SetCustomPermissionListExtFunc(func(loginID, device, deviceID, authType string) ([]string, error) {
        if device == "app" {
            return []string{"mobile:read"}, nil
        }
        return nil, nil
    }).
    Build()
```

New projects should prefer `SetAccessProvider` because it is more expressive.

## Use With Multi-Auth Systems

`AccessSubject.AuthType` carries the current auth system:

```go
PermissionFunc: func(ctx context.Context, subject manager.AccessSubject) ([]string, error) {
    switch subject.AuthType {
    case "admin:":
        return loadAdminPermissions(subject.LoginID)
    case "user:":
        return loadUserPermissions(subject.LoginID)
    default:
        return nil, nil
    }
}
```

## Suggestions

- Use `AccessProvider` when permissions change frequently.
- Store access data in Session when it is stable and tied to login state.
- Use timeouts when the provider calls external services.
- Treat `nil` and `[]string{}` as different results.
- Monitor provider errors in production.

## Related Documentation

- [Permission Management](../core/permission.md)
- [Multi-Auth Systems](../core/multi-auth.md)
- [Framework Integration](../integration/framework-integration.md)
