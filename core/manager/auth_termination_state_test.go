package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerScopedTerminationPersistsStateDuringTemporaryDisable verifies reversible disable rules cannot prevent an explicit terminal state transition. TestManagerScopedTerminationPersistsStateDuringTemporaryDisable 验证可逆封禁规则不会阻止显式终端状态迁移。
func TestManagerScopedTerminationPersistsStateDuringTemporaryDisable(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		action  func(*Manager, string) error
		wantErr error
	}{
		{
			name: "kickout",
			action: func(mgr *Manager, loginID string) error {
				return mgr.KickoutByDevice(ctx, loginID, "web")
			},
			wantErr: derror.ErrTokenKickout,
		},
		{
			name: "replace",
			action: func(mgr *Manager, loginID string) error {
				return mgr.ReplaceByDevice(ctx, loginID, "web")
			},
			wantErr: derror.ErrTokenReplaced,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.IsShare = false
			})
			loginID := "disabled-termination-" + tt.name
			token, err := mgr.Login(ctx, loginID, "web", "browser")
			if err != nil {
				t.Fatalf("Login(web) error = %v", err)
			}
			keptToken, err := mgr.Login(ctx, loginID, "mobile", "phone")
			if err != nil {
				t.Fatalf("Login(mobile) error = %v", err)
			}
			if err = mgr.DisableDevice(ctx, loginID, "web", time.Minute); err != nil {
				t.Fatalf("DisableDevice() error = %v", err)
			}

			if err = tt.action(mgr, loginID); err != nil {
				t.Fatalf("termination action error = %v", err)
			}
			if err = mgr.UntieDevice(ctx, loginID, "web"); err != nil {
				t.Fatalf("UntieDevice() error = %v", err)
			}
			if err = mgr.CheckLogin(ctx, token); !errors.Is(err, tt.wantErr) {
				t.Fatalf("terminated token CheckLogin() error = %v, want %v", err, tt.wantErr)
			}
			if err = mgr.CheckLogin(ctx, keptToken); err != nil {
				t.Fatalf("kept token CheckLogin() error = %v", err)
			}
		})
	}
}

// TestManagerScopedTerminationSkipsEventsForStaleTokens verifies cleanup does not report state changes for already inactive tokens. TestManagerScopedTerminationSkipsEventsForStaleTokens 验证清理过程不会为已失效 Token 上报状态变更事件。
func TestManagerScopedTerminationSkipsEventsForStaleTokens(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})

	staleToken, err := mgr.Login(ctx, "termination-events", "web", "stale")
	if err != nil {
		t.Fatalf("Login(stale) error = %v", err)
	}
	activeToken, err := mgr.Login(ctx, "termination-events", "mobile", "active")
	if err != nil {
		t.Fatalf("Login(active) error = %v", err)
	}
	if err = mgr.storage.Delete(ctx, mgr.getTokenKey(staleToken)); err != nil {
		t.Fatalf("Delete(stale token) error = %v", err)
	}

	var tokens []string
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventKickout, func(data *listener.EventData) {
		tokens = append(tokens, data.Token)
	}, listener.ListenerConfig{Async: false})

	if err = mgr.KickoutByLoginID(ctx, "termination-events"); err != nil {
		t.Fatalf("KickoutByLoginID() error = %v", err)
	}
	if len(tokens) != 1 || tokens[0] != activeToken {
		t.Fatalf("kickout event tokens = %v, want [%s]", tokens, activeToken)
	}
}
