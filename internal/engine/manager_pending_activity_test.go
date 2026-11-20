package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerQueuedActivityRemainsValid covers public operations while accepted activity is waiting for a worker. TestManagerQueuedActivityRemainsValid 覆盖已接受的活跃时间仍在线程池排队时的公开操作。
func TestManagerQueuedActivityRemainsValid(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"check", "terminal", "login cleanup", "shared login", "manual renewal", "refresh", "kickout"} {
		t.Run(operation, func(t *testing.T) {
			mgr := newTerminalLifecycleTestManager(t, func(cfg *config.Config) {
				cfg.IsShare = operation == "shared login"
			})
			pair, err := mgr.LoginWithRefreshToken(ctx, "queued-active", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			if err = mgr.CheckLogin(ctx, pair.AccessToken); err != nil {
				t.Fatal(err)
			}
			if pool.taskCount() != 1 {
				t.Fatal("accepted request did not queue activity")
			}
			if err = mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
				t.Fatal(err)
			}
			events := registerTerminalLifecycleEvents(mgr)
			switch operation {
			case "check":
				err = mgr.CheckLogin(ctx, pair.AccessToken)
			case "terminal":
				var count int
				count, err = mgr.GetOnlineTerminalCount(ctx, pair.LoginID)
				if count != 1 {
					t.Fatal("queued activity was reported offline")
				}
			case "login cleanup":
				_, err = mgr.Login(ctx, pair.LoginID, "mobile", "phone")
			case "shared login":
				var shared string
				shared, err = mgr.Login(ctx, pair.LoginID, "web", "browser")
				if shared != pair.AccessToken {
					t.Fatalf("shared login replaced an active token: %q, %v", shared, err)
				}
			case "manual renewal":
				err = mgr.RenewTimeout(ctx, pair.AccessToken, time.Minute)
			case "refresh":
				var next *RefreshTokenPair
				next, err = mgr.RefreshToken(ctx, pair.RefreshToken)
				if next == nil {
					t.Fatalf("refresh rejected queued activity: %v", err)
				}
			case "kickout":
				err = mgr.Kickout(ctx, pair.AccessToken)
				if _, stateErr := mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(stateErr, derror.ErrTokenKickout) {
					t.Fatalf("kickout lost its explicit state: %v", stateErr)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if operation != "refresh" && operation != "kickout" {
				if _, _, err = mgr.checkLoginAndGetContextNoRenew(ctx, pair.AccessToken); err != nil {
					t.Fatalf("operation retired the active token: %v", err)
				}
			}
			for _, event := range events() {
				if event.Event == listener.EventActiveTimeout {
					t.Fatal("queued activity caused an active-timeout event")
				}
			}
		})
	}
}

// TestManagerPendingActivityRequiresValidLifecycle checks cancellation, reuse and invalid persisted activity. TestManagerPendingActivityRequiresValidLifecycle 校验取消、复用及无效持久化活跃标记。
func TestManagerPendingActivityRequiresValidLifecycle(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"matching", "cancelled", "reused", "revoked", "stale", "missing", "corrupt", "read failure"} {
		t.Run(state, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
			token, err := mgr.Login(ctx, "pending-lifecycle", "web")
			if err != nil {
				t.Fatal(err)
			}
			record, err := mgr.getTokenRecord(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			old := time.Now().Add(-2 * time.Minute).Unix()
			if err = mgr.storage.Set(ctx, mgr.getActiveKey(token), old, time.Hour); err != nil {
				t.Fatal(err)
			}
			pending := *record
			activeAt := time.Now().Unix()
			if state == "reused" {
				pending.AccessID += "-earlier"
			}
			if state == "stale" {
				activeAt = old
			}
			mgr.beginLoginMaintenance(token, &pending, activeAt, false)
			switch state {
			case "cancelled":
				mgr.cancelLoginMaintenance(token)
			case "revoked":
				record.Revoked = true
				err = mgr.saveToStorage(ctx, mgr.getTokenKey(token), *record, time.Hour)
			case "missing":
				err = mgr.storage.Delete(ctx, mgr.getActiveKey(token))
			case "corrupt":
				err = mgr.storage.Set(ctx, mgr.getActiveKey(token), "bad timestamp", time.Hour)
			case "read failure":
				mgr.storage = &managerSessionReadErrorStorage{Storage: mgr.storage, key: mgr.getActiveKey(token)}
			}
			if err != nil {
				t.Fatal(err)
			}
			_, _, expired, inspectErr := mgr.inspectLoginToken(ctx, token, nil)
			switch state {
			case "missing", "corrupt", "revoked":
				if !errors.Is(inspectErr, derror.ErrInvalidToken) {
					t.Fatalf("invalid activity error = %v", inspectErr)
				}
			case "read failure":
				if !errors.Is(inspectErr, derror.ErrStorageUnavailable) {
					t.Fatalf("storage error = %v", inspectErr)
				}
			default:
				if inspectErr != nil || expired != (state != "matching") {
					t.Fatalf("expired/error = %v/%v", expired, inspectErr)
				}
			}
			alive, aliveErr := mgr.checkTerminalTokenAlive(ctx, token)
			if alive != (state == "matching") || (aliveErr != nil && state != "read failure") {
				t.Fatalf("terminal alive/error = %v/%v", alive, aliveErr)
			}
		})
	}
}

// TestManagerActivitySnapshotSurvivesWorkerCompletion checks completion between the activity read and timeout comparison. TestManagerActivitySnapshotSurvivesWorkerCompletion 校验读取活跃标记与超时比较之间任务完成的交错。
func TestManagerActivitySnapshotSurvivesWorkerCompletion(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
	token, err := mgr.Login(ctx, "pending-read-race", "web")
	if err != nil {
		t.Fatal(err)
	}
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err = mgr.storage.Set(ctx, mgr.getActiveKey(token), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
		t.Fatal(err)
	}
	storage := &managerCompleteActivityReadStorage{Storage: mgr.storage, key: mgr.getActiveKey(token), complete: pool.runAll}
	mgr.storage = storage
	if _, _, expired, err := mgr.inspectLoginToken(ctx, token, nil); err != nil || expired {
		t.Fatalf("completed activity expired/error = %v/%v", expired, err)
	}
	if storage.complete != nil || pool.taskCount() != 0 {
		t.Fatal("worker did not complete during the activity read")
	}
}

// managerCompleteActivityReadStorage returns an old marker after completing its pending update. managerCompleteActivityReadStorage 完成排队更新后返回旧活跃标记。
type managerCompleteActivityReadStorage struct {
	adapter.Storage
	key      string
	complete func()
}

// Get completes maintenance after capturing the stored value. Get 在捕获存储值后完成维护任务。
func (s *managerCompleteActivityReadStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.Storage.Get(ctx, key)
	if key == s.key && err == nil && s.complete != nil {
		complete := s.complete
		s.complete = nil
		complete()
	}
	return value, err
}
