package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestManagerScopedTerminalQueries checks scope, order and read-only liveness filtering at public entry points. TestManagerScopedTerminalQueries 从公开入口检查作用域、顺序及只读的存活筛选。
func TestManagerScopedTerminalQueries(t *testing.T) {
	ctx := context.Background()
	for _, scope := range []string{"account", "device", "deviceID", "no match"} {
		for _, query := range []string{"raw", "alive", "count"} {
			t.Run(scope+"/"+query, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsShare, cfg.IsConcurrent, cfg.MaxLoginCount = false, true, config.NoLimit
					cfg.Timeout, cfg.ActiveTimeout = 600, 60
				})
				const id = "terminal-query-cost"
				entries := []struct{ device, deviceID, state string }{
					{"web", "a", "alive"}, {"mobile", "a", "alive"}, {"web", "b", "alive"},
					{"web", "a", "expired"}, {"web", "a", "stale"}, {"web", "a", "inactive"},
					{"web", "blocked", "disabled"}, {"mobile", "a", "expired"},
				}
				tokens, extraKeys := make([]string, len(entries)), make([]string, len(entries))
				for i, entry := range entries {
					token, err := mgr.LoginWithOptions(ctx, LoginOptions{
						LoginID: id, Device: entry.device, DeviceID: entry.deviceID, Extra: map[string]any{"detail": "unused"},
					})
					if err != nil {
						t.Fatal(err)
					}
					record, err := mgr.getTokenRecord(ctx, token)
					if err != nil {
						t.Fatal(err)
					}
					tokens[i], extraKeys[i] = token, mgr.getTokenExtraKey(record.AccessID)
				}
				sess, err := mgr.getSession(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				for i, entry := range entries {
					switch entry.state {
					case "expired":
						err = mgr.storage.Delete(ctx, mgr.getTokenKey(tokens[i]))
					case "stale":
						sess.TerminalInfos[i].Index++
					case "inactive":
						err = mgr.storage.Set(ctx, mgr.getActiveKey(tokens[i]), time.Now().Add(-2*time.Minute).Unix(), time.Hour)
					case "disabled":
						err = mgr.DisableDeviceAndDeviceID(ctx, id, entry.device, entry.deviceID, time.Hour)
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if err = mgr.saveToStorage(ctx, mgr.getSessionKey(id), *sess); err != nil {
					t.Fatal(err)
				}
				base := mgr.storage
				before, err := base.Get(ctx, mgr.getSessionKey(id))
				if err != nil {
					t.Fatal(err)
				}
				storage := &retirementCostStorage{Storage: base, gets: make(map[string]int), sessionKey: mgr.getSessionKey(id)}
				mgr.storage = storage
				pool := &managerQueuedMaintenancePool{}
				mgr.pool = pool
				t.Cleanup(pool.runAll)
				var got []string
				var count int
				device := "web"
				if scope == "no match" {
					device = "absent"
				}
				switch scope {
				case "account":
					if query == "count" {
						count, err = mgr.GetOnlineTerminalCount(ctx, id)
					} else {
						got, err = mgr.GetTokenValueListByLoginID(ctx, id, query == "alive")
					}
				case "device", "no match":
					if query == "count" {
						count, err = mgr.GetOnlineTerminalCountByDevice(ctx, id, " "+device+" ")
					} else {
						got, err = mgr.GetTokenValueListByDevice(ctx, id, " "+device+" ", query == "alive")
					}
				case "deviceID":
					if query == "count" {
						count, err = mgr.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, id, " web ", " a ")
					} else {
						got, err = mgr.GetTokenValueListByDeviceAndDeviceID(ctx, id, " web ", " a ", query == "alive")
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				want := []string{}
				for i, entry := range entries {
					matched := scope == "account" || entry.device == device && (scope != "deviceID" || entry.deviceID == "a")
					if matched && (query == "raw" || entry.state == "alive") {
						want = append(want, tokens[i])
					}
					reads, activeReads := 0, 0
					if matched && query != "raw" {
						reads = 1
						if entry.state == "alive" || entry.state == "inactive" {
							activeReads = 1
						}
					}
					if storage.gets[mgr.getTokenKey(tokens[i])] != reads || storage.gets[mgr.getActiveKey(tokens[i])] != activeReads || storage.gets[extraKeys[i]] != 0 {
						t.Fatalf("entry %d: token/activity/Extra reads = %d/%d/%d, want %d/%d/0", i, storage.gets[mgr.getTokenKey(tokens[i])], storage.gets[mgr.getActiveKey(tokens[i])], storage.gets[extraKeys[i]], reads, activeReads)
					}
				}
				if query == "count" && count != len(want) || query != "count" && !reflect.DeepEqual(got, want) {
					t.Fatalf("list/count = %v/%d, want %v/%d", got, count, want, len(want))
				}
				wantDisableReads := 1
				if query == "raw" || scope == "no match" {
					wantDisableReads = 0
				}
				if storage.gets[mgr.getDisableKey(id)] != wantDisableReads || storage.gets[mgr.getDisableDeviceKey(id, "web")] != wantDisableReads {
					t.Fatal("account/device restriction reads were not reused")
				}
				after, err := base.Get(ctx, mgr.getSessionKey(id))
				if err != nil || !reflect.DeepEqual(before, after) || storage.gets[mgr.getSessionKey(id)] != 1 || storage.sessionWrites != 0 || len(storage.deletes) != 0 || pool.taskCount() != 0 {
					t.Fatal("query rewrote Session, cleaned stale entries, scheduled maintenance or reloaded Session")
				}
			})
		}
	}
}

// terminalQueryFailureStorage fails one key to verify early exit and scope isolation. terminalQueryFailureStorage 使单个键读取失败，以检查提前结束及作用域隔离。
type terminalQueryFailureStorage struct {
	adapter.Storage
	key string
}

// Get injects a failure only for the selected key. Get 仅为指定键注入故障。
func (s *terminalQueryFailureStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.key {
		return nil, errors.New("query read unavailable")
	}
	return s.Storage.Get(ctx, key)
}

// TestManagerLatestTokenQueryStopsAtMatch checks newest-first search and failure propagation without inspecting older candidates. TestManagerLatestTokenQueryStopsAtMatch 检查从新到旧查询及错误传播，命中后不再检查旧候选。
func TestManagerLatestTokenQueryStopsAtMatch(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"newest alive", "newest expired", "newest stale", "newest read failure", "no match"} {
		t.Run(state, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsShare, cfg.IsConcurrent, cfg.MaxLoginCount = false, true, config.NoLimit
			})
			const id = "latest-query-cost"
			var tokens []string
			for _, device := range []string{"web", "web", "mobile", "web", "mobile"} {
				token, err := mgr.Login(ctx, id, device)
				if err != nil {
					t.Fatal(err)
				}
				tokens = append(tokens, token)
			}
			sess, err := mgr.getSession(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			want, device := tokens[3], " web "
			var wantErr error
			switch state {
			case "newest expired":
				err = mgr.storage.Delete(ctx, mgr.getTokenKey(tokens[3]))
				want = tokens[1]
			case "newest stale":
				sess.TerminalInfos[3].Index++
				err = mgr.saveToStorage(ctx, mgr.getSessionKey(id), *sess)
				want = tokens[1]
			case "newest read failure":
				want, wantErr = "", derror.ErrStorageUnavailable
			case "no match":
				want, wantErr, device = "", derror.ErrInvalidToken, "absent"
			}
			if err != nil {
				t.Fatal(err)
			}
			failKey := mgr.getTokenKey(tokens[0])
			if state == "newest read failure" {
				failKey = mgr.getTokenKey(tokens[3])
			}
			storage := &retirementCostStorage{
				Storage: &terminalQueryFailureStorage{Storage: mgr.storage, key: failKey},
				gets:    make(map[string]int), sessionKey: mgr.getSessionKey(id),
			}
			mgr.storage = storage
			got, err := mgr.GetTokenValueByLoginID(ctx, id, device)
			if got != want || !errors.Is(err, wantErr) {
				t.Fatalf("latest = %q, %v, want %q, %v", got, err, want, wantErr)
			}
			for i, token := range tokens {
				reads := 0
				if i == 3 && state != "no match" || i == 1 && (state == "newest expired" || state == "newest stale") {
					reads = 1
				}
				if storage.gets[mgr.getTokenKey(token)] != reads {
					t.Fatalf("token %d reads = %d, want %d", i, storage.gets[mgr.getTokenKey(token)], reads)
				}
			}
			if storage.gets[mgr.getSessionKey(id)] != 1 || storage.sessionWrites != 0 || len(storage.deletes) != 0 {
				t.Fatal("latest query loaded Session twice or performed cleanup")
			}
		})
	}
}

