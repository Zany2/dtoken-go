package manager

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestManagerAuthenticationQueryBoundaries verifies public authentication queries and invalid-input contracts. TestManagerAuthenticationQueryBoundaries 验证公开鉴权查询及无效输入契约。
func TestManagerAuthenticationQueryBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
		cfg.AutoRenew = false
		cfg.Timeout = 30
	})

	if _, err := mgr.Login(ctx, ""); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("Login(empty) error = %v, want ErrIDIsEmpty", err)
	}
	if _, err := mgr.GetSession(ctx, ""); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("GetSession(empty) error = %v, want ErrIDIsEmpty", err)
	}
	if _, err := mgr.GetSessionByToken(ctx, ""); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetSessionByToken(empty) error = %v, want ErrInvalidToken", err)
	}
	if _, err := mgr.GetTokenTTL(ctx, ""); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetTokenTTL(empty) error = %v, want ErrInvalidToken", err)
	}
	if _, err := mgr.GetLoginID(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetLoginID(missing) error = %v, want ErrInvalidToken", err)
	}
	if _, err := mgr.GetDevice(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetDevice(missing) error = %v, want ErrInvalidToken", err)
	}
	if _, err := mgr.GetDeviceID(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetDeviceID(missing) error = %v, want ErrInvalidToken", err)
	}
	if _, _, err := mgr.GetDeviceAndDeviceID(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetDeviceAndDeviceID(missing) error = %v, want ErrInvalidToken", err)
	}
	if _, err := mgr.GetTokenCreateTime(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetTokenCreateTime(missing) error = %v, want ErrInvalidToken", err)
	}
	if _, err := mgr.GetTerminalInfoByToken(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetTerminalInfoByToken(missing) error = %v, want ErrInvalidToken", err)
	}

	customToken, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:  "query-user",
		Device:   " web ",
		DeviceID: " browser-a ",
		Token:    " custom-token ",
	})
	if err != nil {
		t.Fatalf("LoginWithOptions(custom token) error = %v", err)
	}
	if customToken != "custom-token" {
		t.Fatalf("custom token = %q, want custom-token", customToken)
	}
	secondToken, err := mgr.Login(ctx, "query-user", "mobile", "phone-b")
	if err != nil {
		t.Fatalf("Login(second device) error = %v", err)
	}
	thirdToken, err := mgr.Login(ctx, "query-user", "web", "browser-b")
	if err != nil {
		t.Fatalf("Login(third device) error = %v", err)
	}

	info, err := mgr.GetTokenInfo(ctx, customToken)
	if err != nil {
		t.Fatalf("GetTokenInfo(custom token) error = %v", err)
	}
	if info.LoginID != "query-user" || info.Device != "web" || info.DeviceID != "browser-a" || info.Timeout != 30 {
		t.Fatalf("TokenInfo = %+v, want normalized login/device fields and timeout", info)
	}
	if got, err := mgr.GetLoginID(ctx, customToken); err != nil || got != "query-user" {
		t.Fatalf("GetLoginID(custom token) = %q, %v, want query-user", got, err)
	}

	allTokens, err := mgr.GetTokenValueListByLoginID(ctx, "query-user")
	if err != nil || !sameStrings(allTokens, []string{customToken, secondToken, thirdToken}) {
		t.Fatalf("GetTokenValueListByLoginID() = %v, %v, want all tokens", allTokens, err)
	}
	webTokens, err := mgr.GetTokenValueListByDevice(ctx, "query-user", " web ")
	if err != nil || !sameStrings(webTokens, []string{customToken, thirdToken}) {
		t.Fatalf("GetTokenValueListByDevice() = %v, %v, want web tokens", webTokens, err)
	}
	concreteTokens, err := mgr.GetTokenValueListByDeviceAndDeviceID(ctx, "query-user", "web", "browser-a", true)
	if err != nil || !sameStrings(concreteTokens, []string{customToken}) {
		t.Fatalf("GetTokenValueListByDeviceAndDeviceID() = %v, %v, want custom token", concreteTokens, err)
	}
	if got, err := mgr.GetTokenValueByLoginID(ctx, "query-user", "web"); err != nil || got != thirdToken {
		t.Fatalf("GetTokenValueByLoginID(web) = %q, %v, want newest web token %q", got, err, thirdToken)
	}
	if count, err := mgr.GetOnlineTerminalCount(ctx, "query-user"); err != nil || count != 3 {
		t.Fatalf("GetOnlineTerminalCount() = %d, %v, want 3", count, err)
	}
	if count, err := mgr.GetOnlineTerminalCountByDevice(ctx, "query-user", "mobile"); err != nil || count != 1 {
		t.Fatalf("GetOnlineTerminalCountByDevice() = %d, %v, want 1", count, err)
	}
	if count, err := mgr.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "query-user", "web", "browser-a"); err != nil || count != 1 {
		t.Fatalf("GetOnlineTerminalCountByDeviceAndDeviceID() = %d, %v, want 1", count, err)
	}

	terminals, err := mgr.GetTerminalListByLoginID(ctx, "query-user")
	if err != nil || len(terminals) != 3 {
		t.Fatalf("GetTerminalListByLoginID() = %v, %v, want 3 terminals", terminals, err)
	}
	terminals[0].Token = "mutated"
	latestTerminals, err := mgr.GetTerminalListByLoginID(ctx, "query-user", "web")
	if err != nil {
		t.Fatalf("GetTerminalListByLoginID(web) error = %v", err)
	}
	for _, terminal := range latestTerminals {
		if terminal.Token == "mutated" {
			t.Fatal("GetTerminalListByLoginID() returned mutable storage data")
		}
	}
	terminal, err := mgr.GetTerminalInfoByToken(ctx, customToken)
	if err != nil || terminal.DeviceID != "browser-a" {
		t.Fatalf("GetTerminalInfoByToken() = %+v, %v, want browser-a", terminal, err)
	}

	visited := make([]string, 0, 2)
	if err = mgr.ForEachTerminal(ctx, "query-user", func(terminal TerminalInfo) bool {
		visited = append(visited, terminal.Token)
		return len(visited) < 2
	}); err != nil {
		t.Fatalf("ForEachTerminal() error = %v", err)
	}
	if len(visited) != 2 {
		t.Fatalf("ForEachTerminal() visited %v, want exactly two terminals", visited)
	}
	if err = mgr.ForEachTerminalByDevice(ctx, "query-user", "mobile", func(TerminalInfo) bool { return true }); err != nil {
		t.Fatalf("ForEachTerminalByDevice() error = %v", err)
	}
	if err = mgr.ForEachTerminal(ctx, "missing", func(TerminalInfo) bool { return true }); err != nil {
		t.Fatalf("ForEachTerminal(missing) error = %v, want nil", err)
	}
	if err = mgr.ForEachTerminalByDevice(ctx, "query-user", " ", func(TerminalInfo) bool { return true }); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("ForEachTerminalByDevice(blank device) error = %v, want ErrInvalidParam", err)
	}
	if err = mgr.ForEachTerminalByDevice(ctx, "query-user", "mobile", nil); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("ForEachTerminalByDevice(nil visitor) error = %v, want ErrInvalidParam", err)
	}

	if err = mgr.Logout(ctx, customToken); err != nil {
		t.Fatalf("Logout(custom token) error = %v", err)
	}
	if err = mgr.Logout(ctx, customToken); err != nil {
		t.Fatalf("Logout(repeated) error = %v, want idempotent success", err)
	}
	if err = mgr.Kickout(ctx, customToken); err != nil {
		t.Fatalf("Kickout(already logged out) error = %v, want idempotent success", err)
	}
}

