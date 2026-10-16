package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerAccountCleanupPreservesReusedToken verifies stale account terminals cannot retire another account's current token lifecycle. TestManagerAccountCleanupPreservesReusedToken 验证陈旧账号终端不能清理其他账号当前的 Token 生命周期。
func TestManagerAccountCleanupPreservesReusedToken(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []struct {
		name string
		run  func(*Manager, string) error
	}{
		{"logout by login id", func(m *Manager, loginID string) error { return m.LogoutByLoginID(ctx, loginID) }},
		{"logout by device", func(m *Manager, loginID string) error { return m.LogoutByDevice(ctx, loginID, "web") }},
		{"kickout by login id", func(m *Manager, loginID string) error { return m.KickoutByLoginID(ctx, loginID) }},
		{"replace by login id", func(m *Manager, loginID string) error { return m.ReplaceByLoginID(ctx, loginID) }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			mgr := newTerminalLifecycleTestManager(t, nil)
			const oldLoginID, currentLoginID, token = "stale-terminal-owner", "current-token-owner", "reused-terminal-token"

			oldPair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
				LoginID: oldLoginID, Token: token, Device: "web", DeviceID: "browser",
			}})
			if err != nil {
				t.Fatalf("old login error = %v", err)
			}
			oldSession, err := mgr.getSession(ctx, oldLoginID)
			if err != nil || len(oldSession.TerminalInfos) != 1 {
				t.Fatalf("old session = %+v, %v, want one terminal", oldSession, err)
			}

			// Expire the old canonical lifecycle but deliberately retain its lazily deleted Session entry. 使旧主生命周期过期，但故意保留其待懒删除 Session 条目。
			if err = mgr.storage.Delete(ctx,
				mgr.getTokenKey(token), mgr.getRenewKey(token), mgr.getActiveKey(token),
				mgr.getTokenRefreshKey(token), mgr.getRefreshTokenKey(oldPair.RefreshToken),
			); err != nil {
				t.Fatalf("expire old lifecycle error = %v", err)
			}

			currentPair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
				LoginID: currentLoginID, Token: token, Device: "web", DeviceID: "browser",
			}})
			if err != nil {
				t.Fatalf("current login error = %v", err)
			}
			currentRecord, err := mgr.getTokenRecord(ctx, token)
			if err != nil {
				t.Fatalf("get current record error = %v", err)
			}

			// Equalize all non-account fields so account ownership is the only distinguishing value. 对齐账号之外的所有字段，使账号归属成为唯一差异。
			oldSession.TerminalInfos[0].Device = currentRecord.Device
			oldSession.TerminalInfos[0].DeviceID = currentRecord.DeviceID
			oldSession.TerminalInfos[0].CreateTime = currentRecord.CreateTime
			oldSession.TerminalInfos[0].Index = currentRecord.TerminalIndex
			if err = mgr.saveToStorage(ctx, mgr.getSessionKey(oldLoginID), *oldSession); err != nil {
				t.Fatalf("save stale session error = %v", err)
			}

			keys := terminalLifecycleKeys(mgr, token, currentPair.RefreshToken, mgr.getSessionKey(currentLoginID))
			before := captureTerminalLifecycle(t, mgr, ctx, keys)
			events := registerTerminalLifecycleEvents(mgr)

			if err = operation.run(mgr, oldLoginID); err != nil {
				t.Fatalf("%s error = %v", operation.name, err)
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			if _, err = mgr.getSession(ctx, oldLoginID); !errors.Is(err, derror.ErrSessionNotFound) {
				t.Fatalf("stale session error = %v, want ErrSessionNotFound", err)
			}
			if err = mgr.CheckLogin(ctx, token); err != nil {
				t.Fatalf("current token was retired: %v", err)
			}
			assertNoTerminalRetirementEvent(t, events(), token)
		})
	}
}

