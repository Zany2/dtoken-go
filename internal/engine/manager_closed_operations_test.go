package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	derror "github.com/Zany2/dtoken-go/errors"
)

// TestUnavailableManagerBusinessMethods verifies shutdown takes precedence over argument validation across the public business API. TestUnavailableManagerBusinessMethods 验证所有公开业务方法优先检查关闭状态，再校验参数。
func TestUnavailableManagerBusinessMethods(t *testing.T) {
	closing := newTestManager(t, nil)
	BeginClose(closing)
	closed := newTestManager(t, nil)
	closed.CloseManager()
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	for name, mgr := range map[string]*Manager{"nil": nil, "closing": closing, "closed": closed} {
		t.Run(name, func(t *testing.T) {
			value := reflect.ValueOf(mgr)
			checked := 0
			for i := 0; i < value.NumMethod(); i++ {
				method := value.Method(i)
				typ := method.Type()
				if typ.NumIn() == 0 || typ.In(0) != contextType {
					continue
				}
				checked++
				t.Run(value.Type().Method(i).Name, func(t *testing.T) {
					args := make([]reflect.Value, typ.NumIn())
					args[0] = reflect.ValueOf(context.Background())
					for j := 1; j < len(args); j++ {
						args[j] = reflect.Zero(typ.In(j))
					}
					var results []reflect.Value
					if typ.IsVariadic() {
						results = method.CallSlice(args)
					} else {
						results = method.Call(args)
					}
					for _, result := range results {
						if result.Type() == errorType {
							err, _ := result.Interface().(error)
							if !errors.Is(err, derror.ErrManagerNotFound) {
								t.Errorf("error = %v, want ErrManagerNotFound", err)
							}
						} else if !result.IsZero() {
							t.Errorf("result = %v, want zero value", result.Interface())
						}
					}
				})
			}
			if checked == 0 {
				t.Fatal("no public business methods were checked")
			}
		})
	}
}

// TestClosedManagerCannotUseLiveCredentials verifies shutdown rejects valid operations even when borrowed storage remains available. TestClosedManagerCannotUseLiveCredentials 验证借用存储仍可用时，关闭后的有效业务请求仍被拒绝。
func TestClosedManagerCannotUseLiveCredentials(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	mgr.ownership.Storage = false
	const loginID = "closed-manager"
	pair, err := mgr.LoginWithRefreshToken(ctx, loginID, "web", "browser-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{
		mgr.AddPermissions(ctx, loginID, []string{"read"}),
		mgr.AddRoles(ctx, loginID, []string{"user"}),
		mgr.CheckLogin(ctx, pair.AccessToken),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	before := captureTerminalLifecycle(t, mgr, ctx, []string{
		mgr.getTokenKey(pair.AccessToken), mgr.getSessionKey(loginID),
		mgr.getSessionKey("new-account"), mgr.getDisableKey(loginID),
		mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken),
		mgr.getActiveKey(pair.AccessToken), mgr.getRenewKey(pair.AccessToken),
	})
	events := registerTerminalLifecycleEvents(mgr)
	mgr.CloseManager()

	if mgr.IsLogin(ctx, pair.AccessToken) || mgr.HasPermission(ctx, loginID, "read") || mgr.HasRole(ctx, loginID, "user") {
		t.Fatal("closed manager accepted a boolean business check")
	}
	checks := []struct {
		name string
		run  func() error
	}{
		{"login", func() error { _, err := mgr.Login(ctx, "new-account"); return err }},
		{"check login", func() error { return mgr.CheckLogin(ctx, pair.AccessToken) }},
		{"logout", func() error { return mgr.Logout(ctx, pair.AccessToken) }},
		{"session", func() error { _, err := mgr.GetSession(ctx, loginID); return err }},
		{"session write", func() error { return mgr.SetSessionValue(ctx, loginID, "key", "value") }},
		{"disable", func() error { return mgr.Disable(ctx, loginID, time.Hour) }},
		{"refresh", func() error { _, err := mgr.RefreshToken(ctx, pair.RefreshToken); return err }},
		{"permission", func() error { return mgr.CheckPermission(ctx, loginID, "read") }},
		{"role", func() error { return mgr.CheckRole(ctx, loginID, "user") }},
		{"terminals", func() error {
			return mgr.ForEachTerminal(ctx, loginID, func(TerminalInfo) bool {
				t.Error("closed manager invoked a terminal visitor")
				return true
			})
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.run(); !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("error = %v, want ErrManagerNotFound", err)
			}
		})
	}
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
	if len(events()) != 0 {
		t.Fatal("closed manager emitted business events")
	}
}
