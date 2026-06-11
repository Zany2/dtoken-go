package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

func TestManagerOAuth2FacadeFlowAndEvents(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithOAuth2(t)

	var events []*listener.EventData
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		copyData := *data
		events = append(events, &copyData)
	}, listener.ListenerConfig{Async: false})

	client := managerOAuth2TestClient()
	if err := mgr.RegisterOAuth2Client(client); err != nil {
		t.Fatalf("RegisterOAuth2Client() error = %v", err)
	}
	gotClient, err := mgr.GetOAuth2Client(client.ClientID)
	if err != nil {
		t.Fatalf("GetOAuth2Client() error = %v", err)
	}
	if gotClient.ClientID != client.ClientID {
		t.Fatalf("GetOAuth2Client() ClientID = %q, want %q", gotClient.ClientID, client.ClientID)
	}

	code, err := mgr.GenerateOAuth2AuthorizationCode(ctx, client.ClientID, "oauth-user", client.RedirectURIs[0], []string{"read"})
	if err != nil {
		t.Fatalf("GenerateOAuth2AuthorizationCode() error = %v", err)
	}
	token, err := mgr.ExchangeOAuth2CodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
	if err != nil {
		t.Fatalf("ExchangeOAuth2CodeForToken() error = %v", err)
	}
	if token.Token == "" || token.RefreshToken == "" || token.UserID != "oauth-user" {
		t.Fatalf("AccessToken = %+v, want populated user token", token)
	}
	if !mgr.ValidateOAuth2AccessToken(ctx, token.Token) {
		t.Fatal("ValidateOAuth2AccessToken() = false, want true")
	}
	info, err := mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, token.Token)
	if err != nil {
		t.Fatalf("ValidateOAuth2AccessTokenAndGetInfo() error = %v", err)
	}
	if info.Token != token.Token || info.ClientID != client.ClientID {
		t.Fatalf("ValidateOAuth2AccessTokenAndGetInfo() = %+v, want token/client preserved", info)
	}

	refreshed, err := mgr.RefreshOAuth2AccessToken(ctx, client.ClientID, token.RefreshToken, client.ClientSecret)
	if err != nil {
		t.Fatalf("RefreshOAuth2AccessToken() error = %v", err)
	}
	if refreshed.Token == token.Token || refreshed.RefreshToken == token.RefreshToken {
		t.Fatal("RefreshOAuth2AccessToken() reused old token pair, want rotation")
	}
	if err = mgr.RevokeOAuth2Token(ctx, refreshed.Token); err != nil {
		t.Fatalf("RevokeOAuth2Token() error = %v", err)
	}
	if mgr.ValidateOAuth2AccessToken(ctx, refreshed.Token) {
		t.Fatal("ValidateOAuth2AccessToken() after revoke = true, want false")
	}

	passwordToken, err := mgr.OAuth2PasswordGrantToken(ctx, client.ClientID, client.ClientSecret, "alice", "secret", []string{"write"}, func(username, password string) (string, error) {
		if username == "alice" && password == "secret" {
			return "user-alice", nil
		}
		return "", derror.ErrInvalidUserCredentials
	})
	if err != nil {
		t.Fatalf("OAuth2PasswordGrantToken() error = %v", err)
	}
	if passwordToken.UserID != "user-alice" {
		t.Fatalf("OAuth2PasswordGrantToken() UserID = %q, want user-alice", passwordToken.UserID)
	}

	clientToken, err := mgr.OAuth2Token(ctx, &oauth2.TokenRequest{
		GrantType:    oauth2.GrantTypeClientCredentials,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Scopes:       []string{"read"},
	}, nil)
	if err != nil {
		t.Fatalf("OAuth2Token(client_credentials) error = %v", err)
	}
	if clientToken.UserID != client.ClientID {
		t.Fatalf("OAuth2Token(client_credentials) UserID = %q, want client id", clientToken.UserID)
	}

	pkceCode, err := mgr.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, client.ClientID, "oauth-pkce-user", client.RedirectURIs[0], []string{"read"}, "plain-verifier", oauth2.CodeChallengeMethodPlain)
	if err != nil {
		t.Fatalf("GenerateOAuth2AuthorizationCodeWithPKCE() error = %v", err)
	}
	pkceToken, err := mgr.ExchangeOAuth2CodeForTokenWithPKCE(ctx, pkceCode.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], "plain-verifier")
	if err != nil {
		t.Fatalf("ExchangeOAuth2CodeForTokenWithPKCE() error = %v", err)
	}
	if pkceToken.UserID != "oauth-pkce-user" {
		t.Fatalf("ExchangeOAuth2CodeForTokenWithPKCE() UserID = %q, want oauth-pkce-user", pkceToken.UserID)
	}

	if err = mgr.UnregisterOAuth2Client(client.ClientID); err != nil {
		t.Fatalf("UnregisterOAuth2Client() error = %v", err)
	}
	if _, err = mgr.GetOAuth2Client(client.ClientID); !errors.Is(err, derror.ErrClientNotFound) {
		t.Fatalf("GetOAuth2Client(after unregister) error = %v, want ErrClientNotFound", err)
	}

	assertManagerEvent(t, events, listener.EventOAuth2ClientRegister, "", "", "", "", map[string]any{
		listener.ExtraKeyAction:   listener.ActionRegister,
		listener.ExtraKeyClientID: client.ClientID,
	})
	assertManagerEvent(t, events, listener.EventOAuth2CodeGenerate, "oauth-user", "", "", code.Code, map[string]any{
		listener.ExtraKeyAction:   listener.ActionCreate,
		listener.ExtraKeyClientID: client.ClientID,
	})
	assertManagerEvent(t, events, listener.EventOAuth2TokenIssue, "oauth-user", "", "", token.Token, map[string]any{
		listener.ExtraKeyAction:    listener.ActionIssue,
		listener.ExtraKeyGrantType: string(oauth2.GrantTypeAuthorizationCode),
	})
	assertManagerEvent(t, events, listener.EventOAuth2TokenRefresh, "oauth-user", "", "", refreshed.Token, map[string]any{
		listener.ExtraKeyAction:    listener.ActionRefresh,
		listener.ExtraKeyGrantType: string(oauth2.GrantTypeRefreshToken),
	})
	assertManagerEvent(t, events, listener.EventOAuth2TokenRevoke, "oauth-user", "", "", refreshed.Token, map[string]any{
		listener.ExtraKeyAction: listener.ActionRevoke,
	})
	assertManagerEvent(t, events, listener.EventOAuth2TokenIssue, "user-alice", "", "", passwordToken.Token, map[string]any{
		listener.ExtraKeyAction:    listener.ActionIssue,
		listener.ExtraKeyGrantType: string(oauth2.GrantTypePassword),
	})
	assertManagerEvent(t, events, listener.EventOAuth2TokenIssue, client.ClientID, "", "", clientToken.Token, map[string]any{
		listener.ExtraKeyAction:    listener.ActionIssue,
		listener.ExtraKeyGrantType: string(oauth2.GrantTypeClientCredentials),
	})
	assertManagerEvent(t, events, listener.EventOAuth2CodeGenerate, "oauth-pkce-user", "", "", pkceCode.Code, map[string]any{
		listener.ExtraKeyAction:   listener.ActionCreate,
		listener.ExtraKeyClientID: client.ClientID,
	})
	assertManagerEvent(t, events, listener.EventOAuth2TokenIssue, "oauth-pkce-user", "", "", pkceToken.Token, map[string]any{
		listener.ExtraKeyAction:    listener.ActionIssue,
		listener.ExtraKeyGrantType: string(oauth2.GrantTypeAuthorizationCode),
	})
	assertManagerEvent(t, events, listener.EventOAuth2ClientUnregister, "", "", "", "", map[string]any{
		listener.ExtraKeyAction:   listener.ActionUnregister,
		listener.ExtraKeyClientID: client.ClientID,
	})
}

