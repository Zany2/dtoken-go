package manager

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

func TestManagerLoginWithOptionsStoresOverrides(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 600
		cfg.IsConcurrent = true
		cfg.IsShare = true
	})

	token, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:       "opt-user",
		Device:        "web",
		DeviceID:      "browser-1",
		Timeout:       90 * time.Second,
		ActiveTimeout: 30 * time.Second,
		Token:         "custom-token",
		Extra:         map[string]any{"trace": "token-extra"},
		TerminalExtra: map[string]any{"terminal": "extra"},
	})
	if err != nil {
		t.Fatalf("LoginWithOptions() error = %v", err)
	}
	if token != "custom-token" {
		t.Fatalf("LoginWithOptions() token = %q, want custom-token", token)
	}

	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenInfo() error = %v", err)
	}
	if info.Timeout != 90 || info.ActiveTimeout != 30 {
		t.Fatalf("TokenInfo timeout = %d active = %d, want 90 and 30", info.Timeout, info.ActiveTimeout)
	}
	if info.Extra["trace"] != "token-extra" {
		t.Fatalf("TokenInfo.Extra = %+v, want trace", info.Extra)
	}

	terminal, err := mgr.GetTerminalInfoByToken(ctx, token)
	if err != nil {
		t.Fatalf("GetTerminalInfoByToken() error = %v", err)
	}
	if terminal.Extra["terminal"] != "extra" {
		t.Fatalf("TerminalInfo.Extra = %+v, want terminal extra", terminal.Extra)
	}
}

func TestManagerLoginWithOptionsOverridesConcurrencyPolicy(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = true
		cfg.MaxLoginCount = config.NoLimit
	})

	isShare := false
	maxLoginCount := int64(1)
	overflowMode := config.LogoutModeKickout

	first, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:            "policy-user",
		Device:             "web",
		DeviceID:           "a",
		IsShare:            &isShare,
		MaxLoginCount:      &maxLoginCount,
		OverflowLogoutMode: &overflowMode,
	})
	if err != nil {
		t.Fatalf("first LoginWithOptions() error = %v", err)
	}
	second, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:            "policy-user",
		Device:             "mobile",
		DeviceID:           "b",
		IsShare:            &isShare,
		MaxLoginCount:      &maxLoginCount,
		OverflowLogoutMode: &overflowMode,
	})
	if err != nil {
		t.Fatalf("second LoginWithOptions() error = %v", err)
	}
	if first == second {
		t.Fatalf("LoginWithOptions() reused token %q, want new token", second)
	}
	if err = mgr.CheckLogin(ctx, first); !errors.Is(err, derror.ErrTokenKickout) {
		t.Fatalf("first CheckLogin() error = %v, want ErrTokenKickout", err)
	}
	if err = mgr.CheckLogin(ctx, second); err != nil {
		t.Fatalf("second CheckLogin() error = %v", err)
	}
}