// TestManagerTerminalIndexProtectsSameMetadataReuse verifies terminal sequence disambiguates same-account, same-device, same-second reuse. TestManagerTerminalIndexProtectsSameMetadataReuse 验证终端序号可区分同账号、同设备、同秒的 Token 复用。
func TestManagerTerminalIndexProtectsSameMetadataReuse(t *testing.T) {
	ctx := context.Background()

	t.Run("expired terminal cleanup", func(t *testing.T) {
		mgr := newTerminalLifecycleTestManager(t, nil)
		pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
			LoginID: "same-metadata-clean", Token: "same-metadata-clean-token", Device: "web", DeviceID: "browser",
		}})
		if err != nil {
			t.Fatalf("login error = %v", err)
		}
		record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
		if err != nil || record.TerminalIndex <= 0 {
			t.Fatalf("token record = %+v, %v, want terminal index", record, err)
		}
		sess, err := mgr.getSession(ctx, pair.LoginID)
		if err != nil || len(sess.TerminalInfos) != 1 {
			t.Fatalf("session = %+v, %v, want one terminal", sess, err)
		}
		current := sess.TerminalInfos[0]
		stale := current
		stale.Index++
		sess.TerminalInfos = []TerminalInfo{stale, current}
		sess.HistoryTerminalCount = stale.Index
		if err = mgr.saveToStorage(ctx, mgr.getSessionKey(pair.LoginID), *sess); err != nil {
			t.Fatalf("save duplicate terminal fixture error = %v", err)
		}

		before := captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken))
		destroyed, timedOut, err := mgr.cleanExpiredTerminals(ctx, sess)
		if err != nil || destroyed || len(timedOut) != 0 {
			t.Fatalf("cleanExpiredTerminals() = %v, %+v, %v", destroyed, timedOut, err)
		}
		assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
		stored, err := mgr.getSession(ctx, pair.LoginID)
		if err != nil || len(stored.TerminalInfos) != 1 || !reflect.DeepEqual(stored.TerminalInfos[0], current) {
			t.Fatalf("stored terminals = %+v, %v, want only current terminal %+v", stored, err, current)
		}

		shared, err := mgr.getTokenAndShare(ctx, &Session{LoginID: pair.LoginID, TerminalInfos: []TerminalInfo{stale}}, current.Device, current.DeviceID)
		if err != nil || shared != "" {
			t.Fatalf("getTokenAndShare(stale) = %q, %v, want empty", shared, err)
		}
	})

	for _, policy := range []struct {
		name      string
		configure func(*config.Config)
		blocked   listener.Event
	}{
		{
			name: "non-concurrent replacement",
			configure: func(cfg *config.Config) {
				cfg.IsConcurrent = false
				cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
			},
			blocked: listener.EventReplace,
		},
		{
			name: "max-login overflow",
			configure: func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.MaxLoginCount = 1
				cfg.OverflowLogoutMode = config.LogoutModeKickout
			},
			blocked: listener.EventKickout,
		},
	} {
		t.Run(policy.name, func(t *testing.T) {
			mgr := newTerminalLifecycleTestManager(t, policy.configure)
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
				LoginID: "same-metadata-policy", Token: "same-metadata-policy-token", Device: "web", DeviceID: "browser",
			}})
			if err != nil {
				t.Fatalf("current login error = %v", err)
			}
			sess, err := mgr.getSession(ctx, pair.LoginID)
			if err != nil || len(sess.TerminalInfos) != 1 {
				t.Fatalf("current session = %+v, %v", sess, err)
			}
			stale := sess.TerminalInfos[0]
			stale.Index++
			sess.TerminalInfos = []TerminalInfo{stale}
			sess.HistoryTerminalCount = stale.Index
			if err = mgr.saveToStorage(ctx, mgr.getSessionKey(pair.LoginID), *sess); err != nil {
				t.Fatalf("save stale policy session error = %v", err)
			}

			before := captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken))
			events := registerTerminalLifecycleEvents(mgr)
			if _, err = mgr.Login(ctx, pair.LoginID, "mobile", "phone"); err != nil {
				t.Fatalf("replacement login error = %v", err)
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			for _, event := range events() {
				if event.Event == policy.blocked && event.Token == pair.AccessToken {
					t.Fatalf("stale terminal emitted %s for current lifecycle", policy.blocked)
				}
			}
		})
	}
}