// TestManagerAuthenticationFailureAndRenewalBoundaries verifies renewal and storage/generator failure paths. TestManagerAuthenticationFailureAndRenewalBoundaries 验证续期及存储/生成器失败路径。
func TestManagerAuthenticationFailureAndRenewalBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 20
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "renew-boundary")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.RenewTimeout(ctx, token, 0); err != nil {
		t.Fatalf("RenewTimeout(default) error = %v", err)
	}
	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil || info.Timeout != 20 {
		t.Fatalf("GetTokenInfo(after default renew) = %+v, %v, want timeout 20", info, err)
	}
	if err = mgr.LoginByToken(ctx, token); err != nil {
		t.Fatalf("LoginByToken(active) error = %v", err)
	}
	if err = mgr.Kickout(ctx, token); err != nil {
		t.Fatalf("Kickout() error = %v", err)
	}
	if err = mgr.LoginByToken(ctx, token); !errors.Is(err, derror.ErrTokenKickout) {
		t.Fatalf("LoginByToken(kicked) error = %v, want ErrTokenKickout", err)
	}
	if err = mgr.RenewTimeout(ctx, token, time.Minute); !errors.Is(err, derror.ErrTokenKickout) {
		t.Fatalf("RenewTimeout(kicked) error = %v, want ErrTokenKickout", err)
	}

	activeManager := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 20
		cfg.ActiveTimeout = 30
		cfg.AutoRenew = false
	})
	activeToken, err := activeManager.Login(ctx, "missing-active-marker")
	if err != nil {
		t.Fatalf("Login(active marker) error = %v", err)
	}
	if err = activeManager.GetStorage().Delete(ctx, activeManager.getActiveKey(activeToken)); err != nil {
		t.Fatalf("Delete(active marker) error = %v", err)
	}
	if err = activeManager.CheckLogin(ctx, activeToken); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin(missing active marker) error = %v, want ErrInvalidToken", err)
	}

	generatorManager := newTestManager(t, nil)
	generatorManager.generator = managerErrorGenerator{}
	if _, err = generatorManager.Login(ctx, "generator-error"); !errors.Is(err, errManagerGenerator) {
		t.Fatalf("Login(generator error) = %v, want generator error", err)
	}

	storageError := errors.New("query storage unavailable")
	storageManager := newTestManager(t, nil)
	storageManager.storage = &managerFailingStorage{Storage: storageManager.storage, getErr: storageError, ttlErr: storageError}
	if _, err = storageManager.GetTokenInfo(ctx, "token"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("GetTokenInfo(storage error) = %v, want ErrStorageUnavailable", err)
	}
	if _, err = storageManager.GetTokenTTL(ctx, "token"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("GetTokenTTL(storage error) = %v, want ErrStorageUnavailable", err)
	}
}

