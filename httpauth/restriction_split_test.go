package httpauth

import (
	"context"
	"errors"
	"testing"
	"time"

	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// TestCheckRestrictionsRequireExplicitDeclaration covers device, account and index-free credential checks. TestCheckRestrictionsRequireExplicitDeclaration 覆盖设备、账号及脱离索引的凭据检查。
func TestCheckRestrictionsRequireExplicitDeclaration(t *testing.T) {
	ctx := context.Background()
	for _, restriction := range []string{"device", "device id", "retired account", "account without session"} {
		t.Run(restriction, func(t *testing.T) {
			mgr := newAuthcheckTestManager(t)
			token, err := mgr.Login(ctx, "restricted", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			want := derror.ErrDeviceDisabled
			var wantBase error
			switch restriction {
			case "device":
				err = mgr.DisableDevice(ctx, "restricted", "web", time.Hour)
			case "device id":
				err = mgr.DisableDeviceAndDeviceID(ctx, "restricted", "web", "browser", time.Hour)
			case "retired account":
				err = mgr.Disable(ctx, "restricted", time.Hour)
				want, wantBase = derror.ErrAccountDisabled, derror.ErrNotLogin
			case "account without session":
				cfg := mgr.GetConfig()
				if err = mgr.GetStorage().Delete(ctx, utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType)+manager.SessionKeyPrefix+"restricted"); err != nil {
					t.Fatal(err)
				}
				err = mgr.Disable(ctx, "restricted", time.Hour)
				want = derror.ErrAccountDisabled
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Check(ctx, mgr, Request{TokenValue: token, CheckLogin: true}); !errors.Is(err, wantBase) {
				t.Fatalf("base credential check = %v, want %v", err, wantBase)
			}
			for _, login := range []bool{false, true} {
				if _, err = Check(ctx, mgr, Request{TokenValue: token, CheckLogin: login, CheckDisable: true}); !errors.Is(err, want) {
					t.Fatalf("explicit restriction = %v, want %v", err, want)
				}
			}
			if _, err = Check(ctx, mgr, Request{TokenValue: "missing", CheckDisable: true}); !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("restriction-only request accepted missing credential: %v", err)
			}
		})
	}
}
