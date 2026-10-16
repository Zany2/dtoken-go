package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// TestOptionalFacadesReportDisabledModules verifies optional facades preserve disabled-module errors and zero values. TestOptionalFacadesReportDisabledModules 验证可选模块门面保留未启用错误和约定零值。
func TestOptionalFacadesReportDisabledModules(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("optional-disabled").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)
	auth := New(mgr)
	client := &oauth2.Client{ClientID: "disabled-client"}

	globalTicketStatus, globalTicketErr := GetTicketStatus(ctx, "ticket", "optional-disabled")
	instanceTicketStatus, instanceTicketErr := auth.GetTicketStatus(ctx, "ticket")
	globalShortKeyStatus, globalShortKeyErr := GetShortKeyStatus(ctx, "key", "optional-disabled")
	instanceShortKeyStatus, instanceShortKeyErr := auth.GetShortKeyStatus(ctx, "key")
	errChecks := map[string]error{
		"GenerateNonce":             func() error { _, err := GenerateNonce(ctx, "optional-disabled"); return err }(),
		"Auth.GenerateNonce":        func() error { _, err := auth.GenerateNonce(ctx); return err }(),
		"CreateTicket":              func() error { _, err := CreateTicket(ctx, "user", "optional-disabled"); return err }(),
		"Auth.CreateTicket":         func() error { _, err := auth.CreateTicket(ctx, "user"); return err }(),
		"GetTicketStatus":           globalTicketErr,
		"Auth.GetTicketStatus":      instanceTicketErr,
		"CreateShortKey":            func() error { _, err := CreateShortKey(ctx, "optional-disabled"); return err }(),
		"Auth.CreateShortKey":       func() error { _, err := auth.CreateShortKey(ctx); return err }(),
		"GetShortKeyStatus":         globalShortKeyErr,
		"Auth.GetShortKeyStatus":    instanceShortKeyErr,
		"RegisterOAuth2Client":      RegisterOAuth2Client(client, "optional-disabled"),
		"Auth.RegisterOAuth2Client": auth.RegisterOAuth2Client(client),
		"ValidateOAuth2AccessTokenInfo": func() error {
			_, err := ValidateOAuth2AccessTokenAndGetInfo(ctx, "token", "optional-disabled")
			return err
		}(),
		"Auth.ValidateOAuth2TokenInfo": func() error { _, err := auth.ValidateOAuth2AccessTokenAndGetInfo(ctx, "token"); return err }(),
	}
	for name, err := range errChecks {
		if !errors.Is(err, derror.ErrModuleNotEnabled) {
			t.Fatalf("%s error = %v, want ErrModuleNotEnabled", name, err)
		}
	}

	if globalTicketStatus != ticket.StatusInvalid || instanceTicketStatus != ticket.StatusInvalid {
		t.Fatalf("disabled ticket statuses = %q/%q, want invalid", globalTicketStatus, instanceTicketStatus)
	}
	if globalShortKeyStatus != shortkey.StatusInvalid || instanceShortKeyStatus != shortkey.StatusInvalid {
		t.Fatalf("disabled short key statuses = %q/%q, want invalid", globalShortKeyStatus, instanceShortKeyStatus)
	}
	if ValidateOAuth2AccessToken(ctx, "token", "optional-disabled") || auth.ValidateOAuth2AccessToken(ctx, "token") {
		t.Fatal("disabled OAuth2 validation should return false")
	}
}

// TestGlobalRefreshAndIntrospectionFacades verifies refresh-token and introspection global helpers. TestGlobalRefreshAndIntrospectionFacades 验证刷新令牌和令牌检查全局门面。
func TestGlobalRefreshAndIntrospectionFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("refresh-facade").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	pair, err := LoginWithRefreshToken(ctx, "refresh-user", "web", "browser", "refresh-facade")
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	if pair.LoginID != "refresh-user" || pair.Device != "web" || pair.DeviceID != "browser" {
		t.Fatalf("refresh pair = %+v, want login and device metadata", pair)
	}

	info, err := IntrospectToken(ctx, pair.AccessToken, "refresh-facade")
	if err != nil {
		t.Fatalf("IntrospectToken() error = %v", err)
	}
	if !info.Active || info.LoginID != "refresh-user" {
		t.Fatalf("introspection = %+v, want active refresh-user", info)
	}

	ttl, err := GetRefreshTokenTTL(ctx, pair.RefreshToken, "refresh-facade")
	if err != nil {
		t.Fatalf("GetRefreshTokenTTL() error = %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("refresh ttl = %d, want positive", ttl)
	}

	rotated, err := RefreshToken(ctx, pair.RefreshToken, "refresh-facade")
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if rotated.RefreshToken == pair.RefreshToken || rotated.AccessToken == pair.AccessToken {
		t.Fatal("RefreshToken() should rotate both refresh and access tokens")
	}
	if err = RevokeRefreshToken(ctx, rotated.RefreshToken, "refresh-facade"); err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}
	if IsLogin(ctx, rotated.AccessToken, "refresh-facade") {
		t.Fatal("access token should be logged out after refresh token revoke")
	}
}

