package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestManagerTerminalQueriesRejectStaleIdentity checks every terminal identity dimension. TestManagerTerminalQueriesRejectStaleIdentity 验证终端查询核对全部身份维度。
func TestManagerTerminalQueriesRejectStaleIdentity(t *testing.T) {
	ctx := context.Background()
	for _, change := range []struct {
		name  string
		apply func(*TerminalInfo)
	}{
		{"account", func(ti *TerminalInfo) { ti.LoginID = "foreign" }},
		{"device", func(ti *TerminalInfo) { ti.Device = "stale-device" }},
		{"device ID", func(ti *TerminalInfo) { ti.DeviceID = "stale-id" }},
		{"created", func(ti *TerminalInfo) { ti.CreateTime-- }},
		{"sequence", func(ti *TerminalInfo) { ti.Index++ }},
	} {
		t.Run(change.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 30 })
			id := "query-lifecycle"
			token, err := mgr.Login(ctx, id, "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			sess, err := mgr.getSession(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			current := sess.TerminalInfos[0]
			current.Extra = map[string]any{"version": "current"}
			stale := current
			stale.Extra = map[string]any{"version": "stale"}
			change.apply(&stale)
			sess.TerminalInfos = []TerminalInfo{stale}
			if err := mgr.saveToStorage(ctx, mgr.getSessionKey(id), *sess); err != nil {
				t.Fatal(err)
			}
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getSessionKey(id), mgr.getTokenKey(token), mgr.getActiveKey(token)})
			for _, query := range []func() ([]string, error){
				func() ([]string, error) { return mgr.GetTokenValueListByLoginID(ctx, id, true) },
				func() ([]string, error) { return mgr.GetTokenValueListByDevice(ctx, id, stale.Device, true) },
				func() ([]string, error) {
					return mgr.GetTokenValueListByDeviceAndDeviceID(ctx, id, stale.Device, stale.DeviceID, true)
				},
			} {
				if tokens, err := query(); err != nil || len(tokens) != 0 {
					t.Fatalf("stale alive list = %v, %v", tokens, err)
				}
			}
			for _, query := range []func() (int, error){
				func() (int, error) { return mgr.GetOnlineTerminalCount(ctx, id) },
				func() (int, error) { return mgr.GetOnlineTerminalCountByDevice(ctx, id, stale.Device) },
				func() (int, error) {
					return mgr.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, id, stale.Device, stale.DeviceID)
				},
			} {
				if count, err := query(); err != nil || count != 0 {
					t.Fatalf("stale online count = %d, %v", count, err)
				}
			}
			for _, filter := range [][]string{nil, {stale.Device}} {
				if value, err := mgr.GetTokenValueByLoginID(ctx, id, filter...); value != "" || !errors.Is(err, derror.ErrInvalidToken) {
					t.Fatalf("stale latest token = %q, %v", value, err)
				}
			}
			if info, err := mgr.GetTerminalInfoByToken(ctx, token); info != nil || !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("stale detail = %+v, %v", info, err)
			}
			if pool.taskCount() != 0 {
				t.Fatal("failed detail query scheduled maintenance")
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)

			// Raw queries remain snapshots, including stale entries; stopping a visitor must work. 原始查询仍返回包含陈旧条目的快照，并支持回调提前停止。
			if raw, err := mgr.GetTokenValueListByLoginID(ctx, id); err != nil || !reflect.DeepEqual(raw, []string{token}) {
				t.Fatalf("raw list = %v, %v", raw, err)
			}
			if raw, err := mgr.GetTerminalListByLoginID(ctx, id); err != nil || len(raw) != 1 || !reflect.DeepEqual(raw[0], stale) {
				t.Fatalf("raw terminals = %+v, %v", raw, err)
			}
			visited := 0
			if err := mgr.ForEachTerminal(ctx, id, func(ti TerminalInfo) bool { visited++; return false }); err != nil || visited != 1 {
				t.Fatalf("visitor count = %d, %v", visited, err)
			}

			// The stale first and last entries must not hide or double-count the matching lifecycle. 首尾陈旧条目不能掩盖或重复统计中间的有效生命周期。
			sess.TerminalInfos = []TerminalInfo{stale, current, stale}
			if err := mgr.saveToStorage(ctx, mgr.getSessionKey(id), *sess); err != nil {
				t.Fatal(err)
			}
			if count, err := mgr.GetOnlineTerminalCount(ctx, id); err != nil || count != 1 {
				t.Fatalf("matching count = %d, %v", count, err)
			}
			if values, err := mgr.GetTokenValueListByLoginID(ctx, id, true); err != nil || !reflect.DeepEqual(values, []string{token}) {
				t.Fatalf("matching tokens = %v, %v", values, err)
			}
			if value, err := mgr.GetTokenValueByLoginID(ctx, id); err != nil || value != token {
				t.Fatalf("matching latest = %q, %v", value, err)
			}
			if info, err := mgr.GetTerminalInfoByToken(ctx, token); err != nil || info == nil || !reflect.DeepEqual(*info, current) {
				t.Fatalf("matching detail = %+v, %v", info, err)
			}
			if pool.taskCount() != 1 {
				t.Fatal("successful detail query did not preserve activity maintenance")
			}
		})
	}
}