func TestManagerSessionData(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	if _, err := mgr.Login(ctx, "session-user"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := mgr.SetSessionValue(ctx, "session-user", "theme", "dark"); err != nil {
		t.Fatalf("SetSessionValue() error = %v", err)
	}
	value, ok, err := mgr.GetSessionValue(ctx, "session-user", "theme")
	if err != nil {
		t.Fatalf("GetSessionValue() error = %v", err)
	}
	if !ok || value != "dark" {
		t.Fatalf("GetSessionValue() = %v, %v, want dark, true", value, ok)
	}
	if err = mgr.DeleteSessionValue(ctx, "session-user", "theme"); err != nil {
		t.Fatalf("DeleteSessionValue() error = %v", err)
	}
	value, ok, err = mgr.GetSessionValue(ctx, "session-user", "theme")
	if err != nil {
		t.Fatalf("GetSessionValue(after delete) error = %v", err)
	}
	if ok || value != nil {
		t.Fatalf("GetSessionValue(after delete) = %v, %v, want nil, false", value, ok)
	}

	if err = mgr.SetSessionValue(ctx, "", "theme", "dark"); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("SetSessionValue(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if _, _, err = mgr.GetSessionValue(ctx, "session-user", ""); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("GetSessionValue(empty key) error = %v, want ErrInvalidParam", err)
	}
}

func TestManagerTerminate(t *testing.T) {
	ctx := context.Background()

	t.Run("default logout by token", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		token, err := mgr.Login(ctx, "term-user", "web")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		if err = mgr.Terminate(ctx, TerminateOptions{Token: token}); err != nil {
			t.Fatalf("Terminate() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("CheckLogin() error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("kickout by device id", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		first, err := mgr.Login(ctx, "term-device-user", "web", "a")
		if err != nil {
			t.Fatalf("first Login() error = %v", err)
		}
		second, err := mgr.Login(ctx, "term-device-user", "web", "b")
		if err != nil {
			t.Fatalf("second Login() error = %v", err)
		}
		if err = mgr.Terminate(ctx, TerminateOptions{
			Action:   TerminateActionKickout,
			LoginID:  "term-device-user",
			Device:   "web",
			DeviceID: "a",
		}); err != nil {
			t.Fatalf("Terminate(kickout device id) error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, first); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("first CheckLogin() error = %v, want ErrTokenKickout", err)
		}
		if err = mgr.CheckLogin(ctx, second); err != nil {
			t.Fatalf("second CheckLogin() error = %v", err)
		}
	})

	t.Run("invalid options", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		if err := mgr.Terminate(ctx, TerminateOptions{}); !errors.Is(err, derror.ErrIDIsEmpty) {
			t.Fatalf("Terminate(empty) error = %v, want ErrIDIsEmpty", err)
		}
		if err := mgr.Terminate(ctx, TerminateOptions{Token: "x", Action: "unknown"}); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("Terminate(invalid action) error = %v, want ErrInvalidParam", err)
		}
	})
}

func TestManagerStrategy(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	strategy := mgr.GetStrategy()
	if strategy == nil {
		t.Fatal("GetStrategy() = nil, want default strategy")
	}
	if !strategy.PermissionMatcher("order/*", "order/read") {
		t.Fatal("default PermissionMatcher(order/*, order/read) = false, want true")
	}

	customRoleMatcher := func(pattern, role string) bool {
		return strings.EqualFold(pattern, role)
	}
	customMgr := newTestManagerWithStrategy(t, &Strategy{RoleMatcher: customRoleMatcher})
	if _, err := customMgr.Login(ctx, "strategy-user"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := customMgr.AddRoles(ctx, "strategy-user", []string{"Admin"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if !customMgr.HasRole(ctx, "strategy-user", "admin") {
		t.Fatal("HasRole(admin) = false, want custom role matcher to ignore case")
	}
}

func TestManagerPermissionShortcutMatrix(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	token, err := mgr.Login(ctx, "permission-shortcuts", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "permission-shortcuts", []string{"article:*", "order:read", "profile:view"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	permissions, err := mgr.GetPermissionsByToken(ctx, token)
	if err != nil {
		t.Fatalf("GetPermissionsByToken() error = %v", err)
	}
	if !sameStrings(permissions, []string{"article:*", "order:read", "profile:view"}) {
		t.Fatalf("GetPermissionsByToken() = %v, want stored permissions", permissions)
	}
	if err = mgr.CheckPermission(ctx, "permission-shortcuts", "article:create"); err != nil {
		t.Fatalf("CheckPermission(wildcard) error = %v", err)
	}
	if err = mgr.CheckPermissionAnd(ctx, "permission-shortcuts", []string{"article:update", "order:read"}); err != nil {
		t.Fatalf("CheckPermissionAnd() error = %v", err)
	}
	if err = mgr.CheckPermissionAndByToken(ctx, token, []string{"article:delete", "profile:view"}); err != nil {
		t.Fatalf("CheckPermissionAndByToken() error = %v", err)
	}
	if err = mgr.CheckPermissionOrByToken(ctx, token, []string{"missing", "order:read"}); err != nil {
		t.Fatalf("CheckPermissionOrByToken() error = %v", err)
	}
	if !mgr.HasPermissionsOr(ctx, "permission-shortcuts", []string{"missing", "profile:view"}) {
		t.Fatal("HasPermissionsOr() = false, want true")
	}
	if mgr.HasPermissionsOr(ctx, "permission-shortcuts", []string{"missing", ""}) {
		t.Fatal("HasPermissionsOr(missing) = true, want false")
	}
	if err = mgr.RemovePermissions(ctx, "permission-shortcuts", []string{"order:read"}); err != nil {
		t.Fatalf("RemovePermissions() error = %v", err)
	}
	if err = mgr.CheckPermissionOr(ctx, "permission-shortcuts", []string{"missing", "order:read"}); !errors.Is(err, derror.ErrPermissionDenied) {
		t.Fatalf("CheckPermissionOr(removed) error = %v, want ErrPermissionDenied", err)
	}
	if err = mgr.CheckPermissionAnd(ctx, "permission-shortcuts", []string{"article:update", "order:read"}); !errors.Is(err, derror.ErrPermissionDenied) {
		t.Fatalf("CheckPermissionAnd(removed) error = %v, want ErrPermissionDenied", err)
	}
	if err = mgr.CheckPermission(ctx, "permission-shortcuts", ""); !errors.Is(err, derror.ErrPermissionDenied) {
		t.Fatalf("CheckPermission(empty) error = %v, want ErrPermissionDenied", err)
	}
}

func TestManagerRoleShortcutMatrix(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	token, err := mgr.Login(ctx, "role-shortcuts", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddRoles(ctx, "role-shortcuts", []string{"admin", "editor", "auditor"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}

	roles, err := mgr.GetRoles(ctx, "role-shortcuts")
	if err != nil {
		t.Fatalf("GetRoles() error = %v", err)
	}
	if !sameStrings(roles, []string{"admin", "editor", "auditor"}) {
		t.Fatalf("GetRoles() = %v, want stored roles", roles)
	}
	if err = mgr.CheckRoleByToken(ctx, token, "admin"); err != nil {
		t.Fatalf("CheckRoleByToken() error = %v", err)
	}
	if err = mgr.CheckRoleAnd(ctx, "role-shortcuts", []string{"admin", "editor"}); err != nil {
		t.Fatalf("CheckRoleAnd() error = %v", err)
	}
	if err = mgr.CheckRoleAndByToken(ctx, token, []string{"admin", "auditor"}); err != nil {
		t.Fatalf("CheckRoleAndByToken() error = %v", err)
	}
	if err = mgr.CheckRoleOr(ctx, "role-shortcuts", []string{"missing", "editor"}); err != nil {
		t.Fatalf("CheckRoleOr() error = %v", err)
	}
	if err = mgr.CheckRoleOrByToken(ctx, token, []string{"missing", "auditor"}); err != nil {
		t.Fatalf("CheckRoleOrByToken() error = %v", err)
	}
	if !mgr.HasRolesAndByToken(ctx, token, []string{"admin", "editor"}) {
		t.Fatal("HasRolesAndByToken() = false, want true")
	}
	if !mgr.HasRolesOr(ctx, "role-shortcuts", []string{"missing", "auditor"}) {
		t.Fatal("HasRolesOr() = false, want true")
	}
	if mgr.HasRolesOr(ctx, "role-shortcuts", []string{"missing", ""}) {
		t.Fatal("HasRolesOr(missing) = true, want false")
	}
	if err = mgr.RemoveRoles(ctx, "role-shortcuts", []string{"editor"}); err != nil {
		t.Fatalf("RemoveRoles() error = %v", err)
	}
	if err = mgr.CheckRoleOr(ctx, "role-shortcuts", []string{"missing", "editor"}); !errors.Is(err, derror.ErrRoleDenied) {
		t.Fatalf("CheckRoleOr(removed) error = %v, want ErrRoleDenied", err)
	}
	if err = mgr.CheckRoleAnd(ctx, "role-shortcuts", []string{"admin", "editor"}); !errors.Is(err, derror.ErrRoleDenied) {
		t.Fatalf("CheckRoleAnd(removed) error = %v, want ErrRoleDenied", err)
	}
	if err = mgr.CheckRole(ctx, "role-shortcuts", ""); !errors.Is(err, derror.ErrRoleDenied) {
		t.Fatalf("CheckRole(empty) error = %v, want ErrRoleDenied", err)
	}
}

func TestManagerLifecycleHelpers(t *testing.T) {
	logger := &managerLifecycleTestLogger{}
	pool := &managerLifecycleTestPool{running: 1, capacity: 2, usage: 0.5}
	mgr := newTestManagerWithRuntime(t, pool, logger)

	mgr.StartRenewPoolStatusLogger(5 * time.Millisecond)
	waitForManagerTest(t, 100*time.Millisecond, func() bool {
		return logger.infofCount() > 0
	})
	mgr.CloseManager()
	mgr.CloseManager()

	if !pool.stopped() {
		t.Fatal("pool stopped = false, want CloseManager to stop pool")
	}
	if !logger.flushed() || !logger.closed() {
		t.Fatalf("logger flushed/closed = %v/%v, want true/true", logger.flushed(), logger.closed())
	}
	if _, ok := managerBackgrounds.Load(mgr); ok {
		t.Fatal("manager background state still exists after CloseManager")
	}
}

func newTestManagerWithRuntime(t *testing.T, pool adapter.Pool, logger adapter.Log) *Manager {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.AsyncEvent = false
	cfg.AutoRenew = false
	cfg.RenewInterval = config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
	applyManagerTestStorageConfig(t, cfg)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("test config invalid: %v", err)
	}

	return NewManager(
		cfg,
		&managerTestGenerator{},
		newManagerTestStorageForTest(t, cfg),
		managerTestCodec{},
		logger,
		pool,
		nil,
	)
}

func newTestManagerWithStrategy(t *testing.T, strategy *Strategy) *Manager {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.AsyncEvent = false
	cfg.AutoRenew = false
	cfg.RenewInterval = config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
	applyManagerTestStorageConfig(t, cfg)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("test config invalid: %v", err)
	}

	mgr := NewManager(
		cfg,
		&managerTestGenerator{},
		newManagerTestStorageForTest(t, cfg),
		managerTestCodec{},
		adapter.NewNopLogger(),
		nil,
		nil,
		WithStrategy(strategy),
	)
	t.Cleanup(mgr.CloseManager)
	return mgr
}

type managerLifecycleTestPool struct {
	mu       sync.Mutex
	running  int
	capacity int
	usage    float64
	stop     bool
}

func (p *managerLifecycleTestPool) Submit(task func()) error {
	go task()
	return nil
}

func (p *managerLifecycleTestPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stop = true
}

func (p *managerLifecycleTestPool) Stats() (int, int, float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running, p.capacity, p.usage
}

func (p *managerLifecycleTestPool) stopped() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stop
}

type managerLifecycleTestLogger struct {
	mu         sync.Mutex
	infof      int
	flush      bool
	close      bool
	lastFormat string
}

func (l *managerLifecycleTestLogger) Print(v ...any)                 {}
func (l *managerLifecycleTestLogger) Printf(format string, v ...any) {}
func (l *managerLifecycleTestLogger) Debug(v ...any)                 {}
func (l *managerLifecycleTestLogger) Debugf(format string, v ...any) {}
func (l *managerLifecycleTestLogger) Info(v ...any)                  {}
func (l *managerLifecycleTestLogger) Warn(v ...any)                  {}
func (l *managerLifecycleTestLogger) Warnf(format string, v ...any)  {}
func (l *managerLifecycleTestLogger) Error(v ...any)                 {}
func (l *managerLifecycleTestLogger) Errorf(format string, v ...any) {}

func (l *managerLifecycleTestLogger) Infof(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infof++
	l.lastFormat = format
}

func (l *managerLifecycleTestLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.close = true
}

func (l *managerLifecycleTestLogger) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flush = true
}

func (l *managerLifecycleTestLogger) SetLevel(adapter.LogLevel) {}
func (l *managerLifecycleTestLogger) SetPrefix(string)          {}
func (l *managerLifecycleTestLogger) SetStdout(bool)            {}
func (l *managerLifecycleTestLogger) LogPath() string           { return "" }
func (l *managerLifecycleTestLogger) DropCount() uint64         { return 0 }

func (l *managerLifecycleTestLogger) infofCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.infof
}

func (l *managerLifecycleTestLogger) flushed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flush
}

func (l *managerLifecycleTestLogger) closed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.close
}