// TestManagerDisableScopeContracts verifies account, service, and concrete-device disable semantics. TestManagerDisableScopeContracts 验证账号、服务及具体设备封禁语义。
func TestManagerDisableScopeContracts(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	if err := mgr.Disable(ctx, "permanent-account", 0, "manual"); err != nil {
		t.Fatalf("Disable(permanent) error = %v", err)
	}
	if !mgr.IsDisable(ctx, "permanent-account") {
		t.Fatal("IsDisable(permanent) = false, want true")
	}
	if err := mgr.CheckDisable(ctx, "permanent-account"); !errors.Is(err, derror.ErrAccountDisabled) {
		t.Fatalf("CheckDisable(permanent) = %v, want ErrAccountDisabled", err)
	}
	if ttl, err := mgr.GetDisableTTL(ctx, "permanent-account"); err != nil || ttl != -1 {
		t.Fatalf("GetDisableTTL(permanent) = %d, %v, want -1", ttl, err)
	}
	accountInfo, err := mgr.GetDisableInfo(ctx, "permanent-account")
	if err != nil || accountInfo.DisableReason != "manual" || accountInfo.DisableTime <= 0 {
		t.Fatalf("GetDisableInfo(permanent) = %+v, %v, want reason and timestamp", accountInfo, err)
	}
	if err = mgr.Untie(ctx, "permanent-account"); err != nil {
		t.Fatalf("Untie(permanent) error = %v", err)
	}
	if err = mgr.CheckDisable(ctx, "permanent-account"); err != nil {
		t.Fatalf("CheckDisable(after untie) = %v, want nil", err)
	}

	if err = mgr.DisableServiceLevel(ctx, "scope-user", " payments ", 2, 0, "risk"); err != nil {
		t.Fatalf("DisableServiceLevel() error = %v", err)
	}
	if !mgr.IsDisableService(ctx, "scope-user", "payments") || !mgr.IsDisableServiceLevel(ctx, "scope-user", "payments", 1) || !mgr.IsDisableServiceLevel(ctx, "scope-user", "payments", 2) {
		t.Fatal("service disable state did not match expected levels")
	}
	if mgr.IsDisableServiceLevel(ctx, "scope-user", "payments", 3) || mgr.IsDisableServiceLevel(ctx, "scope-user", "payments", -1) {
		t.Fatal("service disable level comparison returned unexpected result")
	}
	if err = mgr.CheckDisableService(ctx, "scope-user", "profile", "payments"); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("CheckDisableService() = %v, want ErrServiceDisabled", err)
	}
	if err = mgr.CheckDisableServiceLevel(ctx, "scope-user", "payments", 1); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("CheckDisableServiceLevel(blocked) = %v, want ErrServiceDisabled", err)
	}
	if err = mgr.CheckDisableServiceLevel(ctx, "scope-user", "payments", 3); err != nil {
		t.Fatalf("CheckDisableServiceLevel(above level) = %v, want nil", err)
	}
	serviceInfo, err := mgr.GetDisableServiceInfo(ctx, "scope-user", "payments")
	if err != nil || serviceInfo.Service != "payments" || serviceInfo.Level != 2 || serviceInfo.DisableReason != "risk" {
		t.Fatalf("GetDisableServiceInfo() = %+v, %v, want normalized service info", serviceInfo, err)
	}
	if ttl, err := mgr.GetDisableServiceTTL(ctx, "scope-user", "payments"); err != nil || ttl != -1 {
		t.Fatalf("GetDisableServiceTTL(permanent) = %d, %v, want -1", ttl, err)
	}
	if err = mgr.UntieService(ctx, "scope-user", " payments "); err != nil {
		t.Fatalf("UntieService() error = %v", err)
	}

	if err = mgr.DisableDevice(ctx, "scope-user", " web ", 0, "device-risk"); err != nil {
		t.Fatalf("DisableDevice(permanent) error = %v", err)
	}
	if !mgr.IsDisableDevice(ctx, "scope-user", "web") || !mgr.IsDisableDeviceAndDeviceID(ctx, "scope-user", "web", "browser") {
		t.Fatal("device type disable did not match concrete device checks")
	}
	if err = mgr.CheckDisableDeviceAndDeviceID(ctx, "scope-user", "web", "browser"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckDisableDeviceAndDeviceID(type disabled) = %v, want ErrDeviceDisabled", err)
	}
	if err = mgr.UntieDevice(ctx, "scope-user", "web"); err != nil {
		t.Fatalf("UntieDevice() error = %v", err)
	}
	if err = mgr.DisableDeviceAndDeviceID(ctx, "scope-user", " web ", " browser ", 0, "concrete-risk"); err != nil {
		t.Fatalf("DisableDeviceAndDeviceID(permanent) error = %v", err)
	}
	if !mgr.IsDisableDeviceAndDeviceID(ctx, "scope-user", "web", "browser") {
		t.Fatal("concrete device disable state = false, want true")
	}
	if ttl, err := mgr.GetDisableDeviceAndDeviceIDTTL(ctx, "scope-user", "web", "browser"); err != nil || ttl != -1 {
		t.Fatalf("GetDisableDeviceAndDeviceIDTTL(permanent) = %d, %v, want -1", ttl, err)
	}
	concreteInfo, err := mgr.GetDisableDeviceAndDeviceIDInfo(ctx, "scope-user", "web", "browser")
	if err != nil || concreteInfo.Device != "web" || concreteInfo.DeviceID != "browser" || concreteInfo.DisableReason != "concrete-risk" {
		t.Fatalf("GetDisableDeviceAndDeviceIDInfo() = %+v, %v, want normalized concrete info", concreteInfo, err)
	}
	if err = mgr.UntieDeviceAndDeviceID(ctx, "scope-user", "web", "browser"); err != nil {
		t.Fatalf("UntieDeviceAndDeviceID() error = %v", err)
	}
	if mgr.IsDisableDeviceAndDeviceID(ctx, "scope-user", "web", "browser") {
		t.Fatal("concrete device disable state = true after untie, want false")
	}
}

