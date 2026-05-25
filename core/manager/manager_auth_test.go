package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestManagerLoginLifecycle verifies basic login persistence and logout invalidation. TestManagerLoginLifecycle 验证基础登录持久化和登出失效。
func TestManagerLoginLifecycle(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "u1", "web", "browser-1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token == "" {
		t.Fatal("Login() token is empty")
	}
	if !mgr.IsLogin(ctx, token) {
		t.Fatal("IsLogin() = false, want true")
	}

	loginID, err := mgr.GetLoginID(ctx, token)
	if err != nil {
		t.Fatalf("GetLoginID() error = %v", err)
	}
	if loginID != "u1" {
		t.Fatalf("GetLoginID() = %q, want u1", loginID)
	}

	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenInfo() error = %v", err)
	}
	if info.LoginID != "u1" || info.Device != "web" || info.DeviceId != "browser-1" || info.Timeout != 60 {
		t.Fatalf("TokenInfo = %+v, want login/device/deviceId/timeout preserved", info)
	}

	sess, err := mgr.GetSessionByToken(ctx, token)
	if err != nil {
		t.Fatalf("GetSessionByToken() error = %v", err)
	}
	if len(sess.TerminalInfos) != 1 || sess.TerminalInfos[0].Token != token {
		t.Fatalf("session terminals = %+v, want one terminal with token", sess.TerminalInfos)
	}

	ttl, err := mgr.GetTokenTTL(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("GetTokenTTL() = %d, want 1..60", ttl)
	}

	if err = mgr.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if mgr.IsLogin(ctx, token) {
		t.Fatal("IsLogin() after Logout = true, want false")
	}
	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin() after Logout error = %v, want ErrInvalidToken", err)
	}
}

// TestManagerKickoutAndReplacePreserveTokenState verifies state markers keep exact failure causes. TestManagerKickoutAndReplacePreserveTokenState 验证状态标记保留精确失败原因。
func TestManagerKickoutAndReplacePreserveTokenState(t *testing.T) {
	ctx := context.Background()

	t.Run("kickout", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		token, err := mgr.Login(ctx, "u2", "web")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		if err = mgr.Kickout(ctx, token); err != nil {
			t.Fatalf("Kickout() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("CheckLogin() error = %v, want ErrTokenKickout", err)
		}
	})

	t.Run("replace", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		token, err := mgr.Login(ctx, "u3", "web")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		if err = mgr.Replace(ctx, token); err != nil {
			t.Fatalf("Replace() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrTokenReplaced) {
			t.Fatalf("CheckLogin() error = %v, want ErrTokenReplaced", err)
		}
	})
}

// TestManagerConcurrencySharingAndOverflow verifies share reuse and overflow policy. TestManagerConcurrencySharingAndOverflow 验证共享复用和超限策略。
func TestManagerConcurrencySharingAndOverflow(t *testing.T) {
	ctx := context.Background()

	t.Run("share reuses newest alive token for same device", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = true
			cfg.AutoRenew = false
		})

		first, err := mgr.Login(ctx, "u4", "web", "same")
		if err != nil {
			t.Fatalf("first Login() error = %v", err)
		}
		second, err := mgr.Login(ctx, "u4", "web", "same")
		if err != nil {
			t.Fatalf("second Login() error = %v", err)
		}
		if second != first {
			t.Fatalf("shared token = %q, want %q", second, first)
		}
		tokens, err := mgr.GetTokenValueListByLoginID(ctx, "u4")
		if err != nil {
			t.Fatalf("GetTokenValueListByLoginID() error = %v", err)
		}
		if len(tokens) != 1 {
			t.Fatalf("token count = %d, want 1", len(tokens))
		}
	})

	t.Run("account max count kicks out oldest token", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
			cfg.MaxLoginCount = 2
			cfg.OverflowLogoutMode = config.LogoutModeKickout
			cfg.AutoRenew = false
		})

		first, err := mgr.Login(ctx, "u5", "web", "a")
		if err != nil {
			t.Fatalf("first Login() error = %v", err)
		}
		second, err := mgr.Login(ctx, "u5", "mobile", "b")
		if err != nil {
			t.Fatalf("second Login() error = %v", err)
		}
		third, err := mgr.Login(ctx, "u5", "desktop", "c")
		if err != nil {
			t.Fatalf("third Login() error = %v", err)
		}

		tokens, err := mgr.GetTokenValueListByLoginID(ctx, "u5", true)
		if err != nil {
			t.Fatalf("GetTokenValueListByLoginID(checkAlive) error = %v", err)
		}
		if !sameStrings(tokens, []string{second, third}) {
			t.Fatalf("alive tokens = %v, want [%s %s]", tokens, second, third)
		}
		if err = mgr.CheckLogin(ctx, first); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("oldest token CheckLogin() error = %v, want ErrTokenKickout", err)
		}
	})
}