// TestGlobalTicketAndShortKeyFacades verifies lifecycle helpers for optional credential modules. TestGlobalTicketAndShortKeyFacades 验证可选凭证模块的生命周期门面。
func TestGlobalTicketAndShortKeyFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("credential-facade").
		EnableTicket().
		EnableShortKey().
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	createdTicket, err := CreateTicketWithOptions(ctx, ticket.CreateOptions{
		LoginID:   "ticket-user",
		Device:    "web",
		DeviceID:  "browser",
		SourceApp: "issuer",
		TargetApp: "target",
		Timeout:   time.Minute,
	}, "credential-facade")
	if err != nil {
		t.Fatalf("CreateTicketWithOptions() error = %v", err)
	}
	if status, err := GetTicketStatus(ctx, createdTicket.Ticket, "credential-facade"); err != nil || status != ticket.StatusValid {
		t.Fatalf("GetTicketStatus() = %q, %v, want valid", status, err)
	}
	validTicket, err := ValidateTicketWithOptions(ctx, createdTicket.Ticket, ticket.ValidateOptions{
		LoginID: "ticket-user",
		Device:  "web",
	}, "credential-facade")
	if err != nil {
		t.Fatalf("ValidateTicketWithOptions() error = %v", err)
	}
	if validTicket.TargetApp != "target" {
		t.Fatalf("ticket target app = %q, want target", validTicket.TargetApp)
	}
	if ttl, err := GetTicketTTL(ctx, createdTicket.Ticket, "credential-facade"); err != nil || ttl <= 0 {
		t.Fatalf("GetTicketTTL() = %d, %v, want positive", ttl, err)
	}
	if _, err = ConsumeTicketWithOptions(ctx, createdTicket.Ticket, ticket.ValidateOptions{LoginID: "ticket-user"}, "credential-facade"); err != nil {
		t.Fatalf("ConsumeTicketWithOptions() error = %v", err)
	}
	if status, err := GetTicketStatus(ctx, createdTicket.Ticket, "credential-facade"); err != nil || status != ticket.StatusConsumed {
		t.Fatalf("GetTicketStatus(consumed) = %q, %v, want consumed", status, err)
	}

	createdKey, err := CreateShortKeyWithOptions(ctx, shortkey.CreateOptions{
		Scene:     "qr-login",
		SourceApp: "issuer",
		TargetApp: "target",
		Timeout:   time.Minute,
	}, "credential-facade")
	if err != nil {
		t.Fatalf("CreateShortKeyWithOptions() error = %v", err)
	}
	if status, err := GetShortKeyStatus(ctx, createdKey.Key, "credential-facade"); err != nil || status != shortkey.StatusPending {
		t.Fatalf("GetShortKeyStatus() = %q, %v, want pending", status, err)
	}
	confirmed, err := ConfirmShortKeyWithOptions(ctx, createdKey.Key, shortkey.ConfirmOptions{
		LoginID:  "short-user",
		Device:   "mobile",
		DeviceID: "phone",
	}, "credential-facade")
	if err != nil {
		t.Fatalf("ConfirmShortKeyWithOptions() error = %v", err)
	}
	if confirmed.Status != shortkey.StatusConfirmed || confirmed.LoginID != "short-user" {
		t.Fatalf("confirmed short key = %+v, want confirmed short-user", confirmed)
	}
	if _, err = ValidateShortKeyWithOptions(ctx, createdKey.Key, shortkey.ValidateOptions{
		LoginID: "short-user",
		Scene:   "qr-login",
	}, "credential-facade"); err != nil {
		t.Fatalf("ValidateShortKeyWithOptions() error = %v", err)
	}
	if ttl, err := GetShortKeyTTL(ctx, createdKey.Key, "credential-facade"); err != nil || ttl <= 0 {
		t.Fatalf("GetShortKeyTTL() = %d, %v, want positive", ttl, err)
	}
	if _, err = ConsumeShortKeyWithOptions(ctx, createdKey.Key, shortkey.ValidateOptions{LoginID: "short-user"}, "credential-facade"); err != nil {
		t.Fatalf("ConsumeShortKeyWithOptions() error = %v", err)
	}
	if status, err := GetShortKeyStatus(ctx, createdKey.Key, "credential-facade"); err != nil || status != shortkey.StatusConsumed {
		t.Fatalf("GetShortKeyStatus(consumed) = %q, %v, want consumed", status, err)
	}
}

// TestInstanceTerminalShortcutFacades verifies instance terminal action shortcuts. TestInstanceTerminalShortcutFacades 验证实例终端操作快捷门面。
func TestInstanceTerminalShortcutFacades(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("terminal-facade").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	token, err := auth.Login(ctx, LoginOptions{LoginID: "terminal-user", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("Auth.Login() error = %v", err)
	}
	if err = auth.KickoutByDeviceAndDeviceID(ctx, "terminal-user", "web", "browser"); err != nil {
		t.Fatalf("KickoutByDeviceAndDeviceID() error = %v", err)
	}
	if auth.IsLogin(ctx, token) {
		t.Fatal("token should not be logged in after kickout")
	}

	token, err = auth.Login(ctx, LoginOptions{LoginID: "terminal-user", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("Auth.Login(second) error = %v", err)
	}
	if err = auth.ReplaceByLoginID(ctx, "terminal-user"); err != nil {
		t.Fatalf("ReplaceByLoginID() error = %v", err)
	}
	if auth.IsLogin(ctx, token) {
		t.Fatal("token should not be logged in after replace")
	}

	token, err = auth.Login(ctx, LoginOptions{LoginID: "terminal-user", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("Auth.Login(third) error = %v", err)
	}
	if err = auth.Logout(ctx, LogoutOptions{Token: token}); err != nil {
		t.Fatalf("Logout(options) error = %v", err)
	}
	if auth.IsLogin(ctx, token) {
		t.Fatal("token should not be logged in after option logout")
	}
}
