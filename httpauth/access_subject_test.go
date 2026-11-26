package httpauth

import (
	"context"
	"errors"
	"testing"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/event"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// TestAccessChecksPreserveTokenSubjectAndFailures covers route and standalone authorization paths. TestAccessChecksPreserveTokenSubjectAndFailures 覆盖路由与独立授权路径的 Token 主体及故障传递。
func TestAccessChecksPreserveTokenSubjectAndFailures(t *testing.T) {
	ctx := context.Background()
	for _, roles := range []bool{false, true} {
		name := "permissions"
		if roles {
			name = "roles"
		}
		t.Run(name, func(t *testing.T) {
			var subjects []manager.AccessSubject
			var providerErr error
			resolve := func(_ context.Context, subject manager.AccessSubject) ([]string, error) {
				subjects = append(subjects, subject)
				return []string{"allowed"}, providerErr
			}
			provider := manager.AccessProviderFunc{PermissionFunc: resolve, RoleFunc: resolve}
			cfg := newAuthcheckTestManager(t).GetConfig().Clone()
			mgr := manager.NewManager(cfg, &authcheckTestGenerator{}, newAuthcheckTestStorage(), authcheckTestCodec{}, adapter.NewNopLogger(), nil, provider)
			t.Cleanup(mgr.CloseManager)
			token, err := mgr.Login(ctx, "scoped-user", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			// Provider-owned authorization must not depend on a Session that it does not use. 提供器负责的授权不能依赖不会使用的 Session。
			if err := mgr.GetStorage().Delete(ctx, utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType)+manager.SessionKeyPrefix+"scoped-user"); err != nil {
				t.Fatal(err)
			}

			// Exercise every combination that previously selected an account-only query. 覆盖此前选择纯账号查询的所有组合。
			for _, checkLogin := range []bool{false, true} {
				for _, checkDisable := range []bool{false, true} {
					for _, logic := range []LogicType{"", LogicAnd, LogicOr} {
						req := Request{TokenValue: token, CheckLogin: checkLogin, CheckDisable: checkDisable, LogicType: logic}
						if roles {
							req.Roles = []string{"allowed"}
						} else {
							req.Permissions = []string{"allowed"}
						}
						subjects = nil
						if _, err := Check(ctx, mgr, req); err != nil {
							t.Fatal(err)
						}
						if len(subjects) != 1 {
							t.Fatalf("provider calls = %d, want 1", len(subjects))
						}
						want := manager.AccessSubject{AuthType: cfg.AuthType, LoginID: "scoped-user", Device: "web", DeviceID: "browser", Token: token}
						if subjects[0] != want {
							t.Fatalf("subject = %+v, want %+v", subjects[0], want)
						}

						// Provider outages must remain distinguishable from access denial. 提供器故障须与正常访问拒绝区分。
						for _, failure := range []error{errors.New("permission service unavailable"), derror.ErrInvalidToken} {
							providerErr = failure
							req.LoginError = errors.New("custom login failure")
							if _, err := Check(ctx, mgr, req); !errors.Is(err, providerErr) {
								t.Fatalf("provider failure = %v, want original error", err)
							}
						}
						providerErr = nil
						if roles {
							req.Roles = []string{"missing"}
						} else {
							req.Permissions = []string{"missing"}
						}
						denied := derror.ErrPermissionDenied
						if roles {
							denied = derror.ErrRoleDenied
						}
						if _, err := Check(ctx, mgr, req); !errors.Is(err, denied) {
							t.Fatalf("denial = %v, want %v", err, denied)
						}
					}
				}
			}
		})
	}
}

// TestCombinedAccessChecksPreservePolicyAndEvents guards provider calls, event order, and short-circuit semantics. TestCombinedAccessChecksPreservePolicyAndEvents 保护提供器调用、事件顺序和短路语义。
func TestCombinedAccessChecksPreservePolicyAndEvents(t *testing.T) {
	ctx := context.Background()
	var subjects []manager.AccessSubject
	var permissionCalls, roleCalls int
	provider := manager.AccessProviderFunc{
		PermissionFunc: func(_ context.Context, subject manager.AccessSubject) ([]string, error) {
			permissionCalls++
			subjects = append(subjects, subject)
			return []string{"read", "write"}, nil
		},
		RoleFunc: func(_ context.Context, subject manager.AccessSubject) ([]string, error) {
			roleCalls++
			subjects = append(subjects, subject)
			return []string{"reader", "writer"}, nil
		},
	}
	cfg := newAuthcheckTestManager(t).GetConfig()
	mgr := manager.NewManager(cfg, &authcheckTestGenerator{}, newAuthcheckTestStorage(), authcheckTestCodec{}, adapter.NewNopLogger(), nil, provider)
	t.Cleanup(mgr.CloseManager)
	token, err := mgr.Login(ctx, "user", "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	var events []*event.EventData
	for _, name := range []event.Event{event.EventPermissionCheck, event.EventRoleCheck} {
		mgr.GetEventManager().RegisterFuncWithConfig(name, func(data *event.EventData) { events = append(events, data) }, event.ListenerConfig{Async: false})
	}
	for _, tc := range []struct {
		name               string
		logic              LogicType
		permissions, roles []string
		want               error
		count              int
	}{
		{"and allowed", LogicAnd, []string{"read", "write"}, []string{"reader", "writer"}, nil, 2},
		{"or allowed", LogicOr, []string{"read", "missing"}, []string{"reader", "missing"}, nil, 2},
		{"and permission denied", LogicAnd, []string{"read", "missing"}, []string{"reader"}, derror.ErrPermissionDenied, 1},
		{"or permission denied", LogicOr, []string{"missing"}, []string{"reader"}, derror.ErrPermissionDenied, 1},
		{"and role denied", LogicAnd, []string{"read"}, []string{"reader", "missing"}, derror.ErrRoleDenied, 2},
		{"or role denied", LogicOr, []string{"read"}, []string{"missing"}, derror.ErrRoleDenied, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			subjects, events = nil, nil
			permissionCalls, roleCalls = 0, 0
			_, err := Check(ctx, mgr, Request{TokenValue: token, CheckLogin: true, CheckDisable: true, Permissions: tc.permissions, Roles: tc.roles, LogicType: tc.logic})
			if !errors.Is(err, tc.want) {
				t.Fatalf("Check() = %v, want %v", err, tc.want)
			}
			if permissionCalls != 1 || roleCalls != tc.count-1 || len(events) != tc.count {
				t.Fatalf("calls/events = %d/%d/%d, want 1/%d/%d", permissionCalls, roleCalls, len(events), tc.count-1, tc.count)
			}
			wantSubject := manager.AccessSubject{AuthType: cfg.AuthType, LoginID: "user", Device: "web", DeviceID: "browser", Token: token}
			for _, subject := range subjects {
				if subject != wantSubject {
					t.Fatalf("subject = %+v", subject)
				}
			}
			for i, data := range events {
				wantEvent := event.EventPermissionCheck
				if i == 1 {
					wantEvent = event.EventRoleCheck
				}
				wantAllowed := tc.want == nil || i < len(events)-1
				if data.Event != wantEvent || data.AuthType != cfg.AuthType || data.LoginID != "user" || data.Device != "web" || data.DeviceID != "browser" || data.Token != token || data.Extra[event.ExtraKeyLogic] != string(tc.logic) || data.Extra[event.ExtraKeyResult] != wantAllowed {
					t.Fatalf("event %d = %+v, want %s, result %v", i, data, wantEvent, wantAllowed)
				}
			}
		})
	}
}