// TestManagerNonConcurrentLoginReplacesOldToken verifies non-concurrent login replaces old sessions. TestManagerNonConcurrentLoginReplacesOldToken 验证非并发登录会顶替旧会话。
func TestManagerNonConcurrentLoginReplacesOldToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = false
		cfg.ConcurrencyScope = config.ConcurrencyScopeAccount
		cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
		cfg.AutoRenew = false
	})

	first, err := mgr.Login(ctx, "u6", "web")
	if err != nil {
		t.Fatalf("first Login() error = %v", err)
	}
	second, err := mgr.Login(ctx, "u6", "mobile")
	if err != nil {
		t.Fatalf("second Login() error = %v", err)
	}
	if first == second {
		t.Fatal("second login reused token, want replacement")
	}
	if err = mgr.CheckLogin(ctx, first); !errors.Is(err, derror.ErrTokenReplaced) {
		t.Fatalf("old token CheckLogin() error = %v, want ErrTokenReplaced", err)
	}
	if !mgr.IsLogin(ctx, second) {
		t.Fatal("new token IsLogin() = false, want true")
	}
}

// TestManagerPermissionsAndRoles verifies cached permission and role mutation and checks. TestManagerPermissionsAndRoles 验证缓存权限和角色的变更与校验。
func TestManagerPermissionsAndRoles(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	token, err := mgr.Login(ctx, "u7", "web")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if err = mgr.AddPermissions(ctx, "u7", []string{"user:*", "order/read", "order/read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if err = mgr.AddRolesByToken(ctx, token, []string{"admin", "editor", "admin"}); err != nil {
		t.Fatalf("AddRolesByToken() error = %v", err)
	}

	perms, err := mgr.GetPermissions(ctx, "u7")
	if err != nil {
		t.Fatalf("GetPermissions() error = %v", err)
	}
	if !sameStrings(perms, []string{"user:*", "order/read"}) {
		t.Fatalf("permissions = %v, want deduped values", perms)
	}
	if !mgr.HasPermission(ctx, "u7", "user:create") {
		t.Fatal("HasPermission(user:create) = false, want wildcard match")
	}
	if !mgr.HasPermissionsAndByToken(ctx, token, []string{"user:delete", "order/read"}) {
		t.Fatal("HasPermissionsAndByToken() = false, want true")
	}
	if mgr.HasPermissionsAnd(ctx, "u7", []string{"user:create", "missing"}) {
		t.Fatal("HasPermissionsAnd() = true, want false")
	}
	if err = mgr.CheckPermissionOr(ctx, "u7", []string{"missing", "order/read"}); err != nil {
		t.Fatalf("CheckPermissionOr() error = %v", err)
	}

	roles, err := mgr.GetRolesByToken(ctx, token)
	if err != nil {
		t.Fatalf("GetRolesByToken() error = %v", err)
	}
	if !sameStrings(roles, []string{"admin", "editor"}) {
		t.Fatalf("roles = %v, want deduped values", roles)
	}
	if !mgr.HasRolesAnd(ctx, "u7", []string{"admin", "editor"}) {
		t.Fatal("HasRolesAnd() = false, want true")
	}
	if err = mgr.RemoveRoles(ctx, "u7", []string{"editor"}); err != nil {
		t.Fatalf("RemoveRoles() error = %v", err)
	}
	if mgr.HasRoleByToken(ctx, token, "editor") {
		t.Fatal("HasRoleByToken(editor) = true after removal, want false")
	}
	if err = mgr.CheckRole(ctx, "u7", "editor"); !errors.Is(err, derror.ErrRoleDenied) {
		t.Fatalf("CheckRole(editor) error = %v, want ErrRoleDenied", err)
	}
}

// TestManagerDisableAccountAndDevice verifies account and device disable behavior. TestManagerDisableAccountAndDevice 验证账号和设备封禁行为。
func TestManagerDisableAccountAndDevice(t *testing.T) {
	ctx := context.Background()

	t.Run("account disable destroys session and blocks old and new login", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		token, err := mgr.Login(ctx, "u8", "web")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		if err = mgr.Disable(ctx, "u8", time.Minute, "risk"); err != nil {
			t.Fatalf("Disable() error = %v", err)
		}
		if !mgr.IsDisable(ctx, "u8") {
			t.Fatal("IsDisable() = false, want true")
		}
		if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrAccountDisabled) {
			t.Fatalf("CheckLogin() error = %v, want ErrAccountDisabled", err)
		}
		if _, err = mgr.GetSession(ctx, "u8"); !errors.Is(err, derror.ErrSessionNotFound) {
			t.Fatalf("GetSession() error = %v, want ErrSessionNotFound", err)
		}
		if _, err = mgr.Login(ctx, "u8", "web"); !errors.Is(err, derror.ErrAccountDisabled) {
			t.Fatalf("Login() while disabled error = %v, want ErrAccountDisabled", err)
		}

		info, err := mgr.GetDisableInfo(ctx, "u8")
		if err != nil {
			t.Fatalf("GetDisableInfo() error = %v", err)
		}
		if info.DisableReason != "risk" {
			t.Fatalf("DisableReason = %q, want risk", info.DisableReason)
		}
		if err = mgr.Untie(ctx, "u8"); err != nil {
			t.Fatalf("Untie() error = %v", err)
		}
		if mgr.IsDisable(ctx, "u8") {
			t.Fatal("IsDisable() after Untie = true, want false")
		}
	})

	t.Run("concrete device disable only blocks matching device id", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		if err := mgr.DisableDeviceAndDeviceId(ctx, "u9", "web", "blocked", time.Minute); err != nil {
			t.Fatalf("DisableDeviceAndDeviceId() error = %v", err)
		}
		if _, err := mgr.Login(ctx, "u9", "web", "blocked"); !errors.Is(err, derror.ErrDeviceDisabled) {
			t.Fatalf("Login(blocked device) error = %v, want ErrDeviceDisabled", err)
		}
		token, err := mgr.Login(ctx, "u9", "web", "allowed")
		if err != nil {
			t.Fatalf("Login(allowed device) error = %v", err)
		}
		if !mgr.IsLogin(ctx, token) {
			t.Fatal("allowed device token IsLogin() = false, want true")
		}
	})
}