// BenchmarkScopedTerminalQueries measures traversal allocations separately from Session decoding. BenchmarkScopedTerminalQueries 将遍历分配与 Session 解码分开度量。
func BenchmarkScopedTerminalQueries(b *testing.B) {
	ctx := context.Background()
	for _, size := range []int{100, 10000} {
		sess := &Session{LoginID: "query-benchmark", TerminalInfos: make([]TerminalInfo, size)}
		mgr := NewManager(config.DefaultConfig(), nil, newManagerTestStorage(), nil, nil, nil, nil)
		b.Cleanup(mgr.CloseManager)
		for i := range sess.TerminalInfos {
			device := "mobile"
			if i%10 == 0 {
				device = "web"
			}
			sess.TerminalInfos[i] = TerminalInfo{LoginID: sess.LoginID, Device: device, DeviceID: "a", Token: fmt.Sprint(i)}
			// Half of the selected entries have expired primary keys. 选中条目中一半的主键已过期。
			if i%20 == 0 {
				record := &tokenRecord{TokenInfo: TokenInfo{
					AuthType: mgr.config.AuthType, LoginID: sess.LoginID, Device: device, DeviceID: "a",
					Timeout: config.NoLimit, ActiveTimeout: config.NoLimit,
				}, AccessID: fmt.Sprint(i)}
				if err := mgr.storage.Set(ctx, mgr.getTokenKey(sess.TerminalInfos[i].Token), encodeTokenRecord(record), 0); err != nil {
					b.Fatal(err)
				}
			}
		}
		for _, query := range []string{"raw", "alive", "count", "no match"} {
			b.Run(fmt.Sprintf("%s/%d", query, size), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if query == "count" {
						if count, err := mgr.countAliveTokens(ctx, sess, "web", "a"); err != nil || count != size/20 {
							b.Fatal(count, err)
						}
						continue
					}
					device, want := "web", 0
					if query == "raw" {
						want = size / 10
					} else if query == "alive" {
						want = size / 20
					} else if query == "no match" {
						device = "absent"
					}
					got, err := mgr.filterTokens(ctx, sess, query == "alive", device, "a")
					if err != nil || len(got) != want {
						b.Fatal(len(got), err)
					}
				}
			})
		}
	}
}
