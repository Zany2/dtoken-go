// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestManagerConstructorDefaults verifies constructor defaults and nil option handling. TestManagerConstructorDefaults 验证构造器默认值及 nil 选项处理。
func TestManagerConstructorDefaults(t *testing.T) {
	manager := NewManager(nil, nil, nil, nil, nil, nil, nil, nil, WithStrategy(nil))
	if manager.GetConfig() == nil {
		t.Fatal("NewManager(nil) config is nil")
	}
	if manager.GetLogger() == nil {
		t.Fatal("NewManager(nil logger) logger is nil")
	}
	if manager.GetStrategy() == nil || manager.GetStrategy().PermissionMatcher == nil || manager.GetStrategy().RoleMatcher == nil || manager.GetStrategy().CreateSession == nil {
		t.Fatal("NewManager() did not install a complete default strategy")
	}
	if manager.IsClosed() {
		t.Fatal("new manager IsClosed() = true, want false")
	}
	manager.CloseManager()
	if !manager.IsClosed() {
		t.Fatal("closed manager IsClosed() = false, want true")
	}

	var nilManager *Manager
	if !nilManager.IsClosed() {
		t.Fatal("nil manager IsClosed() = false, want true")
	}
}

// TestManagerKeyAndDeviceHelpers verifies namespaced key construction and input normalization. TestManagerKeyAndDeviceHelpers 验证命名空间键构造及输入归一化。
func TestManagerKeyAndDeviceHelpers(t *testing.T) {
	manager := newTestManager(t, func(cfg *config.Config) {
		cfg.AuthType = "custom:"
		cfg.KeyPrefix = "prefix:"
	})

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "token", got: manager.getTokenKey("t1"), want: "prefix:custom:token:t1"},
		{name: "session", got: manager.getSessionKey("u1"), want: "prefix:custom:session:u1"},
		{name: "renew", got: manager.getRenewKey("t1"), want: "prefix:custom:renew:t1"},
		{name: "active", got: manager.getActiveKey("t1"), want: "prefix:custom:active:t1"},
		{name: "refresh", got: manager.getRefreshTokenKey("r1"), want: "prefix:custom:refresh:r1"},
		{name: "token refresh", got: manager.getTokenRefreshKey("t1"), want: "prefix:custom:refresh:token:t1"},
		{name: "account disable", got: manager.getDisableKey("u1"), want: "prefix:custom:disable:u1"},
		{name: "service disable", got: manager.getDisableServiceKey("u1", "pay"), want: "prefix:custom:disable:service:u1:pay"},
		{name: "device disable", got: manager.getDisableDeviceKey("u1", "web"), want: "prefix:custom:disable:device:u1:web"},
		{name: "device id disable", got: manager.getDisableDeviceAndDeviceIDKey("u1", "web", "a"), want: "prefix:custom:disable:device:id:u1:web:a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("key = %q, want %q", tt.got, tt.want)
			}
		})
	}

	for _, tt := range []struct {
		args       []string
		wantDevice string
		wantID     string
	}{
		{args: nil, wantDevice: "", wantID: ""},
		{args: []string{" web ", " browser-1 "}, wantDevice: "web", wantID: "browser-1"},
		{args: []string{"web"}, wantDevice: "web", wantID: ""},
		{args: []string{" ", " "}, wantDevice: "", wantID: ""},
		{args: []string{"web", "a", "ignored"}, wantDevice: "web", wantID: "a"},
	} {
		device, deviceID := manager.getDeviceAndDeviceID(tt.args...)
		if device != tt.wantDevice || deviceID != tt.wantID {
			t.Fatalf("getDeviceAndDeviceID(%q) = %q/%q, want %q/%q", tt.args, device, deviceID, tt.wantDevice, tt.wantID)
		}
	}
}

