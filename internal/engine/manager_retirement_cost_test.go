package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerRetirementCosts checks public retirement scopes, record reuse and batched cleanup. TestManagerRetirementCosts 检查公开下线作用域、记录复用及批量清理。
func TestManagerRetirementCosts(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"logout", "kickout", "replace"} {
		for _, scope := range []string{"account", "device", "deviceID", "token"} {
			for _, extra := range []bool{false, true} {
				for _, active := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/extra=%v/active=%v", operation, scope, extra, active), func(t *testing.T) {
						mgr := newTestManager(t, func(cfg *config.Config) {
							cfg.IsShare, cfg.IsConcurrent, cfg.MaxLoginCount = false, true, config.NoLimit
							cfg.Timeout, cfg.ActiveTimeout = 600, config.NoLimit
							if active {
								cfg.ActiveTimeout = 60
							}
						})
						const id = "retirement-cost"
						pairs := make([]*RefreshTokenPair, 0, 3)
						extraKeys := make([]string, 0, 3)
						for i, device := range []string{"web", "web", "mobile"} {
							opts := LoginOptions{LoginID: id, Device: device, DeviceID: fmt.Sprint(i)}
							if extra {
								opts.Extra = map[string]any{"tenant": "test"}
							}
							pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: opts})
							if err != nil {
								t.Fatal(err)
							}
							record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
							if err != nil {
								t.Fatal(err)
							}
							pairs = append(pairs, pair)
							extraKeys = append(extraKeys, mgr.getTokenExtraKey(record.AccessID))
						}
						storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int), sessionKey: mgr.getSessionKey(id)}
						codec := &retirementCostCodec{Codec: mgr.serializer}
						mgr.storage, mgr.serializer = storage, codec
						events := registerTerminalLifecycleEvents(mgr)
						run := retirementOperation(mgr, ctx, operation, scope, id, pairs[0].AccessToken)
						if err := run(); err != nil {
							t.Fatal(err)
						}
						wantCount, wantTokenGets, wantSessionGets := 1, 1, 1
						if scope == "account" {
							wantCount = 3
						} else if scope == "device" {
							wantCount = 2
						} else if scope == "token" {
							// Capture, validation and lock-entry recheck remain outside the reused cleanup read. 捕获、校验及入锁复核仍独立于复用的清理读取。
							wantTokenGets = 4
						}
						if storage.gets[mgr.getSessionKey(id)] != wantSessionGets || codec.decodes != wantSessionGets+wantCount {
							t.Fatalf("session reads/total decodes = %d/%d, want %d/%d", storage.gets[mgr.getSessionKey(id)], codec.decodes, wantSessionGets, wantSessionGets+wantCount)
						}
						metadataBatches, metadataKeys, sessionWrites := 0, 0, storage.sessionWrites
						for _, keys := range storage.deletes {
							batchMetadata := 0
							for _, key := range keys {
								if key == mgr.getSessionKey(id) {
									sessionWrites++
								}
								for _, pair := range pairs {
									if key == mgr.getRenewKey(pair.AccessToken) || key == mgr.getActiveKey(pair.AccessToken) {
										batchMetadata++
									}
								}
							}
							if batchMetadata > 0 {
								metadataBatches++
								metadataKeys += batchMetadata
							}
						}
						if metadataBatches != 1 || metadataKeys != 2*wantCount || sessionWrites != 1 {
							t.Fatalf("metadata batches/keys, session writes = %d/%d/%d", metadataBatches, metadataKeys, sessionWrites)
						}
						for i, pair := range pairs {
							wantGets := 0
							if i < wantCount {
								wantGets = wantTokenGets
							}
							if storage.gets[mgr.getTokenKey(pair.AccessToken)] != wantGets || storage.gets[extraKeys[i]] != 0 {
								t.Fatalf("token %d reads/extra reads = %d/%d", i, storage.gets[mgr.getTokenKey(pair.AccessToken)], storage.gets[extraKeys[i]])
							}
						}
						wantErr := derror.ErrInvalidToken
						wantEvent := listener.EventLogout
						if operation == "kickout" {
							wantErr, wantEvent = derror.ErrTokenKickout, listener.EventKickout
						} else if operation == "replace" {
							wantErr, wantEvent = derror.ErrTokenReplaced, listener.EventReplace
						}
						for i, pair := range pairs {
							_, err := mgr.getTokenRecord(ctx, pair.AccessToken)
							if i >= wantCount {
								if err != nil || !storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
									t.Fatal("unselected terminal was retired")
								}
								continue
							}
							if !errors.Is(err, wantErr) {
								t.Fatalf("retired token = %v, want %v", err, wantErr)
							}
							for _, key := range []string{extraKeys[i], mgr.getRenewKey(pair.AccessToken), mgr.getActiveKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)} {
								if storage.Exists(ctx, key) {
									t.Fatalf("retained cleanup key %s", key)
								}
							}
						}
						count := 0
						for _, event := range events() {
							if event.Event == wantEvent {
								count++
							}
							if event.Event == listener.EventActiveTimeout {
								t.Fatal("unexpected timeout event")
							}
						}
						if count != wantCount {
							t.Fatalf("retirement events = %d, want %d", count, wantCount)
						}
						wantEvents := wantCount
						if scope == "account" {
							wantEvents++
						}
						if err := run(); err != nil || len(events()) != wantEvents {
							t.Fatalf("repeat retirement = %v, events=%d", err, len(events()))
						}
					})
				}
			}
		}
	}
}