// TestManagerDisableInfoDecodeAndStorageErrors verifies disable metadata conversion and storage error propagation. TestManagerDisableInfoDecodeAndStorageErrors 验证封禁元数据转换及存储错误传播。
func TestManagerDisableInfoDecodeAndStorageErrors(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	storage := mgr.GetStorage()

	badAccountKey := mgr.getDisableKey("bad-account-type")
	badServiceKey := mgr.getDisableServiceKey("bad-service", "billing")
	badDeviceKey := mgr.getDisableDeviceKey("bad-device", "web")
	mgr.storage = &managerFailingStorage{Storage: storage, getValues: map[string]any{
		badAccountKey: 123,
		badServiceKey: []byte("not-json"),
		badDeviceKey:  []byte("not-json"),
	}}
	if _, err := mgr.GetDisableInfo(ctx, "bad-account-type"); !errors.Is(err, derror.ErrTypeConvert) {
		t.Fatalf("GetDisableInfo(type error) = %v, want ErrTypeConvert", err)
	}
	if _, err := mgr.GetDisableServiceInfo(ctx, "bad-service", "billing"); !errors.Is(err, derror.ErrSerializeFailed) {
		t.Fatalf("GetDisableServiceInfo(encoding error) = %v, want ErrSerializeFailed", err)
	}
	if _, err := mgr.GetDisableDeviceInfo(ctx, "bad-device", "web"); !errors.Is(err, derror.ErrSerializeFailed) {
		t.Fatalf("GetDisableDeviceInfo(encoding error) = %v, want ErrSerializeFailed", err)
	}

	storageErr := errors.New("disable storage unavailable")
	mgr.storage = &managerFailingStorage{Storage: storage, getErr: storageErr, ttlErr: storageErr}
	if _, err := mgr.GetDisableInfo(ctx, "error-account"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("GetDisableInfo(storage error) = %v, want ErrStorageUnavailable", err)
	}
	if _, err := mgr.GetDisableServiceInfo(ctx, "error-account", "service"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("GetDisableServiceInfo(storage error) = %v, want ErrStorageUnavailable", err)
	}
	if _, err := mgr.GetDisableDeviceInfo(ctx, "error-account", "web"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("GetDisableDeviceInfo(storage error) = %v, want ErrStorageUnavailable", err)
	}
	if _, err := mgr.GetDisableTTL(ctx, "error-account"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("GetDisableTTL(storage error) = %v, want ErrStorageUnavailable", err)
	}
}

