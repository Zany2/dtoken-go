package authcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestNeedAuth verifies empty requests skip auth checks TestNeedAuth 验证空请求会跳过认证校验。
func TestNeedAuth(t *testing.T) {
	if NeedAuth(Request{}) {
		t.Fatal("NeedAuth(empty) = true, want false")
	}
	if !NeedAuth(Request{CheckLogin: true}) {
		t.Fatal("NeedAuth(CheckLogin) = false, want true")
	}
	if !NeedAuth(Request{Permissions: []string{"read"}}) {
		t.Fatal("NeedAuth(Permissions) = false, want true")
	}
}

// TestCheckLoginDisablePermissionsAndRoles verifies shared auth decision behavior TestCheckLoginDisablePermissionsAndRoles 验证公共认证决策行为。
func TestCheckLoginDisablePermissionsAndRoles(t *testing.T) {
	ctx := context.Background()
	mgr := newAuthcheckTestManager(t)

	token, err := mgr.Login(ctx, "u1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "u1", []string{"profile:read", "order:*"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if err = mgr.AddRoles(ctx, "u1", []string{"member", "admin"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}

	result, err := Check(ctx, mgr, Request{
		TokenValue:  token,
		CheckLogin:  true,
		Permissions: []string{"profile:read", "order:update"},
		Roles:       []string{"admin"},
		LogicType:   LogicAnd,
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.LoginID != "u1" {
		t.Fatalf("LoginID = %q, want u1", result.LoginID)
	}

	_, err = Check(ctx, mgr, Request{
		TokenValue:  token,
		CheckLogin:  true,
		Permissions: []string{"missing"},
	})
	if !errors.Is(err, derror.ErrPermissionDenied) {
		t.Fatalf("Check(missing permission) error = %v, want ErrPermissionDenied", err)
	}

	_, err = Check(ctx, mgr, Request{
		TokenValue: token,
		CheckLogin: true,
		Roles:      []string{"missing"},
	})
	if !errors.Is(err, derror.ErrRoleDenied) {
		t.Fatalf("Check(missing role) error = %v, want ErrRoleDenied", err)
	}

	if err = mgr.Disable(ctx, "u1", time.Minute); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	_, err = Check(ctx, mgr, Request{TokenValue: token, CheckLogin: true, CheckDisable: true})
	if !errors.Is(err, derror.ErrNotLogin) && !errors.Is(err, derror.ErrAccountDisabled) {
		t.Fatalf("Check(disabled) error = %v, want auth failure", err)
	}
}

// TestCheckWithoutExplicitLoginUsesTokenChecks verifies middleware-style token checks TestCheckWithoutExplicitLoginUsesTokenChecks 验证中间件风格的 token 校验。
func TestCheckWithoutExplicitLoginUsesTokenChecks(t *testing.T) {
	ctx := context.Background()
	mgr := newAuthcheckTestManager(t)

	token, err := mgr.Login(ctx, "u2")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissionsByToken(ctx, token, []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissionsByToken() error = %v", err)
	}

	result, err := Check(ctx, mgr, Request{
		TokenValue:  token,
		Permissions: []string{"report:read"},
		LogicType:   LogicAnd,
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.LoginID != "" {
		t.Fatalf("LoginID = %q, want empty when login ID is not needed", result.LoginID)
	}
}

// TestGetErrorCodeAndMessage verifies stable error mapping TestGetErrorCodeAndMessage 验证稳定错误映射。
func TestGetErrorCodeAndMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{name: "dtoken error", err: derror.NewDTokenError(499, "custom", derror.ErrInvalidParam), code: 499},
		{name: "not login", err: derror.ErrNotLogin, code: derror.CodeNotLogin},
		{name: "invalid token", err: derror.ErrInvalidToken, code: derror.CodeTokenInvalid},
		{name: "permission", err: derror.ErrPermissionDenied, code: derror.CodePermissionDenied},
		{name: "disabled", err: derror.ErrDeviceDisabled, code: derror.CodeAccountDisabled},
		{name: "bad request", err: derror.ErrIDIsEmpty, code: derror.CodeBadRequest},
		{name: "storage", err: derror.ErrStorageUnavailable, code: derror.CodeStorageError},
		{name: "not found", err: derror.ErrClientNotFound, code: derror.CodeNotFound},
		{name: "oauth2 token", err: derror.ErrInvalidAccessToken, code: derror.CodeTokenInvalid},
		{name: "ticket expired", err: derror.ErrTicketExpired, code: derror.CodeTokenExpired},
		{name: "short key pending", err: derror.ErrShortKeyPending, code: derror.CodeBadRequest},
		{name: "server", err: errors.New("boom"), code: derror.CodeServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, message := GetErrorCodeAndMessage(tt.err)
			if code != tt.code {
				t.Fatalf("code = %d, want %d", code, tt.code)
			}
			if message == "" {
				t.Fatal("message is empty")
			}
		})
	}
}

func newAuthcheckTestManager(t *testing.T) *manager.Manager {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.AsyncEvent = false
	cfg.AutoRenew = false
	cfg.RenewInterval = config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config invalid: %v", err)
	}

	mgr := manager.NewManager(
		cfg,
		&authcheckTestGenerator{},
		newAuthcheckTestStorage(),
		authcheckTestCodec{},
		adapter.NewNopLogger(),
		nil,
		nil,
	)
	t.Cleanup(mgr.CloseManager)
	return mgr
}

type authcheckTestGenerator struct {
	mu  sync.Mutex
	seq int
}

func (g *authcheckTestGenerator) Generate(loginID, device, deviceID string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	return fmt.Sprintf("token-%s-%s-%s-%d", loginID, device, deviceID, g.seq), nil
}

type authcheckTestCodec struct{}

func (authcheckTestCodec) Name() string { return "json-test" }

func (authcheckTestCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

func (authcheckTestCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

type authcheckTestStorage struct {
	mu    sync.RWMutex
	items map[string]authcheckTestStorageItem
}

type authcheckTestStorageItem struct {
	value    any
	expireAt time.Time
}

func newAuthcheckTestStorage() *authcheckTestStorage {
	return &authcheckTestStorage{items: map[string]authcheckTestStorageItem{}}
}

func (s *authcheckTestStorage) Set(_ context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expireAt time.Time
	if expiration > 0 {
		expireAt = time.Now().Add(expiration)
	}
	s.items[key] = authcheckTestStorageItem{value: value, expireAt: expireAt}
	return nil
}

func (s *authcheckTestStorage) Get(_ context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	if item.expired() {
		delete(s.items, key)
		return nil, nil
	}
	return item.value, nil
}

func (s *authcheckTestStorage) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

func (s *authcheckTestStorage) Exists(_ context.Context, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return false
	}
	if item.expired() {
		delete(s.items, key)
		return false
	}
	return true
}

func (s *authcheckTestStorage) Expire(_ context.Context, key string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok || item.expired() {
		delete(s.items, key)
		return derror.ErrInvalidToken
	}
	if expiration > 0 {
		item.expireAt = time.Now().Add(expiration)
	} else {
		item.expireAt = time.Time{}
	}
	s.items[key] = item
	return nil
}

func (s *authcheckTestStorage) TTL(_ context.Context, key string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return adapter.TTLNotFound, nil
	}
	if item.expired() {
		delete(s.items, key)
		return adapter.TTLNotFound, nil
	}
	if item.expireAt.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	ttl := time.Until(item.expireAt)
	if ttl <= 0 {
		delete(s.items, key)
		return adapter.TTLNotFound, nil
	}
	return ttl, nil
}

func (s *authcheckTestStorage) Ping(context.Context) error { return nil }

func (s *authcheckTestStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	value, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	_ = s.Delete(ctx, key)
	return value, nil
}

func (s *authcheckTestStorage) GetAndDeleteMany(ctx context.Context, key string, deleteKeys ...string) (any, error) {
	value, err := s.GetAndDelete(ctx, key)
	if err != nil || value == nil {
		return value, err
	}
	_ = s.Delete(ctx, deleteKeys...)
	return value, nil
}

func (s *authcheckTestStorage) SetIfAbsent(ctx context.Context, key string, value any, expiration time.Duration) (bool, error) {
	if s.Exists(ctx, key) {
		return false, nil
	}
	if err := s.Set(ctx, key, value, expiration); err != nil {
		return false, err
	}
	return true, nil
}

func (s *authcheckTestStorage) Keys(_ context.Context, pattern string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.items))
	for key, item := range s.items {
		if item.expired() {
			delete(s.items, key)
			continue
		}
		if matchAuthcheckTestPattern(pattern, key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *authcheckTestStorage) Clear(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = map[string]authcheckTestStorageItem{}
	return nil
}

func (item authcheckTestStorageItem) expired() bool {
	return !item.expireAt.IsZero() && time.Now().After(item.expireAt)
}

func matchAuthcheckTestPattern(pattern, value string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}
	if !strings.HasPrefix(value, parts[0]) {
		return false
	}
	value = strings.TrimPrefix(value, parts[0])
	for _, part := range parts[1 : len(parts)-1] {
		idx := strings.Index(value, part)
		if idx < 0 {
			return false
		}
		value = value[idx+len(part):]
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(value, last)
}
