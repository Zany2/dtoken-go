package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerScopedPruningCosts checks record reuse and distinguishes natural expiry from explicit retirement. TestManagerScopedPruningCosts 检查记录复用，并区分自然过期与主动下线。
func TestManagerScopedPruningCosts(t *testing.T) {
	ctx := context.Background()
	for _, state := range []string{"alive", "expired", "missing activity", "corrupt activity", "negative activity", "timeout", "kickout", "replaced", "logout marker", "timeout marker", "foreign namespace", "foreign refresh", "malformed"} {
		t.Run(state, func(t *testing.T) {
			mgr := newTerminalLifecycleTestManager(t, nil)
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
				LoginID: "prune-cost", Device: "web", Extra: map[string]any{"tenant": "test"},
			}})
			if err != nil {
				t.Fatal(err)
			}
			other, err := mgr.Login(ctx, pair.LoginID, "mobile")
			if err != nil {
				t.Fatal(err)
			}
			record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
			if err != nil {
				t.Fatal(err)
			}
			primary, extra := mgr.getTokenKey(pair.AccessToken), mgr.getTokenExtraKey(record.AccessID)
			switch state {
			case "expired":
				err = mgr.storage.Delete(ctx, primary, extra)
			case "missing activity":
				err = mgr.storage.Delete(ctx, mgr.getActiveKey(pair.AccessToken))
			case "corrupt activity":
				err = mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), "invalid", time.Hour)
			case "negative activity":
				err = mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), int64(-1), time.Hour)
			case "timeout":
				err = mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour)
			case "kickout", "foreign refresh":
				err = mgr.storage.Set(ctx, primary, string(TokenStateKickOut), time.Hour)
			case "replaced":
				err = mgr.storage.Set(ctx, primary, string(TokenStateReplaced), time.Hour)
			case "logout marker":
				err = mgr.storage.Set(ctx, primary, []byte(TokenStateLogout), time.Hour)
			case "timeout marker":
				err = mgr.storage.Set(ctx, primary, string(TokenStateActiveTimeout), time.Hour)
			case "foreign namespace":
				record.AuthType += "-other"
				err = mgr.storage.Set(ctx, primary, encodeTokenRecord(record), time.Hour)
			case "malformed":
				err = mgr.storage.Set(ctx, primary, "{", time.Hour)
			}
			if err != nil {
				t.Fatal(err)
			}
			if state == "foreign refresh" {
				info, err := mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
				if err != nil {
					t.Fatal(err)
				}
				info.LoginID = "another-account"
				if err := mgr.saveToStorage(ctx, mgr.getRefreshTokenKey(pair.RefreshToken), info); err != nil {
					t.Fatal(err)
				}
			}
			sess, err := mgr.getSession(ctx, pair.LoginID)
			if err != nil {
				t.Fatal(err)
			}
			before := captureTerminalLifecycle(t, mgr, ctx, append(terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken), extra))
			original := append([]TerminalInfo(nil), sess.TerminalInfos...)
			storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int)}
			codec := &retirementCostCodec{Codec: mgr.serializer}
			mgr.storage, mgr.serializer = storage, codec
			unlock := mgr.lockLoginWrite(pair.LoginID)
			destroyed, timedOut, err := mgr.pruneExpiredTerminals(ctx, sess, func(terminal TerminalInfo) bool { return terminal.Device == "web" })
			unlock()
			if err != nil || destroyed {
				t.Fatalf("scoped pruning = %v, %v", destroyed, err)
			}
			if storage.gets[primary] != 1 || storage.gets[extra] != 0 || storage.gets[mgr.getTokenKey(other)] != 0 || storage.gets[mgr.getSessionKey(pair.LoginID)] != 0 {
				t.Fatalf("unexpected pruning reads: %+v", storage.gets)
			}
			wantDecodes := 0
			switch state {
			case "missing activity", "corrupt activity", "negative activity", "timeout", "foreign refresh":
				wantDecodes = 1
			case "kickout", "replaced", "logout marker", "timeout marker":
				wantDecodes = 2
			}
			if codec.decodes != wantDecodes {
				t.Fatalf("pruning decodes = %d, want %d", codec.decodes, wantDecodes)
			}
			if state == "alive" {
				if !reflect.DeepEqual(sess.TerminalInfos, original) {
					t.Fatal("pruning changed live terminals")
				}
			} else if len(sess.TerminalInfos) != 1 || sess.TerminalInfos[0].Token != other {
				t.Fatalf("pruned terminals = %+v", sess.TerminalInfos)
			}
			if state == "timeout" && (len(timedOut) != 1 || timedOut[0].Token != pair.AccessToken) || state != "timeout" && len(timedOut) != 0 {
				t.Fatalf("timeout events = %+v", timedOut)
			}
			preserved := state == "alive" || state == "foreign namespace" || state == "foreign refresh" || state == "malformed"
			if preserved {
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			} else {
				if storage.Exists(ctx, mgr.getActiveKey(pair.AccessToken)) || storage.Exists(ctx, mgr.getRenewKey(pair.AccessToken)) {
					t.Fatal("pruning retained access-only metadata")
				}
				wantRefresh := state == "expired"
				if storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) != wantRefresh || storage.Exists(ctx, mgr.getTokenRefreshKey(pair.AccessToken)) != wantRefresh {
					t.Fatal("pruning changed refresh lifetime incorrectly")
				}
				if state == "missing activity" || state == "corrupt activity" || state == "negative activity" || state == "timeout" {
					if storage.Exists(ctx, extra) {
						t.Fatal("retirement retained Extra")
					}
				}
				if state == "timeout" {
					if _, err := mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(err, derror.ErrActiveTimeout) {
						t.Fatalf("timeout state = %v", err)
					}
				}
			}
			stored, err := mgr.getSession(ctx, pair.LoginID)
			if err != nil || !reflect.DeepEqual(stored.TerminalInfos, original) {
				t.Fatal("pruning persisted Session before the login boundary")
			}
		})
	}
}

