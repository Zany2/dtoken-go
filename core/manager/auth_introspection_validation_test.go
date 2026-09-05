package manager

import (
	"context"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// TestManagerIntrospectionRejectsTokenExpiredBeforeTTLLookup verifies a token cannot remain active after disappearing between storage reads. TestManagerIntrospectionRejectsTokenExpiredBeforeTTLLookup 验证 Token 在两次存储读取之间消失后不会仍被判定为活跃。
func TestManagerIntrospectionRejectsTokenExpiredBeforeTTLLookup(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	token, err := mgr.Login(ctx, "introspection-expiry-boundary", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	mgr.storage = &managerIntrospectionExpireAfterReadStorage{
		Storage:  mgr.storage,
		tokenKey: mgr.getTokenKey(token),
	}
	result, err := mgr.IntrospectToken(ctx, token)
	if err != nil {
		t.Fatalf("IntrospectToken() error = %v", err)
	}
	if result.Active || result.Error != "invalid_token" || result.ExpiresIn != 0 {
		t.Fatalf("IntrospectToken(expired during lookup) = %+v, want inactive invalid_token", result)
	}
}

// TestManagerIntrospectionUsesCanonicalAuthType verifies the storage namespace remains authoritative over token payload metadata. TestManagerIntrospectionUsesCanonicalAuthType 验证存储命名空间优先于 Token 载荷中的认证类型。
func TestManagerIntrospectionUsesCanonicalAuthType(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	token, err := mgr.Login(ctx, "introspection-auth-type", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	tokenInfo, err := mgr.getTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("getTokenInfo() error = %v", err)
	}
	tokenInfo.AuthType = "foreign-auth-type:"
	if err = mgr.saveToStorage(ctx, mgr.getTokenKey(token), *tokenInfo); err != nil {
		t.Fatalf("save mismatched token info error = %v", err)
	}
	loadedInfo, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenInfo() error = %v", err)
	}
	if loadedInfo.AuthType != mgr.config.AuthType {
		t.Fatalf("GetTokenInfo().AuthType = %q, want %q", loadedInfo.AuthType, mgr.config.AuthType)
	}

	result, err := mgr.IntrospectToken(ctx, token)
	if err != nil {
		t.Fatalf("IntrospectToken() error = %v", err)
	}
	if !result.Active || result.AuthType != mgr.config.AuthType {
		t.Fatalf("IntrospectToken() = %+v, want active auth type %q", result, mgr.config.AuthType)
	}
}

type managerIntrospectionExpireAfterReadStorage struct {
	adapter.Storage
	tokenKey string
}

func (s *managerIntrospectionExpireAfterReadStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.Storage.Get(ctx, key)
	if err == nil && key == s.tokenKey {
		_ = s.Storage.Delete(ctx, key)
	}
	return value, err
}
