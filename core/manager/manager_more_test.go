package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

func TestManagerAccessors(t *testing.T) {
	mgr := newTestManager(t, nil)

	if mgr.GetConfig() == nil {
		t.Fatal("GetConfig() = nil")
	}
	if mgr.GetGenerator() == nil {
		t.Fatal("GetGenerator() = nil")
	}
	if mgr.GetStorage() == nil {
		t.Fatal("GetStorage() = nil")
	}
	if mgr.GetSerializer() == nil {
		t.Fatal("GetSerializer() = nil")
	}
	if mgr.GetLogger() == nil {
		t.Fatal("GetLogger() = nil")
	}
	if mgr.GetPool() != nil {
		t.Fatalf("GetPool() = %T, want nil", mgr.GetPool())
	}
	if mgr.GetAccessProvider() != nil {
		t.Fatalf("GetAccessProvider() = %T, want nil", mgr.GetAccessProvider())
	}
	if mgr.GetEventManager() == nil {
		t.Fatal("GetEventManager() = nil")
	}
	if mgr.GetNonceManager() != nil {
		t.Fatalf("GetNonceManager() = %T, want nil before enabling module", mgr.GetNonceManager())
	}
	if mgr.GetOAuth2Manager() != nil {
		t.Fatalf("GetOAuth2Manager() = %T, want nil before enabling module", mgr.GetOAuth2Manager())
	}
	if mgr.GetTicketManager() != nil {
		t.Fatalf("GetTicketManager() = %T, want nil before enabling module", mgr.GetTicketManager())
	}
	if mgr.GetShortKeyManager() != nil {
		t.Fatalf("GetShortKeyManager() = %T, want nil before enabling module", mgr.GetShortKeyManager())
	}
}

func TestManagerLoginRenewAndTokenDetails(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 30
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "detail-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	device, err := mgr.GetDevice(ctx, token)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if device != "web" {
		t.Fatalf("GetDevice() = %q, want web", device)
	}
	deviceID, err := mgr.GetDeviceId(ctx, token)
	if err != nil {
		t.Fatalf("GetDeviceId() error = %v", err)
	}
	if deviceID != "browser-1" {
		t.Fatalf("GetDeviceId() = %q, want browser-1", deviceID)
	}
	combinedDevice, combinedDeviceID, err := mgr.GetDeviceAndDeviceId(ctx, token)
	if err != nil {
		t.Fatalf("GetDeviceAndDeviceId() error = %v", err)
	}
	if combinedDevice != "web" || combinedDeviceID != "browser-1" {
		t.Fatalf("GetDeviceAndDeviceId() = %q/%q, want web/browser-1", combinedDevice, combinedDeviceID)
	}
	createdAt, err := mgr.GetTokenCreateTime(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenCreateTime() error = %v", err)
	}
	if createdAt <= 0 {
		t.Fatalf("GetTokenCreateTime() = %d, want positive unix time", createdAt)
	}

	if err = mgr.RenewTimeout(ctx, token, 75*time.Second); err != nil {
		t.Fatalf("RenewTimeout() error = %v", err)
	}
	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenInfo() error = %v", err)
	}
	if info.Timeout != 75 {
		t.Fatalf("TokenInfo.Timeout = %d, want 75", info.Timeout)
	}
	ttl, err := mgr.GetTokenTTL(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 75 {
		t.Fatalf("GetTokenTTL() = %d, want 1..75", ttl)
	}

	if err = mgr.LoginByToken(ctx, token); err != nil {
		t.Fatalf("LoginByToken() error = %v", err)
	}
	if err = mgr.LoginByToken(ctx, ""); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("LoginByToken(empty) error = %v, want ErrInvalidToken", err)
	}
	if err = mgr.RenewTimeout(ctx, "", time.Second); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("RenewTimeout(empty) error = %v, want ErrInvalidToken", err)
	}
}

