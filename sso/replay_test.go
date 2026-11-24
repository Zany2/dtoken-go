package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// newReplayTestProtection composes isolated test protection with borrowed storage. newReplayTestProtection 使用借用存储组装隔离测试组件。
func newReplayTestProtection(t *testing.T, storage contract.Storage) *ReplayProtection {
	t.Helper()
	if storage == nil {
		memory := NewMemoryStorage()
		storage = memory
	}
	p, err := NewReplayProtection(ReplayProtectionOptions{Storage: storage})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestReplayMetadata checks clock boundaries, required fields, and operation binding. TestReplayMetadata 检查时间边界、必填字段与操作绑定。
func TestReplayMetadata(t *testing.T) {
	p := newReplayTestProtection(t, nil)
	names := DefaultParamNames()
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	base := url.Values{"client": {"app"}, "action": {ActionIntrospect}, "nonce": {"0123456789abcdef"}, "timestamp": {now.Format(time.RFC3339Nano)}}
	for _, tc := range []struct {
		name string
		edit func(url.Values)
		want error
	}{
		{name: "fresh"},
		{name: "past boundary", edit: func(v url.Values) { v.Set("timestamp", now.Add(-p.maxAge).Format(time.RFC3339Nano)) }, want: ErrRequestExpired},
		{name: "inside past window", edit: func(v url.Values) { v.Set("timestamp", now.Add(-p.maxAge+time.Nanosecond).Format(time.RFC3339Nano)) }},
		{name: "future boundary", edit: func(v url.Values) { v.Set("timestamp", now.Add(p.maxAge).Format(time.RFC3339Nano)) }},
		{name: "future outside window", edit: func(v url.Values) { v.Set("timestamp", now.Add(p.maxAge+time.Nanosecond).Format(time.RFC3339Nano)) }, want: ErrRequestExpired},
		{name: "malformed timestamp", edit: func(v url.Values) { v.Set("timestamp", "invalid") }, want: ErrRequestExpired},
		{name: "wrong operation", edit: func(v url.Values) { v.Set("action", ActionRevoke) }, want: ErrInvalidRequest},
		{name: "short nonce", edit: func(v url.Values) { v.Set("nonce", "short") }, want: ErrInvalidRequest},
		{name: "long nonce", edit: func(v url.Values) { v.Set("nonce", strings.Repeat("a", 129)) }, want: ErrInvalidRequest},
		{name: "nonce alphabet", edit: func(v url.Values) { v.Set("nonce", "0123456789abcde/") }, want: ErrInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := cloneValues(base)
			if tc.edit != nil {
				tc.edit(values)
			}
			if err := p.validate(values, names, ActionIntrospect, now); !errors.Is(err, tc.want) {
				t.Fatalf("validate = %v, want %v", err, tc.want)
			}
		})
	}
	for _, key := range []string{"client", "action", "timestamp", "nonce"} {
		for _, duplicate := range []bool{false, true} {
			values := cloneValues(base)
			if duplicate {
				values.Add(key, values.Get(key))
			} else {
				values.Del(key)
			}
			if err := p.validate(values, names, ActionIntrospect, now); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("field %s duplicate=%v: %v", key, duplicate, err)
			}
		}
	}
}

// TestReplayConcurrentClaims checks atomic replicas and the ordinary-storage instance fallback. TestReplayConcurrentClaims 检查原子存储多副本与普通存储实例内回退。
func TestReplayConcurrentClaims(t *testing.T) {
	for _, basic := range []bool{false, true} {
		t.Run(map[bool]string{false: "atomic replicas", true: "ordinary storage"}[basic], func(t *testing.T) {
			memory := NewMemoryStorage()
			var storage contract.Storage = memory
			if basic {
				storage = &basicSSOStorage{inner: memory}
			}
			first := newReplayTestProtection(t, storage)
			second := first
			if !basic {
				second = newReplayTestProtection(t, storage)
			}
			values := first.stamp(url.Values{"client": {"app"}}, DefaultParamNames(), ActionIntrospect)
			var accepted atomic.Int32
			var wg sync.WaitGroup
			for i := 0; i < 32; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					p := []*ReplayProtection{first, second}[i%2]
					err := p.claim(context.Background(), values, DefaultParamNames(), ActionIntrospect)
					if err == nil {
						accepted.Add(1)
					} else if !errors.Is(err, ErrRequestReplayed) {
						t.Errorf("claim: %v", err)
					}
				}(i)
			}
			wg.Wait()
			if accepted.Load() != 1 {
				t.Fatalf("accepted %d requests", accepted.Load())
			}
		})
	}
}

// replayFaultStorage exposes ordinary storage with controllable faults and TTL capture. replayFaultStorage 暴露可控制故障并记录 TTL 的普通存储。
type replayFaultStorage struct {
	contract.Storage
	readErr, writeErr error
	ttl               time.Duration
}

// Get injects read failures. Get 注入读取故障。
func (s *replayFaultStorage) Get(ctx context.Context, key string) (any, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.Storage.Get(ctx, key)
}

