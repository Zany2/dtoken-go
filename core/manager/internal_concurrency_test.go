package manager

import (
	"context"
	"testing"

	"github.com/Zany2/dtoken-go/core/config"
)

// TestManagerSharedLoginRejectsMismatchedTerminalMetadata verifies sharing trusts the token mapping instead of stale terminal fields. TestManagerSharedLoginRejectsMismatchedTerminalMetadata 验证共享登录以 Token 映射为身份真值，不信任错位终端字段。
func TestManagerSharedLoginRejectsMismatchedTerminalMetadata(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 0
	})

	loginID := "shared-terminal-owner"
	sharedCandidate, err := mgr.Login(ctx, loginID, "web", "browser")
	if err != nil {
		t.Fatalf("Login(shared candidate) error = %v", err)
	}
	mismatchedToken, err := mgr.Login(ctx, loginID, "mobile", "phone")
	if err != nil {
		t.Fatalf("Login(mismatched candidate) error = %v", err)
	}

	sess, err := mgr.GetSession(ctx, loginID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	for i := range sess.TerminalInfos {
		if sess.TerminalInfos[i].Token == mismatchedToken {
			sess.TerminalInfos[i].LoginID = "mismatched-terminal-owner"
			sess.TerminalInfos[i].Device = "web"
			sess.TerminalInfos[i].DeviceID = "browser"
		}
	}
	if err = mgr.saveToStorage(ctx, mgr.getSessionKey(loginID), *sess); err != nil {
		t.Fatalf("save mismatched session error = %v", err)
	}

	mgr.config.IsShare = true
	sharedToken, err := mgr.Login(ctx, loginID, "web", "browser")
	if err != nil {
		t.Fatalf("shared Login() error = %v", err)
	}
	if sharedToken != sharedCandidate {
		t.Fatalf("shared Login() token = %q, want valid candidate %q", sharedToken, sharedCandidate)
	}

	info, err := mgr.GetTokenInfo(ctx, mismatchedToken)
	if err != nil {
		t.Fatalf("GetTokenInfo(mismatched candidate) error = %v", err)
	}
	if info.LoginID != loginID || info.Device != "mobile" || info.DeviceID != "phone" {
		t.Fatalf("mismatched candidate identity = %q/%q/%q, want %q/mobile/phone", info.LoginID, info.Device, info.DeviceID, loginID)
	}
	if mgr.storage.Exists(ctx, mgr.getSessionKey("mismatched-terminal-owner")) {
		t.Fatal("shared login created a session from mismatched terminal identity")
	}

	terminals, err := mgr.GetTerminalListByLoginID(ctx, loginID)
	if err != nil {
		t.Fatalf("GetTerminalListByLoginID() error = %v", err)
	}
	if len(terminals) != 1 || terminals[0].Token != sharedCandidate {
		t.Fatalf("terminals after shared cleanup = %+v, want only %q", terminals, sharedCandidate)
	}
}

// TestManagerSharedLoginUsesSessionKeyOwner verifies a corrupted session payload cannot redirect account-scoped writes. TestManagerSharedLoginUsesSessionKeyOwner 验证损坏的 Session 载荷不能重定向账号级写操作。
func TestManagerSharedLoginUsesSessionKeyOwner(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = true
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 0
	})

	victimToken, err := mgr.Login(ctx, "session-key-victim", "mobile", "phone")
	if err != nil {
		t.Fatalf("Login(victim) error = %v", err)
	}
	ownerToken, err := mgr.Login(ctx, "session-key-owner", "web", "browser")
	if err != nil {
		t.Fatalf("Login(owner) error = %v", err)
	}

	ownerSession, err := mgr.GetSession(ctx, "session-key-owner")
	if err != nil {
		t.Fatalf("GetSession(owner) error = %v", err)
	}
	ownerSession.AuthType = "foreign-auth-type:"
	ownerSession.LoginID = "session-key-victim"
	if err = mgr.saveToStorage(ctx, mgr.getSessionKey("session-key-owner"), *ownerSession); err != nil {
		t.Fatalf("save corrupted owner session error = %v", err)
	}

	sharedToken, err := mgr.Login(ctx, "session-key-owner", "web", "browser")
	if err != nil {
		t.Fatalf("shared Login(owner) error = %v", err)
	}
	if sharedToken != ownerToken {
		t.Fatalf("shared Login(owner) token = %q, want %q", sharedToken, ownerToken)
	}
	if err = mgr.CheckLogin(ctx, victimToken); err != nil {
		t.Fatalf("CheckLogin(victim) error = %v, want victim session untouched", err)
	}

	ownerSession, err = mgr.GetSession(ctx, "session-key-owner")
	if err != nil {
		t.Fatalf("GetSession(owner after sharing) error = %v", err)
	}
	if ownerSession.AuthType != mgr.config.AuthType || ownerSession.LoginID != "session-key-owner" {
		t.Fatalf("owner session identity = %q/%q, want %q/session-key-owner", ownerSession.AuthType, ownerSession.LoginID, mgr.config.AuthType)
	}
}

// TestManagerGetTokenAndShareHandlesNilSession verifies the internal sharing helper safely handles a missing session. TestManagerGetTokenAndShareHandlesNilSession 验证内部共享辅助方法可安全处理空 Session。
func TestManagerGetTokenAndShareHandlesNilSession(t *testing.T) {
	mgr := newTestManager(t, nil)
	token, err := mgr.getTokenAndShare(context.Background(), nil, "web", "browser")
	if err != nil || token != "" {
		t.Fatalf("getTokenAndShare(nil) = %q, %v, want empty token and nil error", token, err)
	}
}