func TestManagerLogoutKickoutReplaceScopes(t *testing.T) {
	ctx := context.Background()

	t.Run("logout by device removes only that device", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		web, err := mgr.Login(ctx, "scope-logout", "web", "a")
		if err != nil {
			t.Fatalf("Login(web) error = %v", err)
		}
		mobile, err := mgr.Login(ctx, "scope-logout", "mobile", "a")
		if err != nil {
			t.Fatalf("Login(mobile) error = %v", err)
		}
		if err = mgr.LogoutByDevice(ctx, "scope-logout", "web"); err != nil {
			t.Fatalf("LogoutByDevice() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, web); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("web CheckLogin() error = %v, want ErrInvalidToken", err)
		}
		if err = mgr.CheckLogin(ctx, mobile); err != nil {
			t.Fatalf("mobile CheckLogin() error = %v", err)
		}
	})

	t.Run("kickout and replace by login id preserve causes", func(t *testing.T) {
		kickMgr := newTestManager(t, nil)
		kickToken, err := kickMgr.Login(ctx, "scope-kick")
		if err != nil {
			t.Fatalf("Login(kick) error = %v", err)
		}
		if err = kickMgr.KickoutByLoginID(ctx, "scope-kick"); err != nil {
			t.Fatalf("KickoutByLoginID() error = %v", err)
		}
		if err = kickMgr.CheckLogin(ctx, kickToken); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("kick token CheckLogin() error = %v, want ErrTokenKickout", err)
		}

		replaceMgr := newTestManager(t, nil)
		replaceToken, err := replaceMgr.Login(ctx, "scope-replace")
		if err != nil {
			t.Fatalf("Login(replace) error = %v", err)
		}
		if err = replaceMgr.ReplaceByLoginID(ctx, "scope-replace"); err != nil {
			t.Fatalf("ReplaceByLoginID() error = %v", err)
		}
		if err = replaceMgr.CheckLogin(ctx, replaceToken); !errors.Is(err, derror.ErrTokenReplaced) {
			t.Fatalf("replace token CheckLogin() error = %v, want ErrTokenReplaced", err)
		}
	})

	t.Run("invalid scoped arguments", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		if err := mgr.LogoutByDevice(ctx, "", "web"); !errors.Is(err, derror.ErrIDIsEmpty) {
			t.Fatalf("LogoutByDevice(empty id) error = %v, want ErrIDIsEmpty", err)
		}
		if err := mgr.KickoutByDevice(ctx, "u", " "); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("KickoutByDevice(empty device) error = %v, want ErrInvalidParam", err)
		}
		if err := mgr.ReplaceByDeviceAndDeviceId(ctx, "u", "web"); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("ReplaceByDeviceAndDeviceId(missing id) error = %v, want ErrInvalidParam", err)
		}
	})
}

func TestManagerDisableBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	if err := mgr.Disable(ctx, "", time.Minute); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("Disable(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if err := mgr.Untie(ctx, ""); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("Untie(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if mgr.IsDisable(ctx, "") {
		t.Fatal("IsDisable(empty id) = true, want false")
	}
	if _, err := mgr.GetDisableInfo(ctx, "missing"); !errors.Is(err, derror.ErrAccountNotDisabled) {
		t.Fatalf("GetDisableInfo(missing) error = %v, want ErrAccountNotDisabled", err)
	}
	if ttl, err := mgr.GetDisableTTL(ctx, "missing"); err != nil || ttl != -2 {
		t.Fatalf("GetDisableTTL(missing) = %d, %v, want -2, nil", ttl, err)
	}
	if err := mgr.CheckDisable(ctx, "missing"); err != nil {
		t.Fatalf("CheckDisable(missing) error = %v", err)
	}

	if err := mgr.DisableDevice(ctx, "device-user", " web ", time.Minute, "risk"); err != nil {
		t.Fatalf("DisableDevice() error = %v", err)
	}
	if !mgr.IsDisableDevice(ctx, "device-user", "web") {
		t.Fatal("IsDisableDevice() = false, want true")
	}
	if err := mgr.CheckDisableDevice(ctx, "device-user", "web"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckDisableDevice() error = %v, want ErrDeviceDisabled", err)
	}
	info, err := mgr.GetDisableDeviceInfo(ctx, "device-user", "web")
	if err != nil {
		t.Fatalf("GetDisableDeviceInfo() error = %v", err)
	}
	if info.Device != "web" || info.DisableReason != "risk" {
		t.Fatalf("device disable info = %+v, want web risk", info)
	}
	ttl, err := mgr.GetDisableDeviceTTL(ctx, "device-user", "web")
	if err != nil {
		t.Fatalf("GetDisableDeviceTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("GetDisableDeviceTTL() = %d, want 1..60", ttl)
	}
	if err = mgr.UntieDevice(ctx, "device-user", "web"); err != nil {
		t.Fatalf("UntieDevice() error = %v", err)
	}
	if _, err = mgr.GetDisableDeviceInfo(ctx, "device-user", "web"); !errors.Is(err, derror.ErrDeviceNotDisabled) {
		t.Fatalf("GetDisableDeviceInfo(after untie) error = %v, want ErrDeviceNotDisabled", err)
	}

	if err = mgr.DisableDeviceAndDeviceId(ctx, "device-user", "web", "a", time.Minute); err != nil {
		t.Fatalf("DisableDeviceAndDeviceId() error = %v", err)
	}
	if !mgr.IsDisableDeviceAndDeviceId(ctx, "device-user", "web", "a") {
		t.Fatal("IsDisableDeviceAndDeviceId() = false, want true")
	}
	if err = mgr.CheckDisableDeviceAndDeviceId(ctx, "device-user", "web", "a"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckDisableDeviceAndDeviceId() error = %v, want ErrDeviceDisabled", err)
	}
	concreteInfo, err := mgr.GetDisableDeviceAndDeviceIdInfo(ctx, "device-user", "web", "a")
	if err != nil {
		t.Fatalf("GetDisableDeviceAndDeviceIdInfo() error = %v", err)
	}
	if concreteInfo.Device != "web" || concreteInfo.DeviceId != "a" {
		t.Fatalf("concrete device disable info = %+v, want web/a", concreteInfo)
	}
	if err = mgr.UntieDeviceAndDeviceId(ctx, "device-user", "web", "a"); err != nil {
		t.Fatalf("UntieDeviceAndDeviceId() error = %v", err)
	}
	if ttl, err = mgr.GetDisableDeviceAndDeviceIdTTL(ctx, "device-user", "web", "a"); err != nil || ttl != -2 {
		t.Fatalf("GetDisableDeviceAndDeviceIdTTL(after untie) = %d, %v, want -2, nil", ttl, err)
	}

	for name, testErr := range map[string]error{
		"DisableService empty service":          mgr.DisableService(ctx, "u", " ", time.Minute),
		"DisableServiceLevel empty service":     mgr.DisableServiceLevel(ctx, "u", " ", 1, time.Minute),
		"UntieService empty service":            mgr.UntieService(ctx, "u", " "),
		"CheckDisableService empty service":     mgr.CheckDisableService(ctx, "u", " "),
		"CheckDisableDevice empty device":       mgr.CheckDisableDevice(ctx, "u", " "),
		"CheckDisableConcrete empty device id":  mgr.CheckDisableDeviceAndDeviceId(ctx, "u", "web", " "),
		"GetDisableServiceInfo empty service":   func() error { _, err := mgr.GetDisableServiceInfo(ctx, "u", " "); return err }(),
		"GetDisableDeviceInfo empty device":     func() error { _, err := mgr.GetDisableDeviceInfo(ctx, "u", " "); return err }(),
		"GetDisableConcreteInfo empty device":   func() error { _, err := mgr.GetDisableDeviceAndDeviceIdInfo(ctx, "u", " ", "a"); return err }(),
		"GetDisableServiceTTL empty service":    func() error { _, err := mgr.GetDisableServiceTTL(ctx, "u", " "); return err }(),
		"GetDisableDeviceTTL empty device":      func() error { _, err := mgr.GetDisableDeviceTTL(ctx, "u", " "); return err }(),
		"GetDisableConcreteTTL empty device id": func() error { _, err := mgr.GetDisableDeviceAndDeviceIdTTL(ctx, "u", "web", " "); return err }(),
	} {
		if !errors.Is(testErr, derror.ErrInvalidParam) {
			t.Fatalf("%s error = %v, want ErrInvalidParam", name, testErr)
		}
	}
}

func TestManagerOptionalModulesDisabled(t *testing.T) {
	ctx := context.Background()
	mgr := newBareTestManager(t)

	errChecks := map[string]error{
		"GenerateNonce":            func() error { _, err := mgr.GenerateNonce(ctx); return err }(),
		"GenerateNonceWithTimeout": func() error { _, err := mgr.GenerateNonceWithTimeout(ctx, time.Minute); return err }(),
		"VerifyAndConsumeNonce":    mgr.VerifyAndConsumeNonce(ctx, "nonce"),
		"GetNonceTTL":              func() error { _, err := mgr.GetNonceTTL(ctx, "nonce"); return err }(),
		"CreateTicket":             func() error { _, err := mgr.CreateTicket(ctx, ticket.CreateOptions{}); return err }(),
		"CreateTicketWithTimeout": func() error {
			_, err := mgr.CreateTicketWithTimeout(ctx, ticket.CreateOptions{}, time.Minute)
			return err
		}(),
		"ValidateTicket":  func() error { _, err := mgr.ValidateTicket(ctx, "ticket"); return err }(),
		"ConsumeTicket":   func() error { _, err := mgr.ConsumeTicket(ctx, "ticket"); return err }(),
		"RevokeTicket":    mgr.RevokeTicket(ctx, "ticket"),
		"GetTicketStatus": func() error { _, err := mgr.GetTicketStatus(ctx, "ticket"); return err }(),
		"GetTicketTTL":    func() error { _, err := mgr.GetTicketTTL(ctx, "ticket"); return err }(),
		"CreateShortKey":  func() error { _, err := mgr.CreateShortKey(ctx, shortkey.CreateOptions{}); return err }(),
		"CreateShortKeyWithTimeout": func() error {
			_, err := mgr.CreateShortKeyWithTimeout(ctx, shortkey.CreateOptions{}, time.Minute)
			return err
		}(),
		"ConfirmShortKey":        func() error { _, err := mgr.ConfirmShortKey(ctx, "key", shortkey.ConfirmOptions{}); return err }(),
		"ValidateShortKey":       func() error { _, err := mgr.ValidateShortKey(ctx, "key"); return err }(),
		"ConsumeShortKey":        func() error { _, err := mgr.ConsumeShortKey(ctx, "key"); return err }(),
		"RevokeShortKey":         mgr.RevokeShortKey(ctx, "key"),
		"GetShortKeyStatus":      func() error { _, err := mgr.GetShortKeyStatus(ctx, "key"); return err }(),
		"GetShortKeyTTL":         func() error { _, err := mgr.GetShortKeyTTL(ctx, "key"); return err }(),
		"RegisterOAuth2Client":   mgr.RegisterOAuth2Client(&oauth2.Client{ClientID: "client"}),
		"UnregisterOAuth2Client": mgr.UnregisterOAuth2Client("client"),
		"GetOAuth2Client":        func() error { _, err := mgr.GetOAuth2Client("client"); return err }(),
		"OAuth2Token":            func() error { _, err := mgr.OAuth2Token(ctx, &oauth2.TokenRequest{}, nil); return err }(),
		"GenerateOAuth2AuthorizationCode": func() error {
			_, err := mgr.GenerateOAuth2AuthorizationCode(ctx, "client", "user", "uri", nil)
			return err
		}(),
		"GenerateOAuth2AuthorizationCodeWithPKCE": func() error {
			_, err := mgr.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, "client", "user", "uri", nil, "challenge", "plain")
			return err
		}(),
		"ExchangeOAuth2CodeForToken": func() error {
			_, err := mgr.ExchangeOAuth2CodeForToken(ctx, "code", "client", "secret", "uri")
			return err
		}(),
		"ExchangeOAuth2CodeForTokenWithPKCE": func() error {
			_, err := mgr.ExchangeOAuth2CodeForTokenWithPKCE(ctx, "code", "client", "secret", "uri", "verifier")
			return err
		}(),
		"OAuth2ClientCredentialsToken": func() error { _, err := mgr.OAuth2ClientCredentialsToken(ctx, "client", "secret", nil); return err }(),
		"OAuth2PasswordGrantToken": func() error {
			_, err := mgr.OAuth2PasswordGrantToken(ctx, "client", "secret", "user", "pass", nil, nil)
			return err
		}(),
		"RefreshOAuth2AccessToken":            func() error { _, err := mgr.RefreshOAuth2AccessToken(ctx, "client", "refresh", "secret"); return err }(),
		"ValidateOAuth2AccessTokenAndGetInfo": func() error { _, err := mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, "access"); return err }(),
		"RevokeOAuth2Token":                   mgr.RevokeOAuth2Token(ctx, "access"),
	}
	for name, err := range errChecks {
		if !errors.Is(err, derror.ErrModuleNotEnabled) {
			t.Fatalf("%s error = %v, want ErrModuleNotEnabled", name, err)
		}
	}

	falseChecks := map[string]bool{
		"VerifyNonce":               mgr.VerifyNonce(ctx, "nonce"),
		"IsNonceValid":              mgr.IsNonceValid(ctx, "nonce"),
		"ValidateOAuth2AccessToken": mgr.ValidateOAuth2AccessToken(ctx, "access"),
	}
	for name, got := range falseChecks {
		if got {
			t.Fatalf("%s = true, want false when module disabled", name)
		}
	}
}