// TestManagerAccessProviderFunctionContracts verifies provider fallbacks, normalization, and fail-closed resolution. TestManagerAccessProviderFunctionContracts 验证权限提供器回退、规范化及错误安全拒绝。
func TestManagerAccessProviderFunctionContracts(t *testing.T) {
	ctx := context.Background()
	var empty AccessProviderFunc
	if values, err := empty.Permissions(ctx, AccessSubject{}); err != nil || values != nil {
		t.Fatalf("empty AccessProviderFunc.Permissions() = %v, %v, want nil nil", values, err)
	}
	if values, err := empty.Roles(ctx, AccessSubject{}); err != nil || values != nil {
		t.Fatalf("empty AccessProviderFunc.Roles() = %v, %v, want nil nil", values, err)
	}
	if got := normalizeProviderAccessValues(nil); got != nil {
		t.Fatalf("normalizeProviderAccessValues(nil) = %#v, want nil", got)
	}
	if got := normalizeProviderAccessValues([]string{"", "read", "read"}); !reflect.DeepEqual(got, []string{"read"}) {
		t.Fatalf("normalizeProviderAccessValues() = %v, want [read]", got)
	}
	if got := normalizeProviderAccessValues([]string{}); got == nil || len(got) != 0 {
		t.Fatalf("normalizeProviderAccessValues(empty) = %#v, want non-nil empty", got)
	}

	var expectedAuthType string
	mgr := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{
		PermissionFunc: func(_ context.Context, subject AccessSubject) ([]string, error) {
			if expectedAuthType != "" && subject.AuthType != expectedAuthType {
				t.Errorf("permission subject AuthType = %q, want %q", subject.AuthType, expectedAuthType)
			}
			return []string{"", "provider:read", "provider:read"}, nil
		},
		RoleFunc: func(_ context.Context, subject AccessSubject) ([]string, error) {
			if expectedAuthType != "" && subject.AuthType != expectedAuthType {
				t.Errorf("role subject AuthType = %q, want %q", subject.AuthType, expectedAuthType)
			}
			return []string{"provider-role"}, nil
		},
	})
	expectedAuthType = mgr.config.AuthType
	fallback := []string{"session-value"}
	permissions, err := mgr.loadPermissions(ctx, fallback, AccessSubject{LoginID: "provider-user"})
	if err != nil || !reflect.DeepEqual(permissions, []string{"provider:read"}) {
		t.Fatalf("loadPermissions(provider) = %v, %v, want normalized provider values", permissions, err)
	}
	roles, err := mgr.loadRoles(ctx, fallback, AccessSubject{LoginID: "provider-user"})
	if err != nil || !reflect.DeepEqual(roles, []string{"provider-role"}) {
		t.Fatalf("loadRoles(provider) = %v, %v, want provider values", roles, err)
	}

	providerErr := errors.New("provider failure")
	mgr.accessProvider = AccessProviderFunc{
		PermissionFunc: func(context.Context, AccessSubject) ([]string, error) { return nil, providerErr },
		RoleFunc:       func(context.Context, AccessSubject) ([]string, error) { return nil, providerErr },
	}
	if _, err = mgr.loadPermissions(ctx, fallback, AccessSubject{}); !errors.Is(err, providerErr) {
		t.Fatalf("loadPermissions(error) = %v, want provider error", err)
	}
	if _, err = mgr.loadRoles(ctx, fallback, AccessSubject{}); !errors.Is(err, providerErr) {
		t.Fatalf("loadRoles(error) = %v, want provider error", err)
	}
	if got := mgr.resolvePermissions(ctx, fallback, AccessSubject{}); got == nil || len(got) != 0 {
		t.Fatalf("resolvePermissions(error) = %#v, want non-nil empty", got)
	}
	if got := mgr.resolveRoles(ctx, fallback, AccessSubject{}); got == nil || len(got) != 0 {
		t.Fatalf("resolveRoles(error) = %#v, want non-nil empty", got)
	}
}