func TestManagerOAuth2TokenEndpointRefreshEvent(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithOAuth2(t)

	var events []*listener.EventData
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		copyData := *data
		events = append(events, &copyData)
	}, listener.ListenerConfig{Async: false})

	client := managerOAuth2TestClient()
	if err := mgr.RegisterOAuth2Client(client); err != nil {
		t.Fatalf("RegisterOAuth2Client() error = %v", err)
	}
	issued, err := mgr.OAuth2ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
	if err != nil {
		t.Fatalf("OAuth2ClientCredentialsToken() error = %v", err)
	}
	refreshed, err := mgr.OAuth2Token(ctx, &oauth2.TokenRequest{
		GrantType:    oauth2.GrantTypeRefreshToken,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		RefreshToken: issued.RefreshToken,
	}, nil)
	if err != nil {
		t.Fatalf("OAuth2Token(refresh_token) error = %v", err)
	}

	assertManagerEvent(t, events, listener.EventOAuth2TokenRefresh, client.ClientID, "", "", refreshed.Token, map[string]any{
		listener.ExtraKeyAction:    listener.ActionIssue,
		listener.ExtraKeyGrantType: string(oauth2.GrantTypeRefreshToken),
	})
}

func TestManagerNonceEvents(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithNonce(t)

	var events []*listener.EventData
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		copyData := *data
		events = append(events, &copyData)
	}, listener.ListenerConfig{Async: false})

	value, err := mgr.GenerateNonceWithTimeout(ctx, time.Minute)
	if err != nil {
		t.Fatalf("GenerateNonceWithTimeout() error = %v", err)
	}
	if !mgr.VerifyNonce(ctx, value) {
		t.Fatal("VerifyNonce(first) = false, want true")
	}
	if mgr.VerifyNonce(ctx, value) {
		t.Fatal("VerifyNonce(second) = true, want false")
	}

	assertManagerEvent(t, events, listener.EventNonceGenerate, "", "", "", value, map[string]any{
		listener.ExtraKeyAction: listener.ActionCreate,
		listener.ExtraKeyTTL:    int64(60),
	})
	assertManagerEvent(t, events, listener.EventNonceVerify, "", "", "", value, map[string]any{
		listener.ExtraKeyAction: listener.ActionConsume,
		listener.ExtraKeyResult: true,
	})
	assertManagerEvent(t, events, listener.EventNonceVerify, "", "", "", value, map[string]any{
		listener.ExtraKeyAction: listener.ActionConsume,
		listener.ExtraKeyResult: false,
	})
}

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

func newTestManagerWithOAuth2(t *testing.T) *Manager {
	t.Helper()

	mgr := newTestManager(t, nil)
	WithOAuth2Manager(oauth2.NewDefaultOAuth2Server(
		mgr.GetConfig().AuthType,
		mgr.GetConfig().KeyPrefix,
		mgr.GetStorage(),
		managerTestCodec{},
	))(mgr)
	return mgr
}

func managerOAuth2TestClient() *oauth2.Client {
	return &oauth2.Client{
		ClientID:     "client-manager",
		ClientSecret: "secret-manager",
		RedirectURIs: []string{"https://example.com/callback"},
		GrantTypes: []oauth2.GrantType{
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeRefreshToken,
			oauth2.GrantTypeClientCredentials,
			oauth2.GrantTypePassword,
		},
		Scopes: []string{"read", "write"},
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