// TestManagerDurationResolution verifies timeout conversion and override precedence. TestManagerDurationResolution 验证超时转换及覆盖优先级。
func TestManagerDurationResolution(t *testing.T) {
	manager := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 120
		cfg.ActiveTimeout = 45
	})

	if got := manager.getExpiration(); got != 120*time.Second {
		t.Fatalf("getExpiration() = %s, want 120s", got)
	}
	manager.config.Timeout = config.NoLimit
	if got := manager.getExpiration(); got != 0 {
		t.Fatalf("getExpiration() with NoLimit = %s, want 0", got)
	}
	manager.config.Timeout = 120

	for _, tt := range []struct {
		name    string
		timeout time.Duration
		want    int64
	}{
		{name: "zero", timeout: 0, want: config.NoLimit},
		{name: "negative", timeout: -time.Second, want: config.NoLimit},
		{name: "subsecond", timeout: 500 * time.Millisecond, want: 1},
		{name: "exact second", timeout: time.Second, want: 1},
		{name: "round up", timeout: 1500 * time.Millisecond, want: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := manager.timeoutToSeconds(tt.timeout); got != tt.want {
				t.Fatalf("timeoutToSeconds(%s) = %d, want %d", tt.timeout, got, tt.want)
			}
		})
	}

	for _, tt := range []struct {
		name string
		info *TokenInfo
		want time.Duration
	}{
		{name: "nil uses config", info: nil, want: 120 * time.Second},
		{name: "zero uses config", info: &TokenInfo{Timeout: 0}, want: 120 * time.Second},
		{name: "limited override", info: &TokenInfo{Timeout: 7}, want: 7 * time.Second},
		{name: "unlimited override", info: &TokenInfo{Timeout: config.NoLimit}, want: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := manager.resolveTokenExpiration(tt.info); got != tt.want {
				t.Fatalf("resolveTokenExpiration() = %s, want %s", got, tt.want)
			}
		})
	}

	for _, tt := range []struct {
		name    string
		seconds int64
		want    int64
	}{
		{name: "explicit", seconds: 30, want: 30},
		{name: "unlimited override", seconds: config.NoLimit, want: 0},
		{name: "zero falls back", seconds: 0, want: 45},
		{name: "negative falls back", seconds: -2, want: 45},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := manager.resolveActiveTimeoutFromSeconds(tt.seconds); got != tt.want {
				t.Fatalf("resolveActiveTimeoutFromSeconds(%d) = %d, want %d", tt.seconds, got, tt.want)
			}
		})
	}

	for _, tt := range []struct {
		name    string
		timeout time.Duration
		want    int64
	}{
		{name: "positive", timeout: 1500 * time.Millisecond, want: 2},
		{name: "zero", timeout: 0, want: 0},
		{name: "negative", timeout: -time.Second, want: config.NoLimit},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := manager.activeTimeoutToSeconds(tt.timeout); got != tt.want {
				t.Fatalf("activeTimeoutToSeconds(%s) = %d, want %d", tt.timeout, got, tt.want)
			}
		})
	}
}

// TestManagerTTLConversions verifies public TTL sentinel conversions used by token and refresh responses. TestManagerTTLConversions 验证 Token 与刷新响应使用的 TTL 哨兵转换。
func TestManagerTTLConversions(t *testing.T) {
	for _, tt := range []struct {
		name    string
		ttl     time.Duration
		seconds int64
	}{
		{name: "not found", ttl: adapter.TTLNotFound, seconds: -2},
		{name: "no expire", ttl: adapter.TTLNoExpire, seconds: -1},
		{name: "positive", ttl: 91 * time.Second, seconds: 91},
		{name: "zero", ttl: 0, seconds: 0},
		{name: "other negative", ttl: -3 * time.Second, seconds: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTTLSeconds(tt.ttl); got != tt.seconds {
				t.Fatalf("normalizeTTLSeconds(%s) = %d, want %d", tt.ttl, got, tt.seconds)
			}
		})
	}

	for _, tt := range []struct {
		name    string
		seconds int64
		want    time.Duration
	}{
		{name: "not found", seconds: -2, want: 0},
		{name: "no expire", seconds: config.NoLimit, want: -1},
		{name: "positive", seconds: 3, want: 3 * time.Second},
		{name: "zero", seconds: 0, want: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := secondsToDuration(tt.seconds); got != tt.want {
				t.Fatalf("secondsToDuration(%d) = %s, want %s", tt.seconds, got, tt.want)
			}
		})
	}
}

