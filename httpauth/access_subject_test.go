package httpauth

import (
	"context"
	"errors"
	"testing"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
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
						providerErr = errors.New("permission service unavailable")
						if _, err := Check(ctx, mgr, req); !errors.Is(err, providerErr) {
							t.Fatalf("provider failure = %v, want original error", err)
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
