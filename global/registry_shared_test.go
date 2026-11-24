package global

import (
	"context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestRootBuildSharesLegacyRegistry verifies automatic registration and legacy replacement use one registry. TestRootBuildSharesLegacyRegistry 验证自动注册与旧入口替换共用注册表。
func TestRootBuildSharesLegacyRegistry(t *testing.T) {
	dtoken.CloseAll()
	t.Cleanup(dtoken.CloseAll)
	b := dtoken.NewBuilder().AuthType("shared").IsPrintBanner(false).IsLog(false).AutoRenew(false)
	auth := b.MustBuild()
	mgr, err := GetManager("shared")
	if err != nil || mgr != auth.Manager() {
		t.Fatalf("legacy lookup = %v, %v", mgr, err)
	}
	token, err := Login(context.Background(), "user", "", "", "shared")
	if err != nil || !dtoken.AuthType("shared").IsLogin(context.Background(), token) {
		t.Fatalf("legacy login was not visible to the root facade: %v", err)
	}
	replacement, err := BuildAndSetManager(b)
	if err != nil || !auth.IsClosed() || dtoken.AuthType("shared").Manager() != replacement {
		t.Fatalf("legacy replacement failed: %v", err)
	}
	DeleteAllManager()
	if !replacement.IsClosed() {
		t.Fatal("legacy deletion did not close the shared instance")
	}
	if _, err := dtoken.GetManager("shared"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("root lookup after legacy deletion = %v", err)
	}
}