// TestManagerExpirationHelpers verifies limited expiration handling and token-state TTL fallback. TestManagerExpirationHelpers 验证有限过期处理及 Token 状态 TTL 回退。
func TestManagerExpirationHelpers(t *testing.T) {
	ctx := context.Background()
	manager := newTestManager(t, func(cfg *config.Config) { cfg.Timeout = 30 })
	if err := manager.expireIfLimited(ctx, "missing", 0); err != nil {
		t.Fatalf("expireIfLimited(no limit) error = %v, want nil", err)
	}
	if err := manager.expireIfLimited(ctx, "missing", time.Second); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("expireIfLimited(missing) = %v, want ErrStorageUnavailable", err)
	}
	if err := manager.expireTokenIfLimited(ctx, "missing", time.Second); err != nil {
		t.Fatalf("expireTokenIfLimited(missing) error = %v, want nil", err)
	}

	token, err := manager.Login(ctx, "expiration-user", "web")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = manager.expireTokenIfLimited(ctx, token, 2*time.Second); err != nil {
		t.Fatalf("expireTokenIfLimited(existing) error = %v", err)
	}
	if ttl, ttlErr := manager.GetTokenTTL(ctx, token); ttlErr != nil || ttl <= 0 || ttl > 2 {
		t.Fatalf("GetTokenTTL(after expire) = %d, %v, want 1..2", ttl, ttlErr)
	}
	if ttl := manager.tokenStateExpiration(ctx, token); ttl <= 0 || ttl > 2*time.Second {
		t.Fatalf("tokenStateExpiration(existing) = %s, want positive <=2s", ttl)
	}
	if err = requireManagerTestStorage(t, manager).Set(ctx, manager.getTokenKey("no-expire-state"), "value", 0); err != nil {
		t.Fatalf("Set(no-expire-state) error = %v", err)
	}
	if ttl := manager.tokenStateExpiration(ctx, "no-expire-state"); ttl != 0 {
		t.Fatalf("tokenStateExpiration(no-expire) = %s, want 0", ttl)
	}
}

// TestManagerTokenStateHelpers verifies state mapping and logout mode persistence. TestManagerTokenStateHelpers 验证 Token 状态映射及下线模式持久化。
func TestManagerTokenStateHelpers(t *testing.T) {
	for _, tt := range []struct {
		state TokenState
		want  error
	}{
		{state: TokenStateLogout, want: derror.ErrInvalidToken},
		{state: TokenStateKickOut, want: derror.ErrTokenKickout},
		{state: TokenStateReplaced, want: derror.ErrTokenReplaced},
		{state: TokenStateActiveTimeout, want: derror.ErrActiveTimeout},
		{state: TokenState("unknown"), want: nil},
	} {
		if got := tokenStateError(tt.state); !errors.Is(got, tt.want) {
			t.Fatalf("tokenStateError(%q) = %v, want %v", tt.state, got, tt.want)
		}
	}

	ctx := context.Background()
	for _, tt := range []struct {
		name string
		mode config.LogoutMode
		want error
	}{
		{name: "logout", mode: config.LogoutModeLogout, want: derror.ErrInvalidToken},
		{name: "kickout", mode: config.LogoutModeKickout, want: derror.ErrTokenKickout},
		{name: "replace", mode: config.LogoutModeReplaced, want: derror.ErrTokenReplaced},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager := newTestManager(t, nil)
			token, err := manager.Login(ctx, "state-"+tt.name, "web")
			if err != nil {
				t.Fatalf("Login() error = %v", err)
			}
			if err = manager.applyLogoutModeToToken(ctx, token, tt.mode); err != nil {
				t.Fatalf("applyLogoutModeToToken() error = %v", err)
			}
			if err = manager.CheckLogin(ctx, token); !errors.Is(err, tt.want) {
				t.Fatalf("CheckLogin() error = %v, want %v", err, tt.want)
			}
		})
	}

	manager := newTestManager(t, nil)
	if err := manager.applyLogoutModeToToken(ctx, "missing", config.LogoutMode("invalid")); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("applyLogoutModeToToken(invalid) error = %v, want ErrInvalidParam", err)
	}
	if err := manager.setTokenState(ctx, "state-only", TokenStateKickOut, time.Minute); err != nil {
		t.Fatalf("setTokenState() error = %v", err)
	}
	value, err := requireManagerTestStorage(t, manager).Get(ctx, manager.getTokenKey("state-only"))
	if err != nil {
		t.Fatalf("Get(state-only) error = %v", err)
	}
	if got, ok := value.(string); !ok || got != string(TokenStateKickOut) {
		t.Fatalf("stored token state = %#v, want %q", value, TokenStateKickOut)
	}
}