// Set captures retention and injects write failures. Set 记录保留时长并注入写入故障。
func (s *replayFaultStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	s.ttl = ttl
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

// replayAtomicFaultStorage injects atomic write failures without falling back. replayAtomicFaultStorage 注入原子写入故障且不允许回退。
type replayAtomicFaultStorage struct {
	*MemoryStorage
}

// SetIfAbsent simulates an unavailable atomic backend. SetIfAbsent 模拟原子存储不可用。
func (s *replayAtomicFaultStorage) SetIfAbsent(context.Context, string, any, time.Duration) (bool, error) {
	return false, errors.New("atomic backend unavailable")
}

// TestReplayStorageFailuresAndRetention checks fail-closed behavior and future-skew retention. TestReplayStorageFailuresAndRetention 检查故障时拒绝放行与未来偏差保留时间。
func TestReplayStorageFailuresAndRetention(t *testing.T) {
	memory := NewMemoryStorage()
	storage := &replayFaultStorage{Storage: memory}
	p := newReplayTestProtection(t, storage)
	names := DefaultParamNames()
	values := p.stamp(url.Values{"client": {"app"}}, names, ActionIntrospect)
	values.Set(names.Timestamp, time.Now().Add(p.maxAge-time.Second).Format(time.RFC3339Nano))
	failure := errors.New("storage offline")
	storage.readErr = failure
	if err := p.claim(context.Background(), values, names, ActionIntrospect); !errors.Is(err, ErrStorageUnavailable) || !errors.Is(err, failure) {
		t.Fatalf("read failure: %v", err)
	}
	storage.readErr, storage.writeErr = nil, failure
	if err := p.claim(context.Background(), values, names, ActionIntrospect); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("write failure: %v", err)
	}
	storage.writeErr = nil
	if err := p.claim(context.Background(), values, names, ActionIntrospect); err != nil {
		t.Fatal(err)
	}
	if storage.ttl != 2*p.maxAge {
		t.Fatalf("retention = %s", storage.ttl)
	}
	values.Set(names.Timestamp, time.Now().Format(time.RFC3339Nano))
	if err := p.claim(context.Background(), values, names, ActionIntrospect); !errors.Is(err, ErrRequestReplayed) {
		t.Fatalf("timestamp change reused nonce: %v", err)
	}
	values.Set(names.Client, "another-app")
	if err := p.claim(context.Background(), values, names, ActionIntrospect); err != nil {
		t.Fatalf("client namespace: %v", err)
	}
	broken := newReplayTestProtection(t, &replayAtomicFaultStorage{MemoryStorage: memory})
	if err := broken.claim(context.Background(), values, names, ActionIntrospect); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("atomic failure: %v", err)
	}
}

// TestReplayConfiguration rejects unsafe wiring without changing legacy defaults. TestReplayConfiguration 拒绝不安全配置且不改变旧版默认行为。
func TestReplayConfiguration(t *testing.T) {
	memory := NewMemoryStorage()
	for _, options := range []ReplayProtectionOptions{
		{}, {Storage: memory, MaxAge: -time.Second}, {Storage: memory, MaxAge: time.Duration(1<<63 - 1)},
	} {
		if _, err := NewReplayProtection(options); !errors.Is(err, ErrReplayProtectionConfig) {
			t.Fatalf("constructor: %v", err)
		}
	}
	p := newReplayTestProtection(t, memory)
	server := NewServer()
	defer server.Close()
	for _, cfg := range []ClientConfig{
		{ReplayProtection: p},
		{ReplayProtection: p, CheckSign: true, Params: ParamNames{Action: "sign"}},
		{ReplayProtection: &ReplayProtection{}, CheckSign: true},
	} {
		cfg.ServerURL, cfg.ClientID, cfg.SecretKey = "https://center.example", "app", "secret"
		app := NewClientApp(cfg)
		if _, err := app.AuthURL("https://app.example/callback", nil); !errors.Is(err, ErrReplayProtectionConfig) {
			t.Fatalf("client config: %v", err)
		}
		if _, err := app.VerifyLogoutCallback(httptest.NewRequest(http.MethodPost, "/", nil)); !errors.Is(err, ErrReplayProtectionConfig) {
			t.Fatalf("callback config: %v", err)
		}
		h := NewHTTPServer(server, HTTPOptions{ReplayProtection: cfg.ReplayProtection, ServerOptions: ServerOptions{CheckSign: cfg.CheckSign, Params: cfg.Params}})
		if !errors.Is(h.initializationError(), ErrReplayProtectionConfig) {
			t.Fatalf("server config: %v", h.initializationError())
		}
	}
	legacy := NewClientApp(ClientConfig{ServerURL: "https://center.example", ClientID: "app", CheckSign: true, SecretKey: "secret"})
	target, err := legacy.AuthURL("https://app.example/callback", nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("nonce") != "" || u.Query().Get("action") != "" || !NewSigner("secret").Verify(u.Query()) {
		t.Fatal("legacy signing behavior changed")
	}
}
