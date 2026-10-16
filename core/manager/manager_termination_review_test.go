package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerTerminateRejectsBlankExplicitFilters prevents normalization from widening the target scope. TestManagerTerminateRejectsBlankExplicitFilters 防止规范化扩大目标范围。
func TestManagerTerminateRejectsBlankExplicitFilters(t *testing.T) {
	ctx := context.Background()
	for _, action := range []TerminateAction{"", TerminateActionLogout, TerminateActionKickout, TerminateActionReplace} {
		t.Run(string(action), func(t *testing.T) {
			mgr := newTestManager(t, nil)
			id := "blank-termination-filters"
			var keys []string
			for _, device := range [][2]string{{"web", "one"}, {"web", "two"}, {"mobile", "one"}} {
				pair, err := mgr.LoginWithRefreshToken(ctx, id, device[0], device[1])
				if err != nil {
					t.Fatal(err)
				}
				keys = append(keys, mgr.getTokenKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken))
			}
			keys = append(keys, mgr.getSessionKey(id))
			before := captureTerminalLifecycle(t, mgr, ctx, keys)
			events := registerTerminalLifecycleEvents(mgr)
			for _, input := range []struct {
				opts TerminateOptions
				err  error
			}{
				{TerminateOptions{Token: " \t"}, derror.ErrInvalidToken},
				{TerminateOptions{Token: " \t", Device: "web", DeviceID: "one"}, derror.ErrInvalidToken},
				{TerminateOptions{Device: " \t"}, derror.ErrInvalidParam},
				{TerminateOptions{DeviceID: " \t"}, derror.ErrInvalidParam},
				{TerminateOptions{Device: "web", DeviceID: " \t"}, derror.ErrInvalidParam},
				{TerminateOptions{Device: " \t", DeviceID: "one"}, derror.ErrInvalidParam},
			} {
				input.opts.LoginID, input.opts.Action = id, action
				if err := mgr.Terminate(ctx, input.opts); !errors.Is(err, input.err) {
					t.Fatalf("Terminate(%+v) = %v, want %v", input.opts, err, input.err)
				}
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			}
			if len(events()) != 0 {
				t.Fatalf("invalid filters emitted events: %+v", events())
			}
		})
	}
}

// TestManagerDirectTerminationRetiresResidualLogin verifies revocation survives missing context and reversible bans. TestManagerDirectTerminationRetiresResidualLogin 验证上下文缺失或可逆封禁不阻止撤销登录。
func TestManagerDirectTerminationRetiresResidualLogin(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []struct {
		name     string
		action   TerminateAction
		run      func(*Manager, context.Context, string) error
		stateErr error
		event    listener.Event
	}{
		{"logout", TerminateActionLogout, (*Manager).Logout, derror.ErrInvalidToken, listener.EventLogout},
		{"kickout", TerminateActionKickout, (*Manager).Kickout, derror.ErrTokenKickout, listener.EventKickout},
		{"replace", TerminateActionReplace, (*Manager).Replace, derror.ErrTokenReplaced, listener.EventReplace},
	} {
		for _, condition := range []string{"missing session", "missing activity", "disabled device", "disabled account"} {
			for _, ordinary := range []bool{false, true} {
				backend := "atomic"
				if ordinary {
					backend = "ordinary"
				}
				t.Run(operation.name+"/"+condition+"/"+backend, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.ActiveTimeout = 30
						cfg.RenewInterval = 30
						cfg.IsShare = false
					})
					if ordinary {
						mgr.storage = &managerStorageOnly{inner: newManagerTestStorage()}
					}
					id := "residual-login"
					pair, err := mgr.LoginWithRefreshToken(ctx, id, "web", "target")
					if err != nil {
						t.Fatal(err)
					}
					keptID := id
					if condition == "missing session" || condition == "disabled account" {
						keptID = "other-account"
					}
					kept, err := mgr.LoginWithRefreshToken(ctx, keptID, "mobile", "kept")
					if err != nil {
						t.Fatal(err)
					}
					switch condition {
					case "missing session":
						err = mgr.storage.Delete(ctx, mgr.getSessionKey(id))
					case "missing activity":
						err = mgr.storage.Delete(ctx, mgr.getActiveKey(pair.AccessToken))
					case "disabled device":
						err = mgr.DisableDevice(ctx, id, "web", time.Hour)
					case "disabled account":
						err = mgr.Disable(ctx, id, time.Hour)
						mgr.asyncWG.Wait()
					}
					if err != nil {
						t.Fatal(err)
					}
					keptKeys := []string{mgr.getTokenKey(kept.AccessToken), mgr.getActiveKey(kept.AccessToken), mgr.getRenewKey(kept.AccessToken), mgr.getRefreshTokenKey(kept.RefreshToken), mgr.getTokenRefreshKey(kept.AccessToken)}
					before := captureTerminalLifecycle(t, mgr, ctx, keptKeys)
					events := registerTerminalLifecycleEvents(mgr)
					if err := operation.run(mgr, ctx, pair.AccessToken); err != nil {
						t.Fatal(err)
					}
					firstEvents := events()
					// Repeating through the unified API must neither alter the cause nor emit another event. 通过统一接口重复执行不能改写下线原因或重复发事件。
					if err := mgr.Terminate(ctx, TerminateOptions{Token: pair.AccessToken, Action: operation.action}); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(firstEvents, events()) {
						t.Fatal("repeated termination emitted events")
					}
					assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
					for _, key := range []string{mgr.getActiveKey(pair.AccessToken), mgr.getRenewKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken)} {
						if mgr.storage.Exists(ctx, key) {
							t.Fatalf("termination retained %q", key)
						}
					}
					if _, err := mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
						t.Fatalf("revoked refresh token recovered login: %v", err)
					}
					if condition == "disabled device" {
						if err := mgr.UntieDevice(ctx, id, "web"); err != nil {
							t.Fatal(err)
						}
					}
					if condition == "disabled account" {
						if err := mgr.Untie(ctx, id); err != nil {
							t.Fatal(err)
						}
					}
					wantErr := operation.stateErr
					// A missing activity marker was already invalid and must not be rewritten as a new kickout reason. 缺失活跃标记的登录本已无效，不改写成新的踢出原因。
					if condition == "missing activity" {
						wantErr = derror.ErrInvalidToken
					}
					if err := mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, wantErr) {
						t.Fatalf("terminated token = %v, want %v", err, wantErr)
					}
					if condition != "missing activity" || operation.action == TerminateActionLogout {
						if len(firstEvents) != 1 || firstEvents[0].Event != operation.event || firstEvents[0].Token != pair.AccessToken {
							t.Fatalf("termination events = %+v", firstEvents)
						}
					} else if len(firstEvents) != 0 {
						t.Fatalf("inactive token emitted transition events: %+v", firstEvents)
					}
					if condition != "missing session" && condition != "disabled account" {
						sess, err := mgr.getSession(ctx, id)
						if err != nil || len(sess.TerminalInfos) != 1 || sess.TerminalInfos[0].Token != kept.AccessToken {
							t.Fatalf("remaining session = %+v, %v", sess, err)
						}
					}
				})
			}
		}
	}
}