// TestManagerStorageHelpers verifies serialization, atomic fallback, and key search pagination. TestManagerStorageHelpers 验证序列化、原子回退及键搜索分页。
func TestManagerStorageHelpers(t *testing.T) {
	ctx := context.Background()
	manager := newTestManager(t, nil)
	key := "internal-storage"
	if err := manager.saveToStorage(ctx, key, map[string]any{"value": "ok"}, time.Minute); err != nil {
		t.Fatalf("saveToStorage() error = %v", err)
	}
	if !manager.GetStorage().Exists(ctx, key) {
		t.Fatal("saveToStorage() did not persist key")
	}

	stored, err := manager.saveToStorageIfAbsent(ctx, "atomic-key", "first", time.Minute)
	if err != nil || !stored {
		t.Fatalf("saveToStorageIfAbsent(first) = %v, %v, want true, nil", stored, err)
	}
	stored, err = manager.saveToStorageIfAbsent(ctx, "atomic-key", "second", time.Minute)
	if err != nil || stored {
		t.Fatalf("saveToStorageIfAbsent(second) = %v, %v, want false, nil", stored, err)
	}

	manager.serializer = managerErrorCodec{}
	if err := manager.saveToStorage(ctx, "encode-error", "value"); !errors.Is(err, derror.ErrSerializeFailed) {
		t.Fatalf("saveToStorage(encode error) = %v, want ErrSerializeFailed", err)
	}
	if stored, err := manager.saveToStorageIfAbsent(ctx, "encode-error", "value", time.Minute); stored || !errors.Is(err, derror.ErrSerializeFailed) {
		t.Fatalf("saveToStorageIfAbsent(encode error) = %v, %v, want false and ErrSerializeFailed", stored, err)
	}

	manager = newTestManager(t, nil)
	base := newManagerTestStorage()
	manager.storage = &managerTestFailingSetStorage{managerTestStorage: base, failKey: "set-error"}
	if err := manager.saveToStorage(ctx, "set-error", "value"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("saveToStorage(storage error) = %v, want ErrStorageUnavailable", err)
	}
	manager.storage = &managerTestFailingSetStorage{managerTestStorage: base, failKey: "set-if-absent-error", failSetIfAbsent: true}
	if stored, err := manager.saveToStorageIfAbsent(ctx, "set-if-absent-error", "value", time.Minute); stored || !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("saveToStorageIfAbsent(storage error) = %v, %v, want false and ErrStorageUnavailable", stored, err)
	}

	manager = newTestManager(t, nil)
	manager.storage = &managerStorageOnly{inner: newManagerTestStorage()}
	stored, err = manager.saveToStorageIfAbsent(ctx, "fallback-key", "first", time.Minute)
	if err != nil || !stored {
		t.Fatalf("non-atomic saveToStorageIfAbsent(first) = %v, %v, want true, nil", stored, err)
	}
	stored, err = manager.saveToStorageIfAbsent(ctx, "fallback-key", "second", time.Minute)
	if err != nil || stored {
		t.Fatalf("non-atomic saveToStorageIfAbsent(second) = %v, %v, want false, nil", stored, err)
	}
	if _, err = manager.searchKeys(ctx, "*", 0, -1); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("searchKeys(non-scanner) = %v, want ErrStorageUnavailable", err)
	}

	manager = newTestManager(t, nil)
	for _, key := range []string{"search/a", "search/b", "search/c"} {
		if err := requireManagerTestStorage(t, manager).Set(ctx, key, key, 0); err != nil {
			t.Fatalf("Set(%s) error = %v", key, err)
		}
	}
	keys, err := manager.searchKeys(ctx, "search/*", -1, 2)
	if err != nil || !reflect.DeepEqual(keys, []string{"search/a", "search/b"}) {
		t.Fatalf("searchKeys(first page) = %v, %v, want [search/a search/b], nil", keys, err)
	}
	keys, err = manager.searchKeys(ctx, "search/*", 1, 0)
	if err != nil || len(keys) != 0 {
		t.Fatalf("searchKeys(size zero) = %v, %v, want empty, nil", keys, err)
	}
	keys, err = manager.searchKeys(ctx, "search/*", 0, -1)
	if err != nil || !reflect.DeepEqual(keys, []string{"search/a", "search/b", "search/c"}) {
		t.Fatalf("searchKeys(all) = %v, %v, want all keys, nil", keys, err)
	}
	keys, err = manager.searchKeys(ctx, "search/*", 99, 2)
	if err != nil || len(keys) != 0 {
		t.Fatalf("searchKeys(out of range) = %v, %v, want empty, nil", keys, err)
	}
	values, err := manager.searchValues(ctx, "search/*", "search/", 0, -1)
	if err != nil || !reflect.DeepEqual(values, []string{"a", "b", "c"}) {
		t.Fatalf("searchValues() = %v, %v, want [a b c], nil", values, err)
	}

	manager.storage = &managerScannerErrorStorage{managerTestStorage: newManagerTestStorage()}
	if _, err = manager.searchKeys(ctx, "*", 0, -1); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("searchKeys(scanner error) = %v, want ErrStorageUnavailable", err)
	}
}