// TestManagerAccessProviderOverridesSessionAccess verifies provider data has priority. TestManagerAccessProviderOverridesSessionAccess 验证访问提供器数据优先于会话缓存。
func TestManagerAccessProviderOverridesSessionAccess(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{
		PermissionFunc: func(_ context.Context, subject AccessSubject) ([]string, error) {
			if subject.LoginID != "u10" {
				t.Fatalf("permission subject LoginID = %q, want u10", subject.LoginID)
			}
			return []string{"provider:read"}, nil
		},
		RoleFunc: func(_ context.Context, subject AccessSubject) ([]string, error) {
			if subject.AuthType == "" {
				t.Fatal("role subject AuthType is empty")
			}
			return []string{"provider-role"}, nil
		},
	})

	token, err := mgr.Login(ctx, "u10")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "u10", []string{"session:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if err = mgr.AddRoles(ctx, "u10", []string{"session-role"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}

	if !mgr.HasPermission(ctx, "u10", "provider:read") {
		t.Fatal("HasPermission(provider:read) = false, want true")
	}
	if mgr.HasPermission(ctx, "u10", "session:read") {
		t.Fatal("HasPermission(session:read) = true, want provider override")
	}
	if !mgr.HasRoleByToken(ctx, token, "provider-role") {
		t.Fatal("HasRoleByToken(provider-role) = false, want true")
	}
	if mgr.HasRoleByToken(ctx, token, "session-role") {
		t.Fatal("HasRoleByToken(session-role) = true, want provider override")
	}
}