// TestManagerIntrospectionFullContract verifies active introspection payloads and non-active errors. TestManagerIntrospectionFullContract 验证活跃自省载荷及非活跃错误。
func TestManagerIntrospectionFullContract(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithAccessProvider(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.ActiveTimeout = 30
		cfg.AutoRenew = false
	}, AccessProviderFunc{
		PermissionFunc: func(context.Context, AccessSubject) ([]string, error) { return []string{"provider:read"}, nil },
		RoleFunc:       func(context.Context, AccessSubject) ([]string, error) { return []string{"provider-role"}, nil },
	})
	token, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:  "introspection-full",
		Device:   " web ",
		DeviceID: " browser ",
		Extra:    map[string]any{"trace": "yes"},
	})
	if err != nil {
		t.Fatalf("LoginWithOptions() error = %v", err)
	}
	result, err := mgr.IntrospectToken(ctx, token)
	if err != nil {
		t.Fatalf("IntrospectToken(active) error = %v", err)
	}
	if !result.Active || result.AuthType != mgr.config.AuthType || result.LoginID != "introspection-full" || result.Device != "web" || result.DeviceID != "browser" || result.Timeout != 60 || result.ActiveTimeout != 30 || result.ExpiresIn <= 0 {
		t.Fatalf("active introspection = %+v, want complete token payload", result)
	}
	if !reflect.DeepEqual(result.Permissions, []string{"provider:read"}) || !reflect.DeepEqual(result.Roles, []string{"provider-role"}) || result.Extra["trace"] != "yes" {
		t.Fatalf("active introspection access/extra = permissions:%v roles:%v extra:%v", result.Permissions, result.Roles, result.Extra)
	}

	if err = mgr.Disable(ctx, "introspection-full", time.Minute, "blocked"); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	result, err = mgr.IntrospectToken(ctx, token)
	if err != nil || result.Active || !strings.Contains(result.Error, "account disabled") {
		t.Fatalf("IntrospectToken(disabled) = %+v, %v, want inactive account-disabled result", result, err)
	}

	providerErr := errors.New("introspection provider failure")
	errorManager := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{
		PermissionFunc: func(context.Context, AccessSubject) ([]string, error) { return nil, providerErr },
		RoleFunc:       func(context.Context, AccessSubject) ([]string, error) { return nil, providerErr },
	})
	errorToken, err := errorManager.Login(ctx, "introspection-provider-error")
	if err != nil {
		t.Fatalf("Login(provider error manager) error = %v", err)
	}
	if _, err = errorManager.IntrospectToken(ctx, errorToken); !errors.Is(err, providerErr) {
		t.Fatalf("IntrospectToken(provider error) = %v, want provider error", err)
	}
}

