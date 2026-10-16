package manager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestManagerNamespaceCannotAliasSessionAsToken covers the cross-namespace authentication boundary. TestManagerNamespaceCannotAliasSessionAsToken 覆盖跨命名空间的认证边界。
func TestManagerNamespaceCannotAliasSessionAsToken(t *testing.T) {
	ctx := context.Background()
	admin := newTestManager(t, func(cfg *config.Config) { cfg.AuthType = "admin:" })
	other := newTestManager(t, func(cfg *config.Config) { cfg.AuthType = "admin:token:" })
	other.storage = admin.storage
	if _, err := admin.Login(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Login(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if admin.getTokenKey("session:alice") == other.getSessionKey("alice") {
		t.Fatal("session key aliases another namespace's token")
	}
	if err := admin.CheckLogin(ctx, "session:alice"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin(session alias) = %v, want ErrInvalidToken", err)
	}

	// Historical unescaped records must also be rejected when found under a token key. 历史未转义记录落在 Token 键下时也必须拒绝。
	foreign, err := other.GetSession(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if err = admin.saveToStorage(ctx, admin.getTokenKey("session:alice"), foreign, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = admin.CheckLogin(ctx, "session:alice"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin(legacy session alias) = %v, want ErrInvalidToken", err)
	}
	if err = admin.saveToStorage(ctx, admin.getSessionKey("foreign-session"), foreign, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.GetSession(ctx, "foreign-session"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetSession(foreign namespace) = %v, want ErrInvalidToken", err)
	}
}

// TestManagerAccountDisableCannotChangeOtherKinds covers both new writes and legacy account markers. TestManagerAccountDisableCannotChangeOtherKinds 覆盖新写入及旧账号标记的类型隔离。
func TestManagerAccountDisableCannotChangeOtherKinds(t *testing.T) {
	ctx := context.Background()
	for _, ordinary := range []bool{false, true} {
		for _, kind := range []string{"service", "device", "concrete device"} {
			t.Run(fmt.Sprintf("%s/ordinary=%v", kind, ordinary), func(t *testing.T) {
				mgr := newTestManager(t, nil)
				if ordinary {
					mgr.storage = &managerStorageOnly{inner: newManagerTestStorage()}
				}
				account := "service:alice:pay"
				write := func() error { return mgr.DisableService(ctx, "alice", "pay", time.Minute) }
				check := func() bool { return mgr.IsDisableService(ctx, "alice", "pay") }
				key := mgr.getDisableServiceKey("alice", "pay")
				if kind == "device" {
					account = "device:alice:web"
					write = func() error { return mgr.DisableDevice(ctx, "alice", "web", time.Minute) }
					check = func() bool { return mgr.IsDisableDevice(ctx, "alice", "web") }
					key = mgr.getDisableDeviceKey("alice", "web")
				} else if kind == "concrete device" {
					account = "device:id:alice:web:phone"
					write = func() error { return mgr.DisableDeviceAndDeviceID(ctx, "alice", "web", "phone", time.Minute) }
					check = func() bool { return mgr.IsDisableDeviceAndDeviceID(ctx, "alice", "web", "phone") }
					key = mgr.getDisableDeviceAndDeviceIDKey("alice", "web", "phone")
				}
				if err := write(); err != nil {
					t.Fatal(err)
				}
				before, err := mgr.storage.Get(ctx, key)
				if err != nil {
					t.Fatal(err)
				}
				if mgr.IsDisable(ctx, account) {
					t.Fatal("another disable kind became an account ban")
				}
				if ttl, err := mgr.GetDisableTTL(ctx, account); err != nil || ttl != -2 {
					t.Fatalf("GetDisableTTL(unbanned account) = %d, %v", ttl, err)
				}
				if err = mgr.Untie(ctx, account); err != nil {
					t.Fatal(err)
				}
				if err = mgr.Disable(ctx, account, time.Minute); err != nil {
					t.Fatal(err)
				}
				if !mgr.IsDisable(ctx, account) || !check() {
					t.Fatal("independent disable records did not coexist")
				}
				if err = mgr.Untie(ctx, account); err != nil {
					t.Fatal(err)
				}
				if after, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(after, before) {
					t.Fatalf("account operations changed another kind: %v, %v", after, err)
				}

				// A recognizable legacy account marker remains an account restriction only. 可识别的旧账号标记仍只限制该账号。
				if err = mgr.saveToStorage(ctx, key, DisableInfo{DisableReason: "legacy account"}, time.Minute); err != nil {
					t.Fatal(err)
				}
				if !mgr.IsDisable(ctx, account) || check() {
					t.Fatal("legacy account marker crossed a disable type boundary")
				}
				if err = write(); !errors.Is(err, derror.ErrInvalidParam) {
					t.Fatalf("overwrite legacy account = %v, want ErrInvalidParam", err)
				}
				if err = mgr.Untie(ctx, account); err != nil || mgr.storage.Exists(ctx, key) {
					t.Fatalf("Untie(legacy owner) = %v, want removal", err)
				}
			})
		}
	}
}

// TestManagerDisableLegacyLookupChecksOwner covers same-scope collisions between escaped and raw account IDs. TestManagerDisableLegacyLookupChecksOwner 覆盖业务维度相同而转义账号不同的旧键碰撞。
func TestManagerDisableLegacyLookupChecksOwner(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"account", "service", "device", "concrete device"} {
		t.Run(kind, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			owner, other := "a:b", `a\:b`
			write := func(id string) error { return mgr.Disable(ctx, id, time.Minute) }
			check := func(id string) bool { return mgr.IsDisable(ctx, id) }
			untie := func(id string) error { return mgr.Untie(ctx, id) }
			if kind == "service" {
				write = func(id string) error { return mgr.DisableService(ctx, id, "pay", time.Minute) }
				check = func(id string) bool { return mgr.IsDisableService(ctx, id, "pay") }
				untie = func(id string) error { return mgr.UntieService(ctx, id, "pay") }
			} else if kind == "device" {
				write = func(id string) error { return mgr.DisableDevice(ctx, id, "web", time.Minute) }
				check = func(id string) bool { return mgr.IsDisableDevice(ctx, id, "web") }
				untie = func(id string) error { return mgr.UntieDevice(ctx, id, "web") }
			} else if kind == "concrete device" {
				write = func(id string) error { return mgr.DisableDeviceAndDeviceID(ctx, id, "web", "phone", time.Minute) }
				check = func(id string) bool { return mgr.IsDisableDeviceAndDeviceID(ctx, id, "web", "phone") }
				untie = func(id string) error { return mgr.UntieDeviceAndDeviceID(ctx, id, "web", "phone") }
			}
			if err := write(owner); err != nil {
				t.Fatal(err)
			}
			if check(other) {
				t.Fatal("legacy lookup adopted another account's ban")
			}
			if err := untie(other); err != nil || !check(owner) {
				t.Fatalf("foreign untie affected owner: %v", err)
			}
			if err := write(other); err != nil || !check(other) || !check(owner) {
				t.Fatalf("independent disable failed: %v", err)
			}
			if err := untie(other); err != nil || !check(owner) || check(other) {
				t.Fatalf("own untie crossed account boundary: %v", err)
			}
		})
	}
}

// TestManagerAmbiguousLegacyDisableIsNotOverwrittenOrDeleted requires explicit ownership migration. TestManagerAmbiguousLegacyDisableIsNotOverwrittenOrDeleted 要求对歧义旧数据显式迁移归属。
func TestManagerAmbiguousLegacyDisableIsNotOverwrittenOrDeleted(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	key := mgr.getDisableServiceKey("a:b", "pay")
	if err := mgr.saveToStorage(ctx, key, ServiceDisableInfo{Service: "pay", Level: 1}, time.Minute); err != nil {
		t.Fatal(err)
	}
	before, err := mgr.storage.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a:b", `a\:b`} {
		if err := mgr.CheckDisableService(ctx, id, "pay"); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("CheckDisableService(%q) = %v, want ambiguity error", id, err)
		}
		if _, err := mgr.GetDisableServiceTTL(ctx, id, "pay"); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("GetDisableServiceTTL(%q) = %v, want ambiguity error", id, err)
		}
		if err := mgr.UntieService(ctx, id, "pay"); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("UntieService(%q) = %v, want ambiguity error", id, err)
		}
	}
	if err := mgr.DisableService(ctx, "a:b", "pay", 0); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("DisableService(ambiguous existing key) = %v", err)
	}
	if after, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("ambiguous marker changed: %v, %v", after, err)
	}
}