// TestManagerActiveTimeoutMarksTokenState verifies inactive tokens keep the active-timeout cause. TestManagerActiveTimeoutMarksTokenState 验证不活跃 Token 会保留活跃超时原因。
func TestManagerActiveTimeoutMarksTokenState(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.ActiveTimeout = 1
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "u11")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	storage, ok := mgr.GetStorage().(*managerTestStorage)
	if !ok {
		t.Fatalf("storage type = %T, want *managerTestStorage", mgr.GetStorage())
	}
	activeKey := mgr.getActiveKey(token)
	if err = storage.Set(ctx, activeKey, time.Now().Add(-2*time.Second).Unix(), time.Minute); err != nil {
		t.Fatalf("Set(active marker) error = %v", err)
	}

	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrActiveTimeout) {
		t.Fatalf("CheckLogin() error = %v, want ErrActiveTimeout", err)
	}
	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrActiveTimeout) {
		t.Fatalf("second CheckLogin() error = %v, want persisted ErrActiveTimeout", err)
	}
}

// TestManagerSessionTerminalQueries verifies terminal lookup, filtering, traversal, and search. TestManagerSessionTerminalQueries 验证终端查询、过滤、遍历和搜索。
func TestManagerSessionTerminalQueries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
		cfg.AutoRenew = false
	})

	webA, err := mgr.Login(ctx, "u12", "web", "a")
	if err != nil {
		t.Fatalf("Login(web/a) error = %v", err)
	}
	webB, err := mgr.Login(ctx, "u12", "web", "b")
	if err != nil {
		t.Fatalf("Login(web/b) error = %v", err)
	}
	mobileA, err := mgr.Login(ctx, "u12", "mobile", "a")
	if err != nil {
		t.Fatalf("Login(mobile/a) error = %v", err)
	}

	webTokens, err := mgr.GetTokenValueListByDevice(ctx, "u12", " web ")
	if err != nil {
		t.Fatalf("GetTokenValueListByDevice() error = %v", err)
	}
	if !sameStrings(webTokens, []string{webA, webB}) {
		t.Fatalf("web tokens = %v, want [%s %s]", webTokens, webA, webB)
	}
	concreteTokens, err := mgr.GetTokenValueListByDeviceAndDeviceId(ctx, "u12", "web", "b")
	if err != nil {
		t.Fatalf("GetTokenValueListByDeviceAndDeviceId() error = %v", err)
	}
	if !sameStrings(concreteTokens, []string{webB}) {
		t.Fatalf("concrete tokens = %v, want [%s]", concreteTokens, webB)
	}

	storage := requireManagerTestStorage(t, mgr)
	if err = storage.Delete(ctx, mgr.getTokenKey(webA)); err != nil {
		t.Fatalf("Delete(token key) error = %v", err)
	}
	aliveWebTokens, err := mgr.GetTokenValueListByDevice(ctx, "u12", "web", true)
	if err != nil {
		t.Fatalf("GetTokenValueListByDevice(checkAlive) error = %v", err)
	}
	if !sameStrings(aliveWebTokens, []string{webB}) {
		t.Fatalf("alive web tokens = %v, want [%s]", aliveWebTokens, webB)
	}
	count, err := mgr.GetOnlineTerminalCount(ctx, "u12")
	if err != nil {
		t.Fatalf("GetOnlineTerminalCount() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("online count = %d, want 2", count)
	}
	webCount, err := mgr.GetOnlineTerminalCountByDevice(ctx, "u12", "web")
	if err != nil {
		t.Fatalf("GetOnlineTerminalCountByDevice() error = %v", err)
	}
	if webCount != 1 {
		t.Fatalf("web online count = %d, want 1", webCount)
	}
	concreteCount, err := mgr.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, "u12", "mobile", "a")
	if err != nil {
		t.Fatalf("GetOnlineTerminalCountByDeviceAndDeviceId() error = %v", err)
	}
	if concreteCount != 1 {
		t.Fatalf("mobile/a online count = %d, want 1", concreteCount)
	}

	webTerms, err := mgr.GetTerminalListByLoginID(ctx, "u12", "web")
	if err != nil {
		t.Fatalf("GetTerminalListByLoginID(web) error = %v", err)
	}
	if len(webTerms) != 2 {
		t.Fatalf("web terminal count = %d, want 2 including dead session entry", len(webTerms))
	}
	mobileInfo, err := mgr.GetTerminalInfoByToken(ctx, mobileA)
	if err != nil {
		t.Fatalf("GetTerminalInfoByToken() error = %v", err)
	}
	if mobileInfo.LoginID != "u12" || mobileInfo.Device != "mobile" || mobileInfo.DeviceId != "a" {
		t.Fatalf("terminal info = %+v, want mobile/a terminal", mobileInfo)
	}

	visited := make([]string, 0, 2)
	if err = mgr.ForEachTerminal(ctx, "u12", func(terminal TerminalInfo) bool {
		visited = append(visited, terminal.Token)
		return len(visited) < 2
	}); err != nil {
		t.Fatalf("ForEachTerminal() error = %v", err)
	}
	if len(visited) != 2 {
		t.Fatalf("visited count = %d, want early stop after 2", len(visited))
	}
	if err = mgr.ForEachTerminal(ctx, "u12", nil); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("ForEachTerminal(nil) error = %v, want ErrInvalidParam", err)
	}

	tokenSearch, err := mgr.SearchTokenValue(ctx, "u12-web", 0, -1)
	if err != nil {
		t.Fatalf("SearchTokenValue() error = %v", err)
	}
	if !sameStrings(tokenSearch, []string{webB}) {
		t.Fatalf("token search = %v, want [%s]", tokenSearch, webB)
	}
	sessionSearch, err := mgr.SearchSessionId(ctx, "u1", 0, -1)
	if err != nil {
		t.Fatalf("SearchSessionId() error = %v", err)
	}
	if !sameStrings(sessionSearch, []string{"u12"}) {
		t.Fatalf("session search = %v, want [u12]", sessionSearch)
	}
	pagedSessions, err := mgr.SearchSessionId(ctx, "u1", 1, 1)
	if err != nil {
		t.Fatalf("SearchSessionId(page) error = %v", err)
	}
	if len(pagedSessions) != 0 {
		t.Fatalf("paged sessions = %v, want empty page after first result", pagedSessions)
	}
}