// TestManagerDirectRetirementRejectsLifecycleSwap verifies direct operations recheck AccessID after validation. TestManagerDirectRetirementRejectsLifecycleSwap 验证直接下线操作会在校验后复核 AccessID。
func TestManagerDirectRetirementRejectsLifecycleSwap(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []struct {
		name string
		run  func(*Manager, string) error
	}{
		{"logout", func(m *Manager, token string) error { return m.Logout(ctx, token) }},
		{"kickout", func(m *Manager, token string) error { return m.Kickout(ctx, token) }},
		{"replace", func(m *Manager, token string) error { return m.Replace(ctx, token) }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			mgr := newTerminalLifecycleTestManager(t, nil)
			const loginID, token = "direct-lifecycle-swap", "direct-lifecycle-token"
			if _, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: loginID, Token: token, Device: "web", DeviceID: "browser"}); err != nil {
				t.Fatalf("original login error = %v", err)
			}
			expected, err := mgr.getTokenRecord(ctx, token)
			if err != nil {
				t.Fatalf("get original record error = %v", err)
			}

			var before terminalLifecycleSnapshot
			var events func() []*listener.EventData
			swapped := false
			storage := &managerTerminalLifecycleSwapStorage{Storage: mgr.storage, key: mgr.getSessionKey(loginID)}
			storage.afterRead = func() error {
				if err := mgr.Logout(ctx, token); err != nil {
					return err
				}
				pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
					LoginID: loginID, Token: token, Device: "web", DeviceID: "browser",
				}})
				if err != nil {
					return err
				}
				current, err := mgr.getTokenRecord(ctx, token)
				if err != nil {
					return err
				}
				current.CreateTime = expected.CreateTime
				if err = mgr.saveToStorage(ctx, mgr.getTokenKey(token), *current); err != nil {
					return err
				}
				sess, err := mgr.getSession(ctx, loginID)
				if err != nil {
					return err
				}
				sess.TerminalInfos[0].CreateTime = expected.CreateTime
				if err = mgr.saveToStorage(ctx, mgr.getSessionKey(loginID), *sess); err != nil {
					return err
				}
				before = captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, token, pair.RefreshToken, mgr.getSessionKey(loginID)))
				events = registerTerminalLifecycleEvents(mgr)
				swapped = true
				return nil
			}
			mgr.storage = storage

			if err = operation.run(mgr, token); err != nil {
				t.Fatalf("%s error = %v", operation.name, err)
			}
			if !swapped {
				t.Fatal("replacement lifecycle was not installed")
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			if err = mgr.CheckLogin(ctx, token); err != nil {
				t.Fatalf("replacement lifecycle was retired: %v", err)
			}
			assertNoTerminalRetirementEvent(t, events(), token)
		})
	}
}

// TestManagerLegacyTerminalLifecycleFallback verifies old token records remain compatible with terminal cleanup. TestManagerLegacyTerminalLifecycleFallback 验证旧格式 Token 记录仍兼容终端清理。
func TestManagerLegacyTerminalLifecycleFallback(t *testing.T) {
	ctx := context.Background()
	mgr := newTerminalLifecycleTestManager(t, nil)
	token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "legacy-terminal", Token: "legacy-terminal-token", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("login error = %v", err)
	}
	record, err := mgr.getTokenRecord(ctx, token)
	if err != nil {
		t.Fatalf("get current token record error = %v", err)
	}
	if err = mgr.saveToStorage(ctx, mgr.getTokenKey(token), record.TokenInfo); err != nil {
		t.Fatalf("save legacy token record error = %v", err)
	}
	sess, err := mgr.getSession(ctx, record.LoginID)
	if err != nil {
		t.Fatalf("get session error = %v", err)
	}
	destroyed, timedOut, err := mgr.cleanExpiredTerminals(ctx, sess)
	if err != nil || destroyed || len(timedOut) != 0 || len(sess.TerminalInfos) != 1 {
		t.Fatalf("legacy cleanup = %v, %+v, %+v, %v", destroyed, timedOut, sess.TerminalInfos, err)
	}
	if err = mgr.Logout(ctx, token); err != nil {
		t.Fatalf("legacy Logout() error = %v", err)
	}
	if mgr.storage.Exists(ctx, mgr.getTokenKey(token)) {
		t.Fatal("legacy token record remains after logout")
	}
}