func TestManagerNonceFacade(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithNonce(t)

	value, err := mgr.GenerateNonce(ctx)
	if err != nil {
		t.Fatalf("GenerateNonce() error = %v", err)
	}
	if value == "" {
		t.Fatal("GenerateNonce() returned empty nonce")
	}
	if !mgr.IsNonceValid(ctx, value) {
		t.Fatal("IsNonceValid() = false, want true")
	}
	ttl, err := mgr.GetNonceTTL(ctx, value)
	if err != nil {
		t.Fatalf("GetNonceTTL() error = %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("GetNonceTTL() = %d, want positive", ttl)
	}
	if !mgr.VerifyNonce(ctx, value) {
		t.Fatal("VerifyNonce() = false, want true")
	}
	if mgr.IsNonceValid(ctx, value) {
		t.Fatal("IsNonceValid() after verify = true, want false")
	}

	custom, err := mgr.GenerateNonceWithTimeout(ctx, time.Minute)
	if err != nil {
		t.Fatalf("GenerateNonceWithTimeout() error = %v", err)
	}
	if err = mgr.VerifyAndConsumeNonce(ctx, custom); err != nil {
		t.Fatalf("VerifyAndConsumeNonce() error = %v", err)
	}
}

func newTestManagerWithNonce(t *testing.T) *Manager {
	t.Helper()

	mgr := newTestManager(t, nil)
	WithNonceManager(nonce.NewDefaultNonceManager(
		mgr.GetConfig().AuthType,
		mgr.GetConfig().KeyPrefix,
		mgr.GetStorage(),
	))(mgr)
	return mgr
}

func newBareTestManager(t *testing.T) *Manager {
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
	)
	mgr.nonceManager = nil
	mgr.oauth2Manager = nil
	mgr.ticketManager = nil
	mgr.shortKeyManager = nil
	t.Cleanup(mgr.CloseManager)
	return mgr
}
