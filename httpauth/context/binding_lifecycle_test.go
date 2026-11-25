package context

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Zany2/dtoken-go/contract/nonce"
	"github.com/Zany2/dtoken-go/contract/oauth2"
	"github.com/Zany2/dtoken-go/contract/shortkey"
	"github.com/Zany2/dtoken-go/contract/ticket"
	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// TestAuthenticationRequiresRequestBindings verifies missing bindings return errors before token extraction or login writes. TestAuthenticationRequiresRequestBindings 验证缺失绑定时在 Token 提取或登录写入之前返回错误。
func TestAuthenticationRequiresRequestBindings(t *testing.T) {
	_, req, mgr := newTestDTokenContext(t)
	var absent *testRequestContext
	for _, tc := range []struct {
		name  string
		scope *DTokenContext
		want  error
	}{
		{"manager", NewContext(req, nil), derror.ErrManagerNotFound},
		{"request", NewContext(nil, mgr), derror.ErrInvalidParam},
		{"typed nil request", NewContext(absent, mgr), derror.ErrInvalidParam},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.scope.GetTokenValue() != "" || tc.scope.Auth().IsLogin(ctx) || tc.scope.Access().HasRole(ctx, "admin") {
				t.Fatal("invalid bindings produced token or authentication success")
			}
			_, cookieErr := tc.scope.Cookie().Login(ctx, "rejected-cookie-login")
			for _, err := range []error{cookieErr, tc.scope.Auth().CheckLogin(ctx), tc.scope.Cookie().Logout(ctx)} {
				if !errors.Is(err, tc.want) {
					t.Errorf("error = %v, want %v", err, tc.want)
				}
			}
			tc.scope.Cookie().SetToken("ignored")
			tc.scope.Cookie().ClearToken()
		})
	}
	if _, err := mgr.GetSession(context.Background(), "rejected-cookie-login"); !errors.Is(err, derror.ErrSessionNotFound) {
		t.Fatalf("invalid bindings created session: %v", err)
	}
	manager.BeginClose(mgr)
	if err := NewContext(req, mgr).Auth().CheckLogin(context.Background()); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("closed binding error = %v", err)
	}
}

// TestExplicitLoginWithoutRequest verifies explicit login operations need no request adapter. TestExplicitLoginWithoutRequest 验证显式登录操作无需请求适配器。
func TestExplicitLoginWithoutRequest(t *testing.T) {
	_, _, mgr := newTestDTokenContext(t)
	scope := NewContext(nil, mgr)
	ctx := context.Background()
	token, err := scope.Auth().Login(ctx, "explicit-login")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("explicit login produced an invalid token: %v", err)
	}
	if _, err := scope.Refresh().Login(ctx, "explicit-refresh-login"); err != nil {
		t.Fatalf("refresh login required a request adapter: %v", err)
	}

	missing := NewContext(nil, nil)
	if _, err := missing.Auth().Login(ctx, "missing-manager"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("login without manager error = %v", err)
	}
	if _, err := missing.Refresh().Login(ctx, "missing-manager"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("refresh login without manager error = %v", err)
	}
}

// Unimplemented embedded services detect accidental delegation after shutdown. 未实现的嵌入服务用于发现关闭后的意外转发。
type (
	shutdownNonce    struct{ nonce.Service }
	shutdownOAuth2   struct{ oauth2.Service }
	shutdownTicket   struct{ ticket.Service }
	shutdownShortKey struct{ shortkey.Service }
)

// TestBoundCapabilitiesRejectShutdown verifies every capability operation stops at the bound engine boundary. TestBoundCapabilitiesRejectShutdown 验证所有能力操作在绑定引擎关闭后停止执行。
func TestBoundCapabilitiesRejectShutdown(t *testing.T) {
	for _, phase := range []string{"closing", "closed"} {
		t.Run(phase, func(t *testing.T) {
			scope, _, mgr := newTestDTokenContext(t)
			scope.WithCapabilities(&Capabilities{
				Nonce: shutdownNonce{}, OAuth2: shutdownOAuth2{}, Ticket: shutdownTicket{}, ShortKey: shutdownShortKey{},
			})
			if phase == "closing" {
				manager.BeginClose(mgr)
			} else {
				mgr.CloseManager()
			}
			contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
			errorType := reflect.TypeOf((*error)(nil)).Elem()
			for name, group := range map[string]any{
				"nonce": scope.Nonce(), "oauth2": scope.OAuth2(), "ticket": scope.Ticket(), "shortkey": scope.ShortKey(),
			} {
				value := reflect.ValueOf(group)
				for i := 0; i < value.NumMethod(); i++ {
					t.Run(name+"/"+value.Type().Method(i).Name, func(t *testing.T) {
						method := value.Method(i)
						typ := method.Type()
						args := make([]reflect.Value, typ.NumIn())
						for j := range args {
							args[j] = reflect.Zero(typ.In(j))
							if typ.In(j) == contextType {
								args[j] = reflect.ValueOf(context.Background())
							}
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
							} else if result.Kind() == reflect.Bool && result.Bool() {
								t.Error("closed engine accepted a capability check")
							}
						}
					})
				}
			}
		})
	}
}