// TestManagerSessionQueriesRejectExtraOptions verifies variadic arguments are never silently ignored. TestManagerSessionQueriesRejectExtraOptions 验证可变参数不会被静默忽略。
func TestManagerSessionQueriesRejectExtraOptions(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	storage := &managerGetCountingStorage{Storage: &managerFailingStorage{Storage: mgr.storage, getErr: errors.New("unexpected read")}}
	mgr.storage = storage
	for _, query := range []func(string) error{
		func(id string) error { _, err := mgr.GetTokenValueListByLoginID(ctx, id, false, true); return err },
		func(id string) error {
			_, err := mgr.GetTokenValueListByDevice(ctx, id, "web", false, true)
			return err
		},
		func(id string) error {
			_, err := mgr.GetTokenValueListByDeviceAndDeviceID(ctx, id, "web", "browser", false, true)
			return err
		},
		func(id string) error { _, err := mgr.GetTerminalListByLoginID(ctx, id, "web", "mobile"); return err },
		func(id string) error { _, err := mgr.GetTokenValueByLoginID(ctx, id, "web", "mobile"); return err },
	} {
		if err := query("query-options"); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("extra option error = %v", err)
		}
		if err := query(""); !errors.Is(err, derror.ErrIDIsEmpty) {
			t.Fatalf("empty ID precedence = %v", err)
		}
	}
	if storage.getCount(mgr.getSessionKey("query-options")) != 0 {
		t.Fatal("invalid query read storage")
	}
}

// TestManagerSessionDataPreservesStoredStateOnEncodingFailure covers nil values and rejected writes on both APIs. TestManagerSessionDataPreservesStoredStateOnEncodingFailure 覆盖两类数据接口的空值语义和编码失败保护。
func TestManagerSessionDataPreservesStoredStateOnEncodingFailure(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	id := "session-data-review"
	token, err := mgr.Login(ctx, id, "web")
	if err != nil {
		t.Fatal(err)
	}
	for _, api := range []struct {
		name string
		set  func(string, any) error
		get  func(string) (any, bool, error)
		del  func(string) error
	}{
		{"account", func(k string, v any) error { return mgr.SetSessionValue(ctx, id, k, v) }, func(k string) (any, bool, error) { return mgr.GetSessionValue(ctx, id, k) }, func(k string) error { return mgr.DeleteSessionValue(ctx, id, k) }},
		{"token", func(k string, v any) error { return mgr.SetSessionValueByToken(ctx, token, k, v) }, func(k string) (any, bool, error) { return mgr.GetSessionValueByToken(ctx, token, k) }, func(k string) error { return mgr.DeleteSessionValueByToken(ctx, token, k) }},
	} {
		t.Run(api.name, func(t *testing.T) {
			if err := api.set(" key ", nil); err != nil {
				t.Fatal(err)
			}
			if value, ok, err := api.get("key"); err != nil || !ok || value != nil {
				t.Fatalf("stored nil = %v, %v, %v", value, ok, err)
			}
			before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getSessionKey(id), mgr.getTokenKey(token)})
			if err := api.set("key", make(chan int)); !errors.Is(err, derror.ErrSerializeFailed) {
				t.Fatalf("unsupported value error = %v", err)
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			if err := api.del(" key "); err != nil {
				t.Fatal(err)
			}
			if value, ok, err := api.get("key"); err != nil || ok || value != nil {
				t.Fatalf("deleted value = %v, %v, %v", value, ok, err)
			}
			before = captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getSessionKey(id)})
			if err := api.del("missing"); err != nil {
				t.Fatal(err)
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
		})
	}
}