type managerTerminalLifecycleSwapStorage struct {
	adapter.Storage
	key       string
	afterRead func() error
}

func (s *managerTerminalLifecycleSwapStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.Storage.Get(ctx, key)
	if err != nil || key != s.key || s.afterRead == nil {
		return value, err
	}
	hook := s.afterRead
	s.afterRead = nil
	if err = hook(); err != nil {
		return nil, err
	}
	return value, nil
}

type terminalLifecycleSnapshot struct {
	values map[string]any
	ttls   map[string]time.Duration
}

func newTerminalLifecycleTestManager(t *testing.T, mutate func(*config.Config)) *Manager {
	t.Helper()
	return newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 600
		cfg.RefreshTokenTimeout = 1200
		cfg.ActiveTimeout = 60
		cfg.RenewInterval = 30
		cfg.IsConcurrent = true
		cfg.IsShare = false
		if mutate != nil {
			mutate(cfg)
		}
	})
}

func terminalLifecycleKeys(mgr *Manager, token, refreshToken string, extra ...string) []string {
	keys := []string{
		mgr.getTokenKey(token), mgr.getRenewKey(token), mgr.getActiveKey(token), mgr.getTokenRefreshKey(token),
	}
	if refreshToken != "" {
		keys = append(keys, mgr.getRefreshTokenKey(refreshToken))
	}
	return append(keys, extra...)
}

func captureTerminalLifecycle(t *testing.T, mgr *Manager, ctx context.Context, keys []string) terminalLifecycleSnapshot {
	t.Helper()
	snapshot := terminalLifecycleSnapshot{values: make(map[string]any, len(keys)), ttls: make(map[string]time.Duration, len(keys))}
	for _, key := range keys {
		value, err := mgr.storage.Get(ctx, key)
		if err != nil {
			t.Fatalf("get lifecycle key %q error = %v", key, err)
		}
		ttl, err := mgr.storage.TTL(ctx, key)
		if err != nil {
			t.Fatalf("get lifecycle TTL %q error = %v", key, err)
		}
		snapshot.values[key] = value
		snapshot.ttls[key] = ttl
	}
	return snapshot
}

func assertTerminalLifecycleUnchanged(t *testing.T, mgr *Manager, ctx context.Context, before terminalLifecycleSnapshot) {
	t.Helper()
	for key, want := range before.values {
		got, err := mgr.storage.Get(ctx, key)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("lifecycle key %q changed: got=%v want=%v error=%v", key, got, want, err)
		}
		gotTTL, err := mgr.storage.TTL(ctx, key)
		wantTTL := before.ttls[key]
		if err != nil || (wantTTL <= 0 && gotTTL != wantTTL) || (wantTTL > 0 && (gotTTL <= 0 || gotTTL > wantTTL+time.Second)) {
			t.Errorf("lifecycle TTL %q changed: got=%v want no extension from %v error=%v", key, gotTTL, wantTTL, err)
		}
	}
}

func registerTerminalLifecycleEvents(mgr *Manager) func() []*listener.EventData {
	var events []*listener.EventData
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		copyData := *data
		events = append(events, &copyData)
	}, listener.ListenerConfig{Async: false})
	return func() []*listener.EventData { return events }
}

func assertNoTerminalRetirementEvent(t *testing.T, events []*listener.EventData, token string) {
	t.Helper()
	for _, event := range events {
		if event.Token != token {
			continue
		}
		switch event.Event {
		case listener.EventLogout, listener.EventKickout, listener.EventReplace, listener.EventActiveTimeout:
			t.Fatalf("replacement lifecycle emitted unexpected %s event", event.Event)
		}
	}
}
