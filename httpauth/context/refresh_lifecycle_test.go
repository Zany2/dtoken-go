package context

import (
	"context"
	"testing"
	"time"

	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// TestGroupedRefreshLifecycle checks scoped request copies and refresh helper lifecycle. TestGroupedRefreshLifecycle 验证请求作用域副本和刷新辅助入口的生命周期。
func TestGroupedRefreshLifecycle(t *testing.T) {
	ctx := context.Background()
	dctx, req, mgr := newTestDTokenContext(t)
	pair, err := dctx.Refresh().LoginWithOptions(ctx, manager.RefreshTokenOptions{LoginOptions: manager.LoginOptions{LoginID: "user", Device: "web"}, RefreshTimeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if ttl, err := dctx.Refresh().GetTTL(ctx, pair.RefreshToken); err != nil || ttl <= 0 || ttl > 60 {
		t.Fatalf("refresh ttl=%d,%v", ttl, err)
	}
	req.headers[mgr.GetConfig().TokenName] = pair.AccessToken
	scoped := dctx.WithRequestContext(&testRequestContext{headers: map[string]string{mgr.GetConfig().TokenName: "different"}})
	if dctx.GetTokenValue() != pair.AccessToken || scoped.GetTokenValue() != "different" || scoped.GetManager() != mgr {
		t.Fatal("request copy changed parent or manager binding")
	}
	rotated, err := scoped.Refresh().Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if mgr.IsLogin(ctx, pair.AccessToken) || !mgr.IsLogin(ctx, rotated.AccessToken) {
		t.Fatal("rotation failed")
	}
	if _, err = dctx.Refresh().Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("refresh replay accepted")
	}
	if err = dctx.Refresh().Revoke(ctx, rotated.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if mgr.IsLogin(ctx, rotated.AccessToken) {
		t.Fatal("revoked pair remains active")
	}
	if ttl, err := dctx.Refresh().GetTTL(ctx, rotated.RefreshToken); err != nil || ttl != -2 {
		t.Fatalf("revoked ttl=%d,%v", ttl, err)
	}
}