// TestManagerTerminateRoutingAndValidation verifies token precedence, normalization, and scope validation. TestManagerTerminateRoutingAndValidation 验证 Token 优先级、字段规范化及作用域校验。
func TestManagerTerminateRoutingAndValidation(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})
	target, err := mgr.Login(ctx, "terminate-boundary", "web")
	if err != nil {
		t.Fatalf("Login(target) error = %v", err)
	}
	kept, err := mgr.Login(ctx, "terminate-boundary", "mobile")
	if err != nil {
		t.Fatalf("Login(kept) error = %v", err)
	}
	if err = mgr.Terminate(ctx, TerminateOptions{
		Token:   " " + target + " ",
		LoginID: "ignored-login-id",
		Device:  "missing-device",
		Action:  TerminateActionLogout,
	}); err != nil {
		t.Fatalf("Terminate(token precedence) error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, target); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin(target) = %v, want ErrInvalidToken", err)
	}
	if err = mgr.CheckLogin(ctx, kept); err != nil {
		t.Fatalf("CheckLogin(kept) = %v, want nil", err)
	}

	if err = mgr.Terminate(ctx, TerminateOptions{LoginID: "terminate-default"}); err != nil {
		t.Fatalf("Terminate(missing session default) error = %v, want nil", err)
	}
	if err = mgr.Terminate(ctx, TerminateOptions{}); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("Terminate(empty) = %v, want ErrIDIsEmpty", err)
	}
	if err = mgr.Terminate(ctx, TerminateOptions{LoginID: "u", DeviceID: "id"}); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("Terminate(device id without device) = %v, want ErrInvalidParam", err)
	}
	if err = mgr.Terminate(ctx, TerminateOptions{LoginID: "u", Action: TerminateAction("invalid")}); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("Terminate(invalid action) = %v, want ErrInvalidParam", err)
	}
	if err = mgr.Terminate(ctx, TerminateOptions{Token: "token", Action: TerminateAction("invalid")}); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("Terminate(token invalid action) = %v, want ErrInvalidParam", err)
	}
}

// TestManagerRefreshRevokeWrapsStorageErrors verifies refresh-token revocation exposes the manager storage error contract. TestManagerRefreshRevokeWrapsStorageErrors 验证刷新令牌撤销遵循 Manager 存储错误契约。
func TestManagerRefreshRevokeWrapsStorageErrors(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	refreshToken := "revoke-storage-error"
	if err := mgr.saveToStorage(ctx, mgr.getRefreshTokenKey(refreshToken), RefreshTokenInfo{}, time.Minute); err != nil {
		t.Fatalf("save refresh metadata error = %v", err)
	}
	storageErr := errors.New("revoke delete failed")
	mgr.storage = &managerFailingStorage{Storage: mgr.storage, deleteErr: storageErr}
	if err := mgr.RevokeRefreshToken(ctx, refreshToken); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("RevokeRefreshToken(storage error) = %v, want ErrStorageUnavailable", err)
	}
}

var errManagerGenerator = errors.New("generator failure")

type managerErrorGenerator struct{}

func (managerErrorGenerator) Generate(string, string, string) (string, error) {
	return "", errManagerGenerator
}

type managerFailingStorage struct {
	adapter.Storage
	getErr    error
	ttlErr    error
	deleteErr error
	getValues map[string]any
}

func (s *managerFailingStorage) Get(ctx context.Context, key string) (any, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if value, ok := s.getValues[key]; ok {
		return value, nil
	}
	return s.Storage.Get(ctx, key)
}

func (s *managerFailingStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if s.ttlErr != nil {
		return 0, s.ttlErr
	}
	return s.Storage.TTL(ctx, key)
}

func (s *managerFailingStorage) Delete(ctx context.Context, keys ...string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.Storage.Delete(ctx, keys...)
}

// Close forwards cleanup to wrapped storage when it exposes a close operation. Close 在被包装存储支持时转发清理操作。
func (s *managerFailingStorage) Close() error {
	if closer, ok := s.Storage.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

var _ adapter.Generator = managerErrorGenerator{}
var _ adapter.Storage = (*managerFailingStorage)(nil)
