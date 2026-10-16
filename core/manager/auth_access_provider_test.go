package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/core/listener"
)

// TestTokenAccessProviderErrorsDoNotEmitCheckEvents verifies provider failures fail closed without reporting a business decision. TestTokenAccessProviderErrorsDoNotEmitCheckEvents 验证提供器故障会安全拒绝且不会伪报业务判定事件。
func TestTokenAccessProviderErrorsDoNotEmitCheckEvents(t *testing.T) {
	ctx := context.Background()
	providerErr := errors.New("provider unavailable")
	mgr := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{
		PermissionFunc: func(context.Context, AccessSubject) ([]string, error) {
			return nil, providerErr
		},
		RoleFunc: func(context.Context, AccessSubject) ([]string, error) {
			return nil, providerErr
		},
	})
	token, err := mgr.Login(ctx, "provider-error-events", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	eventCount := 0
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		if data.Event == listener.EventPermissionCheck || data.Event == listener.EventRoleCheck {
			eventCount++
		}
	}, listener.ListenerConfig{Async: false})

	results := []bool{
		mgr.HasPermissionByToken(ctx, token, "article:read"),
		mgr.HasPermissionsAndByToken(ctx, token, []string{"article:read"}),
		mgr.HasPermissionsOrByToken(ctx, token, []string{"article:read"}),
		mgr.HasRoleByToken(ctx, token, "admin"),
		mgr.HasRolesAndByToken(ctx, token, []string{"admin"}),
		mgr.HasRolesOrByToken(ctx, token, []string{"admin"}),
	}
	for index, result := range results {
		if result {
			t.Fatalf("access result %d = true, want false on provider error", index)
		}
	}
	if eventCount != 0 {
		t.Fatalf("provider error check event count = %d, want 0", eventCount)
	}
}