// TestManagerDurationRoundTripSaturates verifies finite lifetimes cannot become unlimited through rounding. TestManagerDurationRoundTripSaturates 验证有限时长不会经取整转换成为永久有效。
func TestManagerDurationRoundTripSaturates(t *testing.T) {
	mgr := newTestManager(t, nil)
	for _, duration := range []time.Duration{1, time.Second, time.Second + 1, time.Duration(math.MaxInt64)} {
		seconds := mgr.timeoutToSeconds(duration)
		got := mgr.resolveTokenExpiration(&TokenInfo{Timeout: seconds})
		if got <= 0 || got < duration {
			t.Fatalf("duration round trip %v -> %d -> %v", duration, seconds, got)
		}
	}
	if got := secondsToDuration(math.MaxInt64); got != time.Duration(math.MaxInt64) {
		t.Fatalf("oversized stored seconds = %v, want maximum finite duration", got)
	}

	// Use the ordinary storage contract without an absolute UnixNano deadline restriction. 使用不受绝对 UnixNano 截止时间限制的普通存储契约。
	mgr.storage = &managerStorageOnly{inner: newManagerTestStorage()}
	ctx := context.Background()
	token, err := mgr.LoginWithTimeout(ctx, "long-lived", time.Duration(math.MaxInt64))
	if err != nil {
		t.Fatal(err)
	}
	shared, err := mgr.Login(ctx, "long-lived")
	if err != nil || shared != token {
		t.Fatalf("shared login = %q, %v, want %q", shared, err, token)
	}
	if ttl, err := mgr.storage.TTL(ctx, mgr.getTokenKey(token)); err != nil || ttl == adapter.TTLNoExpire || ttl <= 0 {
		t.Fatalf("shared token TTL = %v, %v, want finite positive duration", ttl, err)
	}
}