// TestManagerRetirementStateBoundaries preserves expiry, detached-session and TTL fallback behavior. TestManagerRetirementStateBoundaries 保留失效、脱离会话及 TTL 回退行为。
func TestManagerRetirementStateBoundaries(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"kickout", "replace"} {
		for _, state := range []string{"finite", "permanent", "TTL failure", "missing activity", "revoked", "expired", "detached"} {
			t.Run(operation+"/"+state, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.Timeout, cfg.ActiveTimeout = 600, 60
					if state == "permanent" {
						cfg.Timeout = config.NoLimit
					}
				})
				pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
					LoginID: "retirement-boundary", Device: "web", DeviceID: "0", Extra: map[string]any{"tenant": "test"},
				}})
				if err != nil {
					t.Fatal(err)
				}
				record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
				if err != nil {
					t.Fatal(err)
				}
				scope := "account"
				switch state {
				case "missing activity":
					err = mgr.storage.Delete(ctx, mgr.getActiveKey(pair.AccessToken))
				case "expired":
					err = mgr.storage.Delete(ctx, mgr.getTokenKey(pair.AccessToken), mgr.getTokenExtraKey(record.AccessID))
				case "revoked":
					record.Revoked = true
					err = mgr.storage.Set(ctx, mgr.getTokenKey(pair.AccessToken), encodeTokenRecord(record), 10*time.Minute)
				case "detached":
					err = mgr.storage.Delete(ctx, mgr.getSessionKey(pair.LoginID))
					scope = "token"
				}
				if err != nil {
					t.Fatal(err)
				}
				storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int), failTTL: state == "TTL failure"}
				mgr.storage = storage
				events := registerTerminalLifecycleEvents(mgr)
				if err = retirementOperation(mgr, ctx, operation, scope, pair.LoginID, pair.AccessToken)(); err != nil {
					t.Fatal(err)
				}
				wantGets := 1
				if state == "detached" {
					wantGets = 4
				}
				if storage.gets[mgr.getTokenKey(pair.AccessToken)] != wantGets {
					t.Fatalf("primary GETs = %d, want %d", storage.gets[mgr.getTokenKey(pair.AccessToken)], wantGets)
				}
				wantErr, wantEvent := derror.ErrTokenKickout, listener.EventKickout
				if operation == "replace" {
					wantErr, wantEvent = derror.ErrTokenReplaced, listener.EventReplace
				}
				transitioned := state != "missing activity" && state != "expired" && state != "revoked"
				if !transitioned {
					wantErr = derror.ErrInvalidToken
				}
				if _, err = mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(err, wantErr) {
					t.Fatalf("result = %v, want %v", err, wantErr)
				}
				if transitioned {
					ttl, ttlErr := storage.Storage.TTL(ctx, mgr.getTokenKey(pair.AccessToken))
					if ttlErr != nil || state == "permanent" && ttl != adapter.TTLNoExpire || state != "permanent" && (ttl <= 0 || ttl > 10*time.Minute) {
						t.Fatalf("state TTL = %v, %v", ttl, ttlErr)
					}
				}
				for _, key := range []string{mgr.getSessionKey(pair.LoginID), mgr.getTokenExtraKey(record.AccessID), mgr.getActiveKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)} {
					if storage.Exists(ctx, key) {
						t.Fatalf("retained cleanup key %s", key)
					}
				}
				count := 0
				for _, event := range events() {
					if event.Event == wantEvent {
						count++
					}
				}
				if transitioned && count != 1 || !transitioned && count != 0 {
					t.Fatalf("state events = %d, transitioned=%v", count, transitioned)
				}
			})
		}
	}
}