// TestManagerSearchKeywordEscaping verifies wildcard characters are escaped before scanning. TestManagerSearchKeywordEscaping 验证搜索关键词中的通配符会在扫描前转义。
func TestManagerSearchKeywordEscaping(t *testing.T) {
	if got, want := escapeSearchKeyword("a\\b*?"), "a\\\\b\\*\\?"; got != want {
		t.Fatalf("escapeSearchKeyword() = %q, want %q", got, want)
	}
}

// TestManagerSessionCollectionHelpers verifies terminal and access-list collection semantics. TestManagerSessionCollectionHelpers 验证终端及权限角色集合操作语义。
func TestManagerSessionCollectionHelpers(t *testing.T) {
	session := &Session{TerminalInfos: []TerminalInfo{
		{Token: "t1", Device: "web", DeviceID: "a", Index: 1},
		{Token: "t2", Device: "mobile", DeviceID: "b", Index: 2},
		{Token: "t3", Device: "web", DeviceID: "a", Index: 3},
	}}

	if got := session.filterTerminals(nil); len(got) != 0 {
		t.Fatalf("filterTerminals(nil) returned %d terminals, want 0", len(got))
	}
	if got := session.getTerminalsByDevice("web"); !reflect.DeepEqual([]string{got[0].Token, got[1].Token}, []string{"t1", "t3"}) {
		t.Fatalf("getTerminalsByDevice(web) = %+v, want t1/t3", got)
	}
	if got := session.getTerminalsByDeviceAndDeviceID("web", "a"); len(got) != 2 {
		t.Fatalf("getTerminalsByDeviceAndDeviceID(web/a) count = %d, want 2", len(got))
	}
	if latest, ok := session.getLatestTerminalByDevice("web"); !ok || latest.Token != "t3" {
		t.Fatalf("getLatestTerminalByDevice(web) = %+v, %v, want t3, true", latest, ok)
	}
	if _, ok := session.getLatestTerminalByDevice("unknown"); ok {
		t.Fatal("getLatestTerminalByDevice(unknown) = true, want false")
	}
	if !session.hasTerminalToken("t2") || session.hasTerminalToken("") || session.hasTerminalToken("missing") {
		t.Fatal("hasTerminalToken() returned unexpected result")
	}

	removed, ok := session.removeTerminalByToken("")
	if ok || removed.Token != "" {
		t.Fatalf("removeTerminalByToken(empty) = %+v, %v, want zero, false", removed, ok)
	}
	removed, ok = session.removeTerminalByToken("t2")
	if !ok || removed.Token != "t2" || len(session.TerminalInfos) != 2 {
		t.Fatalf("removeTerminalByToken(t2) = %+v, %v; terminals = %+v", removed, ok, session.TerminalInfos)
	}
	session.TerminalInfos = append(session.TerminalInfos, TerminalInfo{Token: "t1", Device: "web", DeviceID: "a", Index: 4})
	removed, ok = session.removeLatestTerminalByToken("t1")
	if !ok || removed.Index != 4 {
		t.Fatalf("removeLatestTerminalByToken(t1) = %+v, %v, want newest t1", removed, ok)
	}
	if _, ok = session.removeLatestTerminalByToken("missing"); ok {
		t.Fatal("removeLatestTerminalByToken(missing) = true, want false")
	}

	session = &Session{TerminalInfos: []TerminalInfo{
		{Token: "a", Device: "web"},
		{Token: "b", Device: "mobile"},
		{Token: "c", Device: "web"},
	}}
	removed, ok = session.removeOldestTerminal("mobile")
	if !ok || removed.Token != "b" {
		t.Fatalf("removeOldestTerminal(mobile) = %+v, %v, want b, true", removed, ok)
	}
	removed, ok = session.removeOldestTerminal()
	if !ok || removed.Token != "a" {
		t.Fatalf("removeOldestTerminal() = %+v, %v, want a, true", removed, ok)
	}
	if _, ok = session.removeOldestTerminal("missing"); ok {
		t.Fatal("removeOldestTerminal(missing) = true, want false")
	}
	removed = session.removeAllTerminals()
	if len(removed) != 1 || len(session.TerminalInfos) != 0 {
		t.Fatalf("removeAllTerminals() = %+v; remaining = %+v", removed, session.TerminalInfos)
	}

	values := addUniqueStrings([]string{"a"}, "", "a", "b", "b")
	if !reflect.DeepEqual(values, []string{"a", "b"}) {
		t.Fatalf("addUniqueStrings() = %v, want [a b]", values)
	}
	if got := normalizeAccessValues(nil); got != nil {
		t.Fatalf("normalizeAccessValues(nil) = %#v, want nil", got)
	}
	if got := normalizeAccessValues([]string{"", "a", "a", "b"}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("normalizeAccessValues() = %v, want [a b]", got)
	}
	if got := normalizeAccessValues([]string{}); got == nil || len(got) != 0 {
		t.Fatalf("normalizeAccessValues(empty) = %#v, want non-nil empty", got)
	}
	if got := removeStrings([]string{"a", "b", "a"}, "", "a"); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("removeStrings() = %v, want [b]", got)
	}

	session.addPermissions("read", "read", "write", "")
	session.removePermissions("read")
	session.addRoles("admin", "admin", "user")
	session.removeRoles("admin")
	if !reflect.DeepEqual(session.Permissions, []string{"write"}) || !reflect.DeepEqual(session.Roles, []string{"user"}) {
		t.Fatalf("access collections = permissions:%v roles:%v, want write/user", session.Permissions, session.Roles)
	}
}

