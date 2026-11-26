package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestManagerOAuth2FacadeFlowAndEvents verifies OAuth2 facade operations and emitted events. TestManagerOAuth2FacadeFlowAndEvents 验证 OAuth2 门面操作与事件触发。

// TestManagerOAuth2TokenEndpointRefreshEvent verifies refresh grants emit the OAuth2 refresh event. TestManagerOAuth2TokenEndpointRefreshEvent 验证刷新授权会触发 OAuth2 刷新事件。

// TestManagerNonceEvents verifies nonce generation and verification events. TestManagerNonceEvents 验证 Nonce 生成与校验事件。

// TestManagerAccessProviderSubjectAndEmptyOverride verifies provider subjects and explicit empty access overrides. TestManagerAccessProviderSubjectAndEmptyOverride 验证访问提供器主体与显式空权限覆盖。
func TestManagerAccessProviderSubjectAndEmptyOverride(t *testing.T) {
	ctx := context.Background()
	var permissionSubjects []AccessSubject
	var roleSubjects []AccessSubject

	provider := AccessProviderFunc{
		PermissionFunc: func(ctx context.Context, subject AccessSubject) ([]string, error) {
			permissionSubjects = append(permissionSubjects, subject)
			return []string{}, nil
		},
		RoleFunc: func(ctx context.Context, subject AccessSubject) ([]string, error) {
			roleSubjects = append(roleSubjects, subject)
			return []string{}, nil
		},
	}
	mgr := newTestManagerWithAccessProvider(t, func(cfg *config.Config) {
		cfg.AuthType = "provider-auth"
	}, provider)

	token, err := mgr.Login(ctx, "provider-user", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissionsByToken(ctx, token, []string{"session:read"}); err != nil {
		t.Fatalf("AddPermissionsByToken() error = %v", err)
	}
	if err = mgr.AddRolesByToken(ctx, token, []string{"session-role"}); err != nil {
		t.Fatalf("AddRolesByToken() error = %v", err)
	}

	if mgr.HasPermission(ctx, "provider-user", "session:read") {
		t.Fatal("HasPermission() = true, want provider empty slice to override session permissions")
	}
	if mgr.HasPermissionByToken(ctx, token, "session:read") {
		t.Fatal("HasPermissionByToken() = true, want provider empty slice to override session permissions")
	}
	if mgr.HasRole(ctx, "provider-user", "session-role") {
		t.Fatal("HasRole() = true, want provider empty slice to override session roles")
	}
	if mgr.HasRoleByToken(ctx, token, "session-role") {
		t.Fatal("HasRoleByToken() = true, want provider empty slice to override session roles")
	}

	wantAccountSubject := AccessSubject{AuthType: "provider-auth:", LoginID: "provider-user"}
	wantTokenSubject := AccessSubject{
		AuthType: "provider-auth:",
		LoginID:  "provider-user",
		Device:   "web",
		DeviceID: "browser",
		Token:    token,
	}
	if !containsAccessSubject(permissionSubjects, wantAccountSubject) {
		t.Fatalf("permission subjects = %+v, want account subject %+v", permissionSubjects, wantAccountSubject)
	}
	if !containsAccessSubject(permissionSubjects, wantTokenSubject) {
		t.Fatalf("permission subjects = %+v, want token subject %+v", permissionSubjects, wantTokenSubject)
	}
	if !containsAccessSubject(roleSubjects, wantAccountSubject) {
		t.Fatalf("role subjects = %+v, want account subject %+v", roleSubjects, wantAccountSubject)
	}
	if !containsAccessSubject(roleSubjects, wantTokenSubject) {
		t.Fatalf("role subjects = %+v, want token subject %+v", roleSubjects, wantTokenSubject)
	}
}

// TestManagerSessionDataBoundaries verifies session data validation and missing-value boundaries. TestManagerSessionDataBoundaries 验证 Session 数据校验与缺失值边界。
func TestManagerSessionDataBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	if err := mgr.SetSessionValue(ctx, "missing-session", "theme", "dark"); !errors.Is(err, derror.ErrSessionNotFound) {
		t.Fatalf("SetSessionValue(missing session) error = %v, want ErrSessionNotFound", err)
	}
	if _, _, err := mgr.GetSessionValue(ctx, "missing-session", "theme"); !errors.Is(err, derror.ErrSessionNotFound) {
		t.Fatalf("GetSessionValue(missing session) error = %v, want ErrSessionNotFound", err)
	}
	if err := mgr.DeleteSessionValue(ctx, "missing-session", "theme"); !errors.Is(err, derror.ErrSessionNotFound) {
		t.Fatalf("DeleteSessionValue(missing session) error = %v, want ErrSessionNotFound", err)
	}

	if _, err := mgr.Login(ctx, "session-boundary"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := mgr.DeleteSessionValue(ctx, "session-boundary", "missing-key"); err != nil {
		t.Fatalf("DeleteSessionValue(missing key) error = %v", err)
	}
	if err := mgr.DeleteSessionValue(ctx, "", "theme"); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("DeleteSessionValue(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if err := mgr.DeleteSessionValue(ctx, "session-boundary", ""); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("DeleteSessionValue(empty key) error = %v, want ErrInvalidParam", err)
	}
	if _, _, err := mgr.GetSessionValue(ctx, "", "theme"); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("GetSessionValue(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if err := mgr.SetSessionValue(ctx, "session-boundary", "", "dark"); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("SetSessionValue(empty key) error = %v, want ErrInvalidParam", err)
	}
}

// TestManagerSessionDataByToken validates token-based session data access and logout protection. TestManagerSessionDataByToken 验证基于 Token 的会话数据访问及登出保护。
func TestManagerSessionDataByToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "session-token-user", "web")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.SetSessionValueByToken(ctx, token, " theme ", "dark"); err != nil {
		t.Fatalf("SetSessionValueByToken() error = %v", err)
	}
	if value, ok, err := mgr.GetSessionValueByToken(ctx, token, "theme"); err != nil || !ok || value != "dark" {
		t.Fatalf("GetSessionValueByToken() = %#v/%v, %v", value, ok, err)
	}
	if err = mgr.DeleteSessionValueByToken(ctx, token, "theme"); err != nil {
		t.Fatalf("DeleteSessionValueByToken() error = %v", err)
	}
	if _, ok, err := mgr.GetSessionValueByToken(ctx, token, "theme"); err != nil || ok {
		t.Fatalf("GetSessionValueByToken(after delete) = ok:%v, err:%v", ok, err)
	}
	if err = mgr.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if err = mgr.SetSessionValueByToken(ctx, token, "theme", "light"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("SetSessionValueByToken(after logout) error = %v, want ErrInvalidToken", err)
	}
}

func containsAccessSubject(subjects []AccessSubject, want AccessSubject) bool {
	for _, subject := range subjects {
		if subject == want {
			return true
		}
	}
	return false
}
