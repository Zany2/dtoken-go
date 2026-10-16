package context

import (
	stdctx "context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestSessionWritesRevalidateToken verifies a revoked request token cannot mutate an existing account session. TestSessionWritesRevalidateToken 验证请求 Token 撤销后不能修改仍存在的账号会话。
func TestSessionWritesRevalidateToken(t *testing.T) {
	for _, operation := range []string{"set", "delete"} {
		t.Run(operation, func(t *testing.T) {
			ctx := stdctx.Background()
			_, req, base := newTestDTokenContext(t)
			storage := &sessionRevokingStorage{Storage: base.GetStorage()}
			mgr := manager.NewManager(base.GetConfig(), base.GetGenerator(), storage, base.GetSerializer(), base.GetLogger(), nil, nil)
			t.Cleanup(mgr.CloseManager)
			dctx := NewContext(req, mgr)
			token, err := mgr.Login(ctx, "session-user")
			if err != nil {
				t.Fatal(err)
			}
			if err = mgr.SetSessionValue(ctx, "session-user", "theme", "original"); err != nil {
				t.Fatal(err)
			}
			cfg := mgr.GetConfig()
			req.headers[cfg.TokenName] = token
			storage.sessionKey = cfg.KeyPrefix + cfg.AuthType + manager.SessionKeyPrefix + "session-user"
			storage.tokenKey = cfg.KeyPrefix + cfg.AuthType + config.TokenKeyPrefix + token
			storage.armed = true

			if operation == "set" {
				err = dctx.Session().SetValue(ctx, "theme", "changed")
			} else {
				err = dctx.Session().DeleteValue(ctx, "theme")
			}
			if !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("session %s error = %v, want ErrInvalidToken", operation, err)
			}
			if storage.armed {
				t.Fatal("token revocation hook was not reached")
			}
			value, exists, err := mgr.GetSessionValue(ctx, "session-user", "theme")
			if err != nil || !exists || value != "original" {
				t.Fatalf("session value = %v, %v, %v, want original", value, exists, err)
			}
		})
	}
}

// sessionRevokingStorage deterministically revokes the token between validation and the session mutation. sessionRevokingStorage 确定性地在校验与会话修改之间撤销 Token。
type sessionRevokingStorage struct {
	adapter.Storage
	sessionKey string
	tokenKey   string
	armed      bool
}

func (s *sessionRevokingStorage) Get(ctx stdctx.Context, key string) (any, error) {
	value, err := s.Storage.Get(ctx, key)
	if err == nil && s.armed && key == s.sessionKey {
		s.armed = false
		if err = s.Storage.Delete(ctx, s.tokenKey); err != nil {
			return nil, err
		}
	}
	return value, err
}