// TestManagerRetirementFailures retains the Session index until sequential cleanup succeeds. TestManagerRetirementFailures 在顺序清理成功前保留 Session 索引。
func TestManagerRetirementFailures(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"logout", "kickout", "replace"} {
		for _, failure := range []string{"credential", "activity", "refresh"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) { cfg.IsShare = false })
				var pairs []*RefreshTokenPair
				for _, deviceID := range []string{"0", "1"} {
					pair, err := mgr.LoginWithRefreshToken(ctx, "retirement-failure", "web", deviceID)
					if err != nil {
						t.Fatal(err)
					}
					pairs = append(pairs, pair)
				}
				storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int)}
				switch failure {
				case "credential":
					if operation == "logout" {
						storage.failDeleteKey = mgr.getTokenKey(pairs[1].AccessToken)
					} else {
						storage.failSetKey = mgr.getTokenKey(pairs[1].AccessToken)
					}
				case "activity":
					storage.failDeleteKey = mgr.getActiveKey(pairs[0].AccessToken)
				case "refresh":
					storage.failDeleteKey = mgr.getRefreshTokenKey(pairs[0].RefreshToken)
				}
				mgr.storage = storage
				events := registerTerminalLifecycleEvents(mgr)
				run := retirementOperation(mgr, ctx, operation, "account", pairs[0].LoginID, pairs[0].AccessToken)
				if err := run(); !errors.Is(err, derror.ErrStorageUnavailable) || len(events()) != 0 {
					t.Fatalf("failed retirement = %v, events=%d", err, len(events()))
				}
				sess, err := mgr.getSession(ctx, pairs[0].LoginID)
				if err != nil || len(sess.TerminalInfos) != 2 {
					t.Fatalf("failure lost cleanup index: %+v, %v", sess, err)
				}
				storage.failDeleteKey, storage.failSetKey = "", ""
				if err := run(); err != nil {
					t.Fatalf("retry = %v", err)
				}
				if storage.Exists(ctx, mgr.getSessionKey(pairs[0].LoginID)) {
					t.Fatal("retry retained Session")
				}
				for _, pair := range pairs {
					if storage.Exists(ctx, mgr.getTokenRefreshKey(pair.AccessToken)) || storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
						t.Fatal("retry retained refresh credential")
					}
				}
			})
		}
	}
}

// TestManagerDirectRetirementSessionBoundaries checks deferred Session loading, timeout causes and failure isolation. TestManagerDirectRetirementSessionBoundaries 验证延后加载 Session、超时原因及失败隔离。
func TestManagerDirectRetirementSessionBoundaries(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"logout", "kickout", "replace"} {
		for _, state := range []string{"missing", "detached", "corrupt", "foreign", "read failure", "timeout"} {
			t.Run(operation+"/"+state, func(t *testing.T) {
				mgr := newTerminalLifecycleTestManager(t, nil)
				pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
					LoginID: "direct-session", Extra: map[string]any{"source": "test"},
				}})
				if err != nil {
					t.Fatal(err)
				}
				record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
				if err != nil {
					t.Fatal(err)
				}
				sessionKey := mgr.getSessionKey(pair.LoginID)
				var wantErr error
				switch state {
				case "missing":
					err = mgr.storage.Delete(ctx, sessionKey)
				case "detached", "foreign":
					var sess *Session
					sess, err = mgr.getSession(ctx, pair.LoginID)
					if err != nil {
						t.Fatal(err)
					}
					if state == "detached" {
						sess.TerminalInfos = nil
					} else {
						sess.AuthType = "another-auth-system"
						wantErr = derror.ErrInvalidToken
					}
					err = mgr.saveToStorage(ctx, sessionKey, *sess)
				case "corrupt":
					err = mgr.storage.Set(ctx, sessionKey, []byte("{broken"), time.Hour)
					wantErr = derror.ErrSerializeFailed
				case "read failure":
					wantErr = derror.ErrStorageUnavailable
				case "timeout":
					err = mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour)
				}
				if err != nil {
					t.Fatal(err)
				}
				keys := terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken, sessionKey, mgr.getTokenExtraKey(record.AccessID), mgr.getRefreshTokenKey(pair.RefreshToken))
				before := captureTerminalLifecycle(t, mgr, ctx, keys)
				storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int), sessionKey: sessionKey}
				if state == "read failure" {
					storage.failGetKey = sessionKey
				}
				mgr.storage = storage
				pool := &managerQueuedMaintenancePool{}
				mgr.pool = pool
				t.Cleanup(pool.runAll)
				events := registerTerminalLifecycleEvents(mgr)
				err = retirementOperation(mgr, ctx, operation, "token", pair.LoginID, pair.AccessToken)()
				if !errors.Is(err, wantErr) || pool.taskCount() != 0 {
					t.Fatalf("retirement = %v, want %v; maintenance = %d", err, wantErr, pool.taskCount())
				}
				wantSessionGets := 1
				if state == "timeout" {
					// A suspected timeout still loads context and rechecks it under the account lock. 疑似超时仍加载上下文，并在账号锁内重新核对。
					wantSessionGets = 2
				}
				if storage.gets[sessionKey] != wantSessionGets {
					t.Fatalf("Session reads = %d, want %d", storage.gets[sessionKey], wantSessionGets)
				}
				storage.failGetKey = ""
				if wantErr != nil {
					assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
					if len(events()) != 0 {
						t.Fatal("failed Session validation emitted retirement events")
					}
					return
				}
				wantState, wantEvent := derror.ErrInvalidToken, listener.EventLogout
				switch {
				case state == "timeout":
					wantState, wantEvent = derror.ErrActiveTimeout, listener.EventActiveTimeout
				case operation == "kickout":
					wantState, wantEvent = derror.ErrTokenKickout, listener.EventKickout
				case operation == "replace":
					wantState, wantEvent = derror.ErrTokenReplaced, listener.EventReplace
				}
				if _, err = mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(err, wantState) {
					t.Fatalf("retired state = %v, want %v", err, wantState)
				}
				for _, key := range keys {
					if key == mgr.getTokenKey(pair.AccessToken) || key == sessionKey {
						continue
					}
					if storage.Exists(ctx, key) {
						t.Fatalf("retained metadata %s", key)
					}
				}
				gotEvents := events()
				if state == "timeout" {
					if len(gotEvents) != 2 || gotEvents[0].Event != listener.EventDestroySession {
						t.Fatalf("timeout events = %+v", gotEvents)
					}
					gotEvents = gotEvents[1:]
				}
				if len(gotEvents) != 1 || gotEvents[0].Event != wantEvent {
					t.Fatalf("retirement events = %+v, want %s", gotEvents, wantEvent)
				}
				if state == "detached" {
					assertTerminalLifecycleUnchanged(t, mgr, ctx, terminalLifecycleSnapshot{
						values: map[string]any{sessionKey: before.values[sessionKey]},
						ttls:   map[string]time.Duration{sessionKey: before.ttls[sessionKey]},
					})
				} else if storage.Exists(ctx, sessionKey) {
					t.Fatal("missing or destroyed Session was recreated")
				}
			})
		}
	}
}