// TestManagerDeviceScopeConcurrency verifies device scoped overflow and replacement. TestManagerDeviceScopeConcurrency 验证设备作用域的超限和替换策略。
func TestManagerDeviceScopeConcurrency(t *testing.T) {
	ctx := context.Background()

	t.Run("device max count only removes oldest within that device", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
			cfg.ConcurrencyScope = config.ConcurrencyScopeDevice
			cfg.MaxLoginCount = 2
			cfg.OverflowLogoutMode = config.LogoutModeKickout
			cfg.AutoRenew = false
		})

		web1, err := mgr.Login(ctx, "u13", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		web2, err := mgr.Login(ctx, "u13", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}
		mobile1, err := mgr.Login(ctx, "u13", "mobile", "a")
		if err != nil {
			t.Fatalf("Login(mobile/a) error = %v", err)
		}
		web3, err := mgr.Login(ctx, "u13", "web", "c")
		if err != nil {
			t.Fatalf("Login(web/c) error = %v", err)
		}

		webTokens, err := mgr.GetTokenValueListByDevice(ctx, "u13", "web", true)
		if err != nil {
			t.Fatalf("GetTokenValueListByDevice(web, checkAlive) error = %v", err)
		}
		if !sameStrings(webTokens, []string{web2, web3}) {
			t.Fatalf("alive web tokens = %v, want [%s %s]", webTokens, web2, web3)
		}
		mobileTokens, err := mgr.GetTokenValueListByDevice(ctx, "u13", "mobile", true)
		if err != nil {
			t.Fatalf("GetTokenValueListByDevice(mobile, checkAlive) error = %v", err)
		}
		if !sameStrings(mobileTokens, []string{mobile1}) {
			t.Fatalf("alive mobile tokens = %v, want [%s]", mobileTokens, mobile1)
		}
		if err = mgr.CheckLogin(ctx, web1); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("oldest web token CheckLogin() error = %v, want ErrTokenKickout", err)
		}
	})

	t.Run("non concurrent device scope keeps other devices alive", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = false
			cfg.ConcurrencyScope = config.ConcurrencyScopeDevice
			cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
			cfg.AutoRenew = false
		})

		web1, err := mgr.Login(ctx, "u14", "web", "a")
		if err != nil {
			t.Fatalf("Login(web/a) error = %v", err)
		}
		mobile1, err := mgr.Login(ctx, "u14", "mobile", "a")
		if err != nil {
			t.Fatalf("Login(mobile/a) error = %v", err)
		}
		web2, err := mgr.Login(ctx, "u14", "web", "b")
		if err != nil {
			t.Fatalf("Login(web/b) error = %v", err)
		}

		if err = mgr.CheckLogin(ctx, web1); !errors.Is(err, derror.ErrTokenReplaced) {
			t.Fatalf("old web token CheckLogin() error = %v, want ErrTokenReplaced", err)
		}
		if err = mgr.CheckLogin(ctx, mobile1); err != nil {
			t.Fatalf("mobile token CheckLogin() error = %v, want nil", err)
		}
		if err = mgr.CheckLogin(ctx, web2); err != nil {
			t.Fatalf("new web token CheckLogin() error = %v, want nil", err)
		}
	})
}

