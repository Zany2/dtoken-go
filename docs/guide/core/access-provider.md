# AccessProvider

Import aliases for code fragments: dt = github.com/Zany2/dtoken-go.

**[中文文档](../core/access-provider_zh.md)**

`AccessProvider` resolves permissions and roles dynamically from business systems. It fits projects where access data lives in a database, RBAC service, configuration center, or third-party permission platform.

## Relationship With Session Access

DToken-Go supports two access sources:

1. Permissions and roles stored in Session through APIs such as `AddPermissions` and `AddRoles`.
2. Permissions and roles returned dynamically by `AccessProvider` (including `AccessProviderFunc`).

When `AccessProvider` is configured, non-`nil` data returned by the provider takes priority over Session data. If the provider returns `nil`, DToken-Go falls back to Session data.

Token queries and HTTP authorization validate credentials and build the full `AccessSubject` before calling the required provider. On the normal path, a non-nil result (including an empty slice) or an error avoids Session reads and decoding. Session data is loaded only when the provider is absent or returns nil. Combined HTTP permission and role checks reuse one lazy Session read, and permission rejection skips the role query. Reuse is limited to one operation; permission results are not cached across requests.

With valid credentials and provider-owned authorization, a missing or corrupt Session does not block the decision. Session fallback still reports the corresponding errors. Inactivity cleanup, background maintenance, and full introspection retain their own Session requirements outside this read budget. Authorization queries do not implicitly check restrictions; access points must explicitly enable `CheckDisable` when needed.

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

<!-- dtoken-compile: access-provider -->
```go
package main

import (
	"context"
	dt "github.com/Zany2/dtoken-go"
)

func main() {
	provider := dt.AccessProviderFunc{
		PermissionFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
			if subject.AuthType == "admin:" {
				return []string{"admin:*"}, nil
			}
			return []string{"user:read"}, nil
		},
		RoleFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
			if subject.LoginID == "10001" {
				return []string{"admin"}, nil
			}
			return []string{"user"}, nil
		},
	}

	auth, err := dt.NewBuilder().
		SetAccessProvider(provider).
		Build()
	if err != nil {
		panic(err)
	}
	defer auth.Close()
}
```

## Fallback To Session

Returning `nil` means the provider does not handle this query, so DToken-Go should use Session data:

```go
provider := dt.AccessProviderFunc{
    PermissionFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
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

Integration middleware uses token-based `Check*` APIs to preserve the request's device and Token fields. Provider errors propagate to the integration failure handler, allowing it to distinguish service failures from `ErrPermissionDenied` or `ErrRoleDenied`.

```go
PermissionFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
    return nil, errors.New("permission service unavailable")
}
```

## Function Providers And Legacy Callback Migration

The current builder uses `SetAccessProvider`. The old `SetCustomPermissionListFunc`, `SetCustomRoleListFunc`, and Ext variants are no longer provided. Move callbacks to `AccessProviderFunc.PermissionFunc` / `RoleFunc` and read namespace, account, device, and token from subject.

```go
provider := dt.AccessProviderFunc{
    PermissionFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
        if subject.Device == "app" {
            return []string{"mobile:read", "mobile:write"}, nil
        }
        return nil, nil
    },
    RoleFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
        if subject.LoginID == "10001" {
            return []string{"admin"}, nil
        }
        return []string{}, nil
    },
}
auth, err := dt.NewBuilder().SetAccessProvider(provider).Build()
if err != nil {
    panic(err)
}
defer auth.Close()
```

Account queries do not populate token or device fields. Use token queries or framework authentication when those fields are needed. Only nil results fall back to the session; a non-nil empty slice explicitly denies permissions or roles.

## Use With Multi-Auth Systems

`AccessSubject.AuthType` carries the current auth system:

```go
PermissionFunc: func(ctx context.Context, subject dt.AccessSubject) ([]string, error) {
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