// TestManagerStrategyDefaultsAndMatchers verifies strategy normalization and wildcard matching rules. TestManagerStrategyDefaultsAndMatchers 验证策略归一化及通配符匹配规则。
func TestManagerStrategyDefaultsAndMatchers(t *testing.T) {
	if strategy := (*Strategy)(nil).normalize(); strategy == nil || strategy.PermissionMatcher == nil || strategy.RoleMatcher == nil || strategy.CreateSession == nil {
		t.Fatal("nil strategy did not normalize to complete defaults")
	}

	customSession := func(authType, loginID string, createTime int64) *Session {
		return &Session{AuthType: authType, LoginID: loginID, CreateTime: createTime}
	}
	strategy := (&Strategy{PermissionMatcher: func(pattern, permission string) bool { return pattern == "custom" && permission == "value" }, CreateSession: customSession}).normalize()
	if !strategy.PermissionMatcher("custom", "value") || strategy.PermissionMatcher("read", "read") {
		t.Fatal("custom permission matcher was not preserved")
	}
	if !strategy.RoleMatcher("admin", "admin") || strategy.RoleMatcher("admin", "user") {
		t.Fatal("default role matcher was not installed")
	}
	if got := strategy.CreateSession("auth:", "u1", 7); got == nil || got.LoginID != "u1" || got.CreateTime != 7 {
		t.Fatalf("custom CreateSession() = %+v, want custom session", got)
	}

	for _, tt := range []struct {
		pattern    string
		permission string
		want       bool
	}{
		{pattern: "*", permission: "anything", want: true},
		{pattern: "read", permission: "read", want: true},
		{pattern: "read", permission: "write", want: false},
		{pattern: "user:*", permission: "user:read", want: true},
		{pattern: "user:*", permission: "user:read:extra", want: false},
		{pattern: "user/*", permission: "user/read", want: true},
		{pattern: "user/*", permission: "user/read/extra", want: false},
		{pattern: "user:*", permission: "user/read", want: false},
	} {
		if got := defaultPermissionMatcher(tt.pattern, tt.permission); got != tt.want {
			t.Fatalf("defaultPermissionMatcher(%q, %q) = %v, want %v", tt.pattern, tt.permission, got, tt.want)
		}
	}
	if !defaultRoleMatcher("admin", "admin") || defaultRoleMatcher("admin", "user") {
		t.Fatal("defaultRoleMatcher() returned unexpected result")
	}
}

