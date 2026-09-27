package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
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

// TestManagerIntrospectionRejectsForeignAuthType verifies foreign payloads cannot be adopted by the current namespace. TestManagerIntrospectionRejectsForeignAuthType 验证当前命名空间不能接管其他认证体系载荷。
func TestManagerIntrospectionRejectsForeignAuthType(t *testing.T) {
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
	if _, err := mgr.GetTokenInfo(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetTokenInfo(foreign namespace) error = %v, want ErrInvalidToken", err)
	}

	result, err := mgr.IntrospectToken(ctx, token)
	if err != nil {
		t.Fatalf("IntrospectToken() error = %v", err)
	}
	if result.Active {
		t.Fatalf("IntrospectToken() = %+v, want inactive foreign record", result)
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