// TestManagerServiceDisableLevel verifies service disable levels, TTL, and untie behavior. TestManagerServiceDisableLevel 验证服务封禁等级、TTL 和解封行为。
func TestManagerServiceDisableLevel(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	if err := mgr.DisableServiceLevel(ctx, "u15", "pay", 3, time.Minute, "risk"); err != nil {
		t.Fatalf("DisableServiceLevel() error = %v", err)
	}
	if !mgr.IsDisableService(ctx, "u15", "pay") {
		t.Fatal("IsDisableService() = false, want true")
	}
	if !mgr.IsDisableServiceLevel(ctx, "u15", "pay", 2) {
		t.Fatal("IsDisableServiceLevel(level 2) = false, want true")
	}
	if mgr.IsDisableServiceLevel(ctx, "u15", "pay", 4) {
		t.Fatal("IsDisableServiceLevel(level 4) = true, want false")
	}
	if err := mgr.CheckDisableService(ctx, "u15", "profile", "pay"); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("CheckDisableService() error = %v, want ErrServiceDisabled", err)
	}
	if err := mgr.CheckDisableServiceLevel(ctx, "u15", "pay", 3); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("CheckDisableServiceLevel(level 3) error = %v, want ErrServiceDisabled", err)
	}
	if err := mgr.CheckDisableServiceLevel(ctx, "u15", "pay", 4); err != nil {
		t.Fatalf("CheckDisableServiceLevel(level 4) error = %v, want nil", err)
	}

	info, err := mgr.GetDisableServiceInfo(ctx, "u15", " pay ")
	if err != nil {
		t.Fatalf("GetDisableServiceInfo() error = %v", err)
	}
	if info.Service != "pay" || info.Level != 3 || info.DisableReason != "risk" {
		t.Fatalf("service disable info = %+v, want pay level 3 risk", info)
	}
	ttl, err := mgr.GetDisableServiceTTL(ctx, "u15", "pay")
	if err != nil {
		t.Fatalf("GetDisableServiceTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("service disable ttl = %d, want 1..60", ttl)
	}

	if err = mgr.UntieService(ctx, "u15", "pay"); err != nil {
		t.Fatalf("UntieService() error = %v", err)
	}
	if mgr.IsDisableService(ctx, "u15", "pay") {
		t.Fatal("IsDisableService() after UntieService = true, want false")
	}
	ttl, err = mgr.GetDisableServiceTTL(ctx, "u15", "pay")
	if err != nil {
		t.Fatalf("GetDisableServiceTTL(after untie) error = %v", err)
	}
	if ttl != -2 {
		t.Fatalf("service disable ttl after untie = %d, want -2", ttl)
	}
	if _, err = mgr.GetDisableServiceInfo(ctx, "u15", "pay"); !errors.Is(err, derror.ErrServiceNotDisabled) {
		t.Fatalf("GetDisableServiceInfo(after untie) error = %v, want ErrServiceNotDisabled", err)
	}
}

// newTestManager builds a manager with isolated test components. newTestManager 使用隔离测试组件构建管理器。
func newTestManager(t *testing.T, mutate func(*config.Config)) *Manager {
	t.Helper()
	return newTestManagerWithAccessProvider(t, mutate, nil)
}

