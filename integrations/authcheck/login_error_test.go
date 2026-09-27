package authcheck

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestCheckLoginPreservesFailureCauses separates invalid credentials from backend failures and restrictions. TestCheckLoginPreservesFailureCauses 区分无效凭证、后端故障与封禁限制。
func TestCheckLoginPreservesFailureCauses(t *testing.T) {
	cfg := newAuthcheckTestManager(t).GetConfig()
	storage := &loginErrorStorage{Storage: newAuthcheckTestStorage()}
	mgr := manager.NewManager(cfg, &authcheckTestGenerator{}, storage, authcheckTestCodec{}, adapter.NewNopLogger(), nil, nil)
	t.Cleanup(mgr.CloseManager)
	ctx := context.Background()
	token, err := mgr.Login(ctx, "user", "web")
	if err != nil {
		t.Fatal(err)
	}
	customLoginError := errors.New("custom login failure")
	check := func(token string, want error) {
		t.Helper()
		result, err := Check(ctx, mgr, Request{TokenValue: token, CheckLogin: true, LoginError: customLoginError})
		if !errors.Is(err, want) || (result == nil) != (want != nil) {
			t.Fatalf("Check()=%v,%v, want %v", result, err, want)
		}
	}
	check(token, nil)
	check("", customLoginError)
	check("missing-token", customLoginError)

	storage.err = errors.New("backend unavailable")
	check(token, derror.ErrStorageUnavailable)
	storage.err = context.Canceled
	check(token, derror.ErrStorageUnavailable)
	storage.err = nil

	if err := mgr.DisableDevice(ctx, "user", "web", time.Minute); err != nil {
		t.Fatal(err)
	}
	check(token, derror.ErrDeviceDisabled)
	if err := mgr.UntieDevice(ctx, "user", "web"); err != nil {
		t.Fatal(err)
	}
	check(token, nil)
	if err := mgr.Kickout(ctx, token); err != nil {
		t.Fatal(err)
	}
	check(token, customLoginError)
}

type loginErrorStorage struct {
	adapter.Storage
	err error
}

// Get injects a read failure without changing the stored login state. Get 注入读取故障，不改变已存储的登录态。
func (s *loginErrorStorage) Get(ctx context.Context, key string) (any, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.Storage.Get(ctx, key)
}