// TestManagerTimeoutRetirementCosts checks authentication and refresh entry points after inactivity is confirmed. TestManagerTimeoutRetirementCosts 检查鉴权与刷新入口确认不活跃超时后的清理成本。
func TestManagerTimeoutRetirementCosts(t *testing.T) {
	ctx := context.Background()
	for _, entry := range []string{"check", "refresh"} {
		t.Run(entry, func(t *testing.T) {
			mgr := newTerminalLifecycleTestManager(t, nil)
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
				LoginID: "timeout-cost", Device: "web", Extra: map[string]any{"tenant": "test"},
			}})
			if err != nil {
				t.Fatal(err)
			}
			record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
				t.Fatal(err)
			}
			storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int)}
			mgr.storage = storage
			events := registerTerminalLifecycleEvents(mgr)
			wantGets := 3
			if entry == "check" {
				err = mgr.CheckLogin(ctx, pair.AccessToken)
			} else {
				_, err = mgr.RefreshToken(ctx, pair.RefreshToken)
				wantGets = 2
			}
			if !errors.Is(err, derror.ErrActiveTimeout) || storage.gets[mgr.getTokenKey(pair.AccessToken)] != wantGets || storage.gets[mgr.getTokenExtraKey(record.AccessID)] != 0 {
				t.Fatalf("timeout = %v, primary reads=%d, want %d", err, storage.gets[mgr.getTokenKey(pair.AccessToken)], wantGets)
			}
			got := events()
			if len(got) != 2 || got[0].Event != listener.EventDestroySession || got[1].Event != listener.EventActiveTimeout {
				t.Fatalf("timeout events = %+v", got)
			}
			for _, key := range []string{mgr.getTokenExtraKey(record.AccessID), mgr.getSessionKey(pair.LoginID), mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken)} {
				if storage.Exists(ctx, key) {
					t.Fatalf("timeout retained %s", key)
				}
			}
		})
	}
}

// TestManagerPruningFailuresRetainIndex keeps retry metadata when credential or auxiliary cleanup fails. TestManagerPruningFailuresRetainIndex 在凭据或附属清理失败时保留重试索引。
func TestManagerPruningFailuresRetainIndex(t *testing.T) {
	ctx := context.Background()
	for _, failure := range []string{"state", "metadata", "read"} {
		t.Run(failure, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
			token, err := mgr.Login(ctx, "pruning-failure", "web")
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.storage.Set(ctx, mgr.getActiveKey(token), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
				t.Fatal(err)
			}
			sess, err := mgr.getSession(ctx, "pruning-failure")
			if err != nil {
				t.Fatal(err)
			}
			original := append([]TerminalInfo(nil), sess.TerminalInfos...)
			storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int)}
			mgr.storage = storage
			switch failure {
			case "state":
				storage.failSetKey = mgr.getTokenKey(token)
			case "metadata":
				storage.failDeleteKey = mgr.getActiveKey(token)
			case "read":
				mgr.storage = &managerSessionReadErrorStorage{Storage: storage, key: mgr.getTokenKey(token)}
			}
			unlock := mgr.lockLoginWrite(sess.LoginID)
			_, _, err = mgr.pruneExpiredTerminals(ctx, sess, nil)
			unlock()
			if !errors.Is(err, derror.ErrStorageUnavailable) || !reflect.DeepEqual(sess.TerminalInfos, original) {
				t.Fatalf("failed pruning = %v, terminals=%+v", err, sess.TerminalInfos)
			}
			storage.failSetKey, storage.failDeleteKey = "", ""
			mgr.storage = storage
			unlock = mgr.lockLoginWrite(sess.LoginID)
			destroyed, _, err := mgr.pruneExpiredTerminals(ctx, sess, nil)
			unlock()
			if err != nil || !destroyed || len(sess.TerminalInfos) != 0 {
				t.Fatalf("retry pruning = %v, %v", destroyed, err)
			}
		})
	}
}