func requireManagerTestStorage(t *testing.T, mgr *Manager) *managerTestStorage {
	t.Helper()
	storage, ok := mgr.GetStorage().(*managerTestStorage)
	if !ok {
		t.Fatalf("storage type = %T, want *managerTestStorage", mgr.GetStorage())
	}
	return storage
}

// newTestManagerWithAccessProvider builds a manager with a custom access provider. newTestManagerWithAccessProvider 使用自定义访问提供器构建管理器。
func newTestManagerWithAccessProvider(t *testing.T, mutate func(*config.Config), provider AccessProvider) *Manager {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.AsyncEvent = false
	cfg.AutoRenew = false
	cfg.RenewInterval = config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
	if mutate != nil {
		mutate(cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("test config invalid: %v", err)
	}

	mgr := NewManager(
		cfg,
		&managerTestGenerator{},
		newManagerTestStorage(),
		managerTestCodec{},
		adapter.NewNopLogger(),
		nil,
		provider,
	)
	t.Cleanup(mgr.CloseManager)
	return mgr
}

type managerTestGenerator struct {
	mu  sync.Mutex
	seq int
}

func (g *managerTestGenerator) Generate(loginID, device, deviceID string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	return fmt.Sprintf("token-%s-%s-%s-%d", loginID, device, deviceID, g.seq), nil
}

type managerTestCodec struct{}

func (managerTestCodec) Name() string { return "json-test" }

func (managerTestCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

func (managerTestCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

type managerTestStorage struct {
	mu    sync.RWMutex
	items map[string]managerTestStorageItem
}

type managerTestStorageItem struct {
	value    any
	expireAt time.Time
}

func newManagerTestStorage() *managerTestStorage {
	return &managerTestStorage{items: make(map[string]managerTestStorageItem)}
}

func (s *managerTestStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expireAt time.Time
	if expiration > 0 {
		expireAt = time.Now().Add(expiration)
	}
	s.items[key] = managerTestStorageItem{value: value, expireAt: expireAt}
	return nil
}

func (s *managerTestStorage) Get(_ context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	if item.expired() {
		delete(s.items, key)
		return nil, nil
	}
	return item.value, nil
}

func (s *managerTestStorage) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

func (s *managerTestStorage) Exists(_ context.Context, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return false
	}
	if item.expired() {
		delete(s.items, key)
		return false
	}
	return true
}

func (s *managerTestStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok || item.expired() {
		delete(s.items, key)
		return derror.ErrInvalidToken
	}
	if expiration > 0 {
		item.expireAt = time.Now().Add(expiration)
	} else {
		item.expireAt = time.Time{}
	}
	s.items[key] = item
	return nil
}

func (s *managerTestStorage) TTL(_ context.Context, key string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return adapter.TTLNotFound, nil
	}
	if item.expired() {
		delete(s.items, key)
		return adapter.TTLNotFound, nil
	}
	if item.expireAt.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	ttl := time.Until(item.expireAt)
	if ttl <= 0 {
		delete(s.items, key)
		return adapter.TTLNotFound, nil
	}
	return ttl, nil
}

func (s *managerTestStorage) Ping(context.Context) error { return nil }

func (s *managerTestStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	delete(s.items, key)
	if item.expired() {
		return nil, nil
	}
	return item.value, nil
}

func (s *managerTestStorage) Keys(_ context.Context, pattern string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.items))
	for key, item := range s.items {
		if item.expired() {
			delete(s.items, key)
			continue
		}
		if matchManagerTestPattern(pattern, key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *managerTestStorage) Clear(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[string]managerTestStorageItem)
	return nil
}

func (item managerTestStorageItem) expired() bool {
	return !item.expireAt.IsZero() && time.Now().After(item.expireAt)
}

func matchManagerTestPattern(pattern, value string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}
	if !strings.HasPrefix(value, parts[0]) {
		return false
	}
	value = strings.TrimPrefix(value, parts[0])
	for _, part := range parts[1 : len(parts)-1] {
		idx := strings.Index(value, part)
		if idx < 0 {
			return false
		}
		value = value[idx+len(part):]
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(value, last)
}

func sameStrings(got, want []string) bool {
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
