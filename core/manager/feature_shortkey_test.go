package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/shortkey"
)

func TestManagerShortKeyConfirmAndConsume(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithShortKey(t, nil)

	created, err := mgr.CreateShortKey(ctx, shortkey.CreateOptions{
		Scene:     "qr-login",
		SourceApp: "portal",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("CreateShortKey() error = %v", err)
	}
	if len(created.Key) != shortkey.DefaultLength || created.Status != shortkey.StatusPending {
		t.Fatalf("CreateShortKey() = %+v, want default-length pending key", created)
	}
	if _, err = mgr.ConsumeShortKey(ctx, created.Key); !errors.Is(err, shortkey.ErrShortKeyPending) {
		t.Fatalf("ConsumeShortKey() pending error = %v, want ErrShortKeyPending", err)
	}

	confirmed, err := mgr.ConfirmShortKey(ctx, created.Key, shortkey.ConfirmOptions{
		LoginID:  "user-1001",
		Device:   "mobile",
		DeviceId: "ios-1",
		Scopes:   []string{"profile"},
		Extra:    map[string]any{"confirmedBy": "scan"},
	})
	if err != nil {
		t.Fatalf("ConfirmShortKey() error = %v", err)
	}
	if confirmed.Status != shortkey.StatusConfirmed || confirmed.LoginID != "user-1001" {
		t.Fatalf("ConfirmShortKey() = %+v, want confirmed user", confirmed)
	}

	validated, err := mgr.ValidateShortKey(ctx, created.Key, shortkey.ValidateOptions{
		LoginID:   "user-1001",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("ValidateShortKey() error = %v", err)
	}
	if validated.Scene != "qr-login" || validated.DeviceId != "ios-1" {
		t.Fatalf("ValidateShortKey() = %+v, want preserved scene/device", validated)
	}

	result, err := mgr.ConsumeShortKey(ctx, created.Key, shortkey.ValidateOptions{
		LoginID:   "user-1001",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("ConsumeShortKey() error = %v", err)
	}
	if result.ShortKey.Status != shortkey.StatusConsumed || result.ShortKey.LoginID != "user-1001" {
		t.Fatalf("ConsumeShortKey() = %+v, want consumed user key", result.ShortKey)
	}
	status, err := mgr.GetShortKeyStatus(ctx, created.Key)
	if err != nil {
		t.Fatalf("GetShortKeyStatus() error = %v", err)
	}
	if status != shortkey.StatusConsumed {
		t.Fatalf("GetShortKeyStatus() = %s, want consumed", status)
	}
}

func TestManagerShortKeyBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithShortKey(t, nil)

	created, err := mgr.CreateShortKey(ctx, shortkey.CreateOptions{
		LoginID:   "user-1001",
		Scene:     "share",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("CreateShortKey() error = %v", err)
	}
	if _, err = mgr.ValidateShortKey(ctx, created.Key, shortkey.ValidateOptions{TargetApp: "console"}); !errors.Is(err, shortkey.ErrShortKeyMismatch) {
		t.Fatalf("ValidateShortKey() mismatch error = %v, want ErrShortKeyMismatch", err)
	}

	if err = mgr.RevokeShortKey(ctx, created.Key); err != nil {
		t.Fatalf("RevokeShortKey() error = %v", err)
	}
	status, err := mgr.GetShortKeyStatus(ctx, created.Key)
	if err != nil {
		t.Fatalf("GetShortKeyStatus() revoked error = %v", err)
	}
	if status != shortkey.StatusRevoked {
		t.Fatalf("GetShortKeyStatus() = %s, want revoked", status)
	}
	if _, err = mgr.ValidateShortKey(ctx, created.Key); !errors.Is(err, shortkey.ErrShortKeyRevoked) {
		t.Fatalf("ValidateShortKey() revoked error = %v, want ErrShortKeyRevoked", err)
	}

	expiring, err := mgr.CreateShortKeyWithTimeout(ctx, shortkey.CreateOptions{LoginID: "user-1002"}, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("CreateShortKeyWithTimeout() error = %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	status, err = mgr.GetShortKeyStatus(ctx, expiring.Key)
	if err != nil {
		t.Fatalf("GetShortKeyStatus() expired error = %v", err)
	}
	if status != shortkey.StatusInvalid {
		t.Fatalf("GetShortKeyStatus() after storage expiry = %s, want invalid", status)
	}
	ttl, err := mgr.GetShortKeyTTL(ctx, expiring.Key)
	if err != nil {
		t.Fatalf("GetShortKeyTTL() error = %v", err)
	}
	if ttl != -2 {
		t.Fatalf("GetShortKeyTTL() = %d, want -2", ttl)
	}
}

func newTestManagerWithShortKey(t *testing.T, mutate func(*config.Config)) *Manager {
	t.Helper()
	mgr := newTestManager(t, mutate)
	WithShortKeyManager(shortkey.NewDefaultManager(
		mgr.GetConfig().AuthType,
		mgr.GetConfig().KeyPrefix,
		mgr.GetStorage(),
		mgr.GetSerializer(),
	))(mgr)
	return mgr
}