// TestManagerLoginWriteLockSerializesAndCleans verifies per-login lock serialization and registry cleanup. TestManagerLoginWriteLockSerializesAndCleans 验证账号锁串行化及注册表清理。
func TestManagerLoginWriteLockSerializesAndCleans(t *testing.T) {
	manager := newTestManager(t, nil)
	if unlock := manager.lockLoginWrite(""); unlock == nil {
		t.Fatal("lockLoginWrite(empty) returned nil unlock")
	}

	firstUnlock := manager.lockLoginWrite("lock-user")
	started := make(chan struct{})
	acquired := make(chan struct{})
	go func() {
		close(started)
		secondUnlock := manager.lockLoginWrite("lock-user")
		close(acquired)
		secondUnlock()
	}()
	<-started
	select {
	case <-acquired:
		t.Fatal("second lock acquired before first unlock")
	case <-time.After(20 * time.Millisecond):
	}
	firstUnlock()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second lock did not acquire after first unlock")
	}
	waitForManagerTest(t, time.Second, func() bool {
		manager.loginLocksMu.Lock()
		defer manager.loginLocksMu.Unlock()
		return len(manager.loginLocks) == 0
	})
}

// TestManagerSubmitAsyncFallbacks verifies goroutine fallback when no pool or submission fails. TestManagerSubmitAsyncFallbacks 验证无协程池及提交失败时的 Goroutine 回退。
func TestManagerSubmitAsyncFallbacks(t *testing.T) {
	for _, tt := range []struct {
		name string
		pool adapter.Pool
	}{
		{name: "nil pool", pool: nil},
		{name: "submit error", pool: managerSubmitErrorPool{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			manager := newTestManager(t, nil)
			manager.pool = tt.pool
			done := make(chan struct{})
			manager.submitAsync(tt.name, func() { close(done) })
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("fallback async task did not run")
			}
		})
	}
}

type managerErrorCodec struct{}

func (managerErrorCodec) Name() string { return "error" }

func (managerErrorCodec) Encode(any) ([]byte, error) { return nil, errors.New("encode failed") }

func (managerErrorCodec) Decode([]byte, any) error { return errors.New("decode failed") }

type managerStorageOnly struct {
	inner *managerTestStorage
}

func (s *managerStorageOnly) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return s.inner.Set(ctx, key, value, expiration)
}

func (s *managerStorageOnly) Get(ctx context.Context, key string) (any, error) {
	return s.inner.Get(ctx, key)
}

func (s *managerStorageOnly) Delete(ctx context.Context, keys ...string) error {
	return s.inner.Delete(ctx, keys...)
}

func (s *managerStorageOnly) Exists(ctx context.Context, key string) bool {
	return s.inner.Exists(ctx, key)
}

func (s *managerStorageOnly) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return s.inner.Expire(ctx, key, expiration)
}

func (s *managerStorageOnly) TTL(ctx context.Context, key string) (time.Duration, error) {
	return s.inner.TTL(ctx, key)
}

func (s *managerStorageOnly) Ping(ctx context.Context) error {
	return s.inner.Ping(ctx)
}

type managerScannerErrorStorage struct {
	*managerTestStorage
}

func (*managerScannerErrorStorage) Keys(context.Context, string) ([]string, error) {
	return nil, errors.New("scan failed")
}

type managerSubmitErrorPool struct{}

func (managerSubmitErrorPool) Submit(func()) error { return errors.New("submit failed") }

func (managerSubmitErrorPool) Stop() {}

func (managerSubmitErrorPool) Stats() (int, int, float64) { return 0, 0, 0 }

var _ adapter.Storage = (*managerStorageOnly)(nil)
var _ adapter.ScannerStorage = (*managerScannerErrorStorage)(nil)
