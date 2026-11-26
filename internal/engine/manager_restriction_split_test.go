package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestManagerCredentialChecksIgnoreRestrictionStorage isolates credential validity from restriction reads and failures. TestManagerCredentialChecksIgnoreRestrictionStorage 将凭据有效性与封禁读取及故障隔离。
func TestManagerCredentialChecksIgnoreRestrictionStorage(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	token, err := mgr.Login(ctx, "restriction-split", "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	if err = mgr.DisableDeviceAndDeviceID(ctx, "restriction-split", "web", "browser", time.Hour); err != nil {
		t.Fatal(err)
	}
	if !mgr.IsLogin(ctx, token) {
		t.Fatal("device restriction invalidated credential")
	}
	if _, err = CheckTokenAccess(ctx, mgr, token, nil, nil, false, true); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("explicit restriction = %v", err)
	}
	info, err := mgr.IntrospectToken(ctx, token)
	if err != nil || info.Active {
		t.Fatalf("introspection bypassed restriction: %+v, %v", info, err)
	}
	if _, err = mgr.Login(ctx, "restriction-split", "web", "browser"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("issuance bypassed restriction: %v", err)
	}

	// A corrupt restriction marker is visible only to operations requesting that restriction. 损坏的封禁记录只影响要求该限制的操作。
	if err = mgr.storage.Set(ctx, mgr.getDisableKey("restriction-split"), "{", time.Hour); err != nil {
		t.Fatal(err)
	}
	storage := &managerGetCountingStorage{Storage: mgr.storage}
	mgr.storage = storage
	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatal(err)
	}
	if id, err := mgr.GetLoginID(ctx, token); err != nil || id != "restriction-split" {
		t.Fatalf("identity = %q, %v", id, err)
	}
	if storage.getCount(mgr.getDisableKey("restriction-split")) != 0 {
		t.Fatal("base check read restriction storage")
	}
	if _, err = CheckTokenAccess(ctx, mgr, token, nil, nil, false, true); !errors.Is(err, derror.ErrSerializeFailed) {
		t.Fatalf("explicit restriction hid corrupt marker: %v", err)
	}
}

// TestManagerRestrictedAccessDoesNotQueueMaintenance keeps rejected access from refreshing activity or token TTL. TestManagerRestrictedAccessDoesNotQueueMaintenance 防止被拒绝的访问刷新活跃时间或 Token TTL。
func TestManagerRestrictedAccessDoesNotQueueMaintenance(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
	token, err := mgr.Login(ctx, "denied-maintenance", "web")
	if err != nil {
		t.Fatal(err)
	}
	if err = mgr.DisableDevice(ctx, "denied-maintenance", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	if _, err = CheckTokenAccess(ctx, mgr, token, nil, nil, false, true); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("restriction = %v", err)
	}
	if pool.taskCount() != 0 {
		t.Fatal("denied access queued maintenance")
	}
	if _, err = CheckTokenAccess(ctx, mgr, token, nil, nil, false, false); err != nil {
		t.Fatal(err)
	}
	if pool.taskCount() != 1 {
		t.Fatal("valid credential did not queue normal maintenance")
	}
}