// retirementOperation selects the public operation without bypassing validation. retirementOperation 选择公开操作，不绕过校验。
func retirementOperation(m *Manager, ctx context.Context, operation, scope, id, token string) func() error {
	byScope := map[string][]func() error{
		"account":  {func() error { return m.LogoutByLoginID(ctx, id) }, func() error { return m.KickoutByLoginID(ctx, id) }, func() error { return m.ReplaceByLoginID(ctx, id) }},
		"device":   {func() error { return m.LogoutByDevice(ctx, id, "web") }, func() error { return m.KickoutByDevice(ctx, id, "web") }, func() error { return m.ReplaceByDevice(ctx, id, "web") }},
		"deviceID": {func() error { return m.LogoutByDeviceAndDeviceID(ctx, id, "web", "0") }, func() error { return m.KickoutByDeviceAndDeviceID(ctx, id, "web", "0") }, func() error { return m.ReplaceByDeviceAndDeviceID(ctx, id, "web", "0") }},
		"token":    {func() error { return m.Logout(ctx, token) }, func() error { return m.Kickout(ctx, token) }, func() error { return m.Replace(ctx, token) }},
	}
	index := map[string]int{"logout": 0, "kickout": 1, "replace": 2}[operation]
	return byScope[scope][index]
}

// retirementCostStorage records calls and injects failures without requiring optional storage capabilities. retirementCostStorage 记录调用并注入故障，不要求可选存储能力。
type retirementCostStorage struct {
	adapter.Storage
	gets          map[string]int
	deletes       [][]string
	sessionKey    string
	sessionWrites int
	failGetKey    string
	failDeleteKey string
	failSetKey    string
	failTTL       bool
}

func (s *retirementCostStorage) Get(ctx context.Context, key string) (any, error) {
	s.gets[key]++
	if key == s.failGetKey {
		return nil, errors.New("injected read failure")
	}
	return s.Storage.Get(ctx, key)
}

func (s *retirementCostStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if key == s.failSetKey {
		return errors.New("injected state write failure")
	}
	if key == s.sessionKey {
		s.sessionWrites++
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

func (s *retirementCostStorage) Delete(ctx context.Context, keys ...string) error {
	s.deletes = append(s.deletes, append([]string(nil), keys...))
	for _, key := range keys {
		if key == s.failDeleteKey {
			return errors.New("injected cleanup failure")
		}
	}
	return s.Storage.Delete(ctx, keys...)
}

func (s *retirementCostStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if s.failTTL {
		return 0, errors.New("injected TTL failure")
	}
	return s.Storage.TTL(ctx, key)
}

// retirementCostCodec counts decoding across the complete operation. retirementCostCodec 统计完整操作的解码次数。
type retirementCostCodec struct {
	adapter.Codec
	decodes int
}

func (c *retirementCostCodec) Decode(data []byte, target any) error {
	c.decodes++
	return c.Codec.Decode(data, target)
}
