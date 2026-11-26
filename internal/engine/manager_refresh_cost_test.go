package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerRefreshOperationCosts checks record reuse across complete rotation and revocation flows. TestManagerRefreshOperationCosts 检查完整轮换与撤销流程中的记录复用。
func TestManagerRefreshOperationCosts(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"basic", "atomic"} {
		for _, operation := range []string{"rotate", "revoke"} {
			for _, activity := range []string{"disabled", "enabled"} {
				t.Run(mode+"/"+operation+"/"+activity, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.AutoRenew = false
						cfg.ActiveTimeout = config.NoLimit
						if activity == "enabled" {
							cfg.ActiveTimeout = 60
						}
					})
					pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
						LoginID: "refresh-cost", Device: "web", DeviceID: "browser",
						Extra: map[string]any{"tenant": "original"}, TerminalExtra: map[string]any{"client": "original"},
					}})
					if err != nil {
						t.Fatal(err)
					}
					storage, codec := instrumentRefreshOperation(mgr, pair, mode)
					events := 0
					event := listener.EventRefreshTokenRotate
					if operation == "revoke" {
						event = listener.EventRefreshTokenRevoke
					}
					mgr.GetEventManager().RegisterFuncWithConfig(event, func(*listener.EventData) { events++ }, listener.ListenerConfig{Async: false})
					var next *RefreshTokenPair
					if operation == "rotate" {
						next, err = mgr.RefreshToken(ctx, pair.RefreshToken)
					} else {
						err = mgr.RevokeRefreshToken(ctx, pair.RefreshToken)
					}
					if err != nil || events != 1 {
						t.Fatalf("operation = %v, events=%d", err, events)
					}
					wantGets, wantDecodes := 2, 2
					if mode == "atomic" {
						if operation == "revoke" {
							wantGets = 1
						} else {
							wantDecodes = 3
						}
					}
					if storage.refreshGets != wantGets || codec.refreshDecodes != wantDecodes || storage.consumes != 1 {
						t.Fatalf("refresh GETs/decodes/consumes = %d/%d/%d, want %d/%d/1", storage.refreshGets, codec.refreshDecodes, storage.consumes, wantGets, wantDecodes)
					}
					wantAccess := 0
					if operation == "rotate" {
						wantAccess = 1
					}
					if storage.accessGetsAtConsume != wantAccess || storage.sessionGetsAtConsume != 0 {
						t.Fatalf("pre-consumption access/session GETs = %d/%d, want %d/0", storage.accessGetsAtConsume, storage.sessionGetsAtConsume, wantAccess)
					}
					if mgr.storage.Exists(ctx, storage.refreshKey) || mgr.storage.Exists(ctx, storage.accessKey) || mgr.storage.Exists(ctx, mgr.getTokenRefreshKey(pair.AccessToken)) {
						t.Fatal("operation retained the old credential or its reverse binding")
					}
					if next != nil {
						info, err := mgr.getRefreshTokenInfo(ctx, next.RefreshToken)
						if err != nil || info.Extra["tenant"] != "original" || info.TerminalExtra["client"] != "original" {
							t.Fatalf("replacement metadata = %+v, %v", info, err)
						}
					}
					if _, err := mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
						t.Fatalf("replay = %v", err)
					}
					if err := mgr.RevokeRefreshToken(ctx, pair.RefreshToken); err != nil || events != 1 {
						t.Fatalf("idempotent revoke = %v, events=%d", err, events)
					}
				})
			}
		}
	}
}

// TestManagerRefreshConsumptionFailures preserves access state when consumption fails or a credential disappears before the locked operation. TestManagerRefreshConsumptionFailures 验证消费失败或凭据在锁内操作前消失时保留访问状态。
func TestManagerRefreshConsumptionFailures(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"basic", "atomic"} {
		for _, operation := range []string{"rotate", "revoke"} {
			for _, failure := range []string{"delete failure", "disappeared"} {
				t.Run(mode+"/"+operation+"/"+failure, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) { cfg.AutoRenew = false })
					pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-failure", "web")
					if err != nil {
						t.Fatal(err)
					}
					before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getTokenKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getSessionKey(pair.LoginID)})
					storage, _ := instrumentRefreshOperation(mgr, pair, mode)
					if failure == "delete failure" {
						storage.beforeConsume = func() error { return errors.New("consume unavailable") }
					} else {
						storage.afterRefreshRead = func(count int) error {
							if count == 1 {
								return storage.Storage.Delete(ctx, storage.refreshKey)
							}
							return nil
						}
					}
					events := 0
					for _, event := range []listener.Event{listener.EventRefreshTokenRotate, listener.EventRefreshTokenRevoke, listener.EventLogout} {
						mgr.GetEventManager().RegisterFuncWithConfig(event, func(*listener.EventData) { events++ }, listener.ListenerConfig{Async: false})
					}
					if operation == "rotate" {
						_, err = mgr.RefreshToken(ctx, pair.RefreshToken)
					} else {
						err = mgr.RevokeRefreshToken(ctx, pair.RefreshToken)
					}
					var wantErr error
					if failure == "delete failure" {
						wantErr = derror.ErrStorageUnavailable
					} else if operation == "rotate" {
						wantErr = derror.ErrInvalidRefreshToken
					}
					if !errors.Is(err, wantErr) || events != 0 {
						t.Fatalf("operation = %v, want %v; events=%d", err, wantErr, events)
					}
					assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
					if storage.Storage.Exists(ctx, storage.refreshKey) != (failure == "delete failure") {
						t.Fatal("unexpected refresh state after failed consumption")
					}
				})
			}
		}
	}
}

// TestManagerRefreshAtomicConsumptionChecksIdentity keeps access cleanup bound to the preflight identity. TestManagerRefreshAtomicConsumptionChecksIdentity 验证访问清理仍绑定预检身份。
func TestManagerRefreshAtomicConsumptionChecksIdentity(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"rotate", "revoke"} {
		for _, field := range []string{"account", "access token", "access identity", "creation time"} {
			t.Run(operation+"/"+field, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) { cfg.AutoRenew = false })
				pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-owner", "web")
				if err != nil {
					t.Fatal(err)
				}
				record, err := mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
				if err != nil {
					t.Fatal(err)
				}
				switch field {
				case "account":
					record.LoginID += "-changed"
				case "access token":
					record.AccessToken += "-changed"
				case "access identity":
					record.AccessID += "-changed"
				case "creation time":
					record.CreateTime++
				}
				encoded, err := mgr.serializer.Encode(record)
				if err != nil {
					t.Fatal(err)
				}
				before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getTokenKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getSessionKey(pair.LoginID)})
				storage, _ := instrumentRefreshOperation(mgr, pair, "atomic")
				storage.beforeConsume = func() error { return storage.Storage.Set(ctx, storage.refreshKey, encoded, 0) }
				events := 0
				for _, event := range []listener.Event{listener.EventRefreshTokenRotate, listener.EventRefreshTokenRevoke, listener.EventLogout} {
					mgr.GetEventManager().RegisterFuncWithConfig(event, func(*listener.EventData) { events++ }, listener.ListenerConfig{Async: false})
				}
				if operation == "rotate" {
					_, err = mgr.RefreshToken(ctx, pair.RefreshToken)
					if !errors.Is(err, derror.ErrInvalidRefreshToken) {
						t.Fatalf("changed consumption = %v", err)
					}
				} else if err = mgr.RevokeRefreshToken(ctx, pair.RefreshToken); err != nil {
					t.Fatal(err)
				}
				if events != 0 || storage.Storage.Exists(ctx, storage.refreshKey) {
					t.Fatalf("mismatched consumption: events=%d", events)
				}
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			})
		}
	}
}

// instrumentRefreshOperation counts only the original credential and preserves the selected storage capabilities. instrumentRefreshOperation 仅统计原凭据，并保留选定的存储能力。
func instrumentRefreshOperation(mgr *Manager, pair *RefreshTokenPair, mode string) (*refreshCostStorage, *refreshCostCodec) {
	storage := &refreshCostStorage{Storage: mgr.storage, refreshKey: mgr.getRefreshTokenKey(pair.RefreshToken), accessKey: mgr.getTokenKey(pair.AccessToken), sessionKey: mgr.getSessionKey(pair.LoginID)}
	codec := &refreshCostCodec{Codec: mgr.serializer}
	mgr.storage, mgr.serializer = storage, codec
	if mode == "atomic" {
		mgr.storage = &refreshAtomicCostStorage{refreshCostStorage: storage, AtomicStorage: storage.Storage.(adapter.AtomicStorage)}
	}
	return storage, codec
}

type refreshCostStorage struct {
	adapter.Storage
	refreshKey, accessKey, sessionKey              string
	refreshGets, accessGets, sessionGets, consumes int
	accessGetsAtConsume, sessionGetsAtConsume      int
	afterRefreshRead                               func(int) error
	beforeConsume                                  func() error
}

// Get counts selected reads and models changes after the account-lookup snapshot. Get 统计选定读取，并模拟定位账号快照后的状态变化。
func (s *refreshCostStorage) Get(ctx context.Context, key string) (any, error) {
	data, err := s.Storage.Get(ctx, key)
	switch key {
	case s.refreshKey:
		s.refreshGets++
		if err == nil && s.afterRefreshRead != nil {
			err = s.afterRefreshRead(s.refreshGets)
		}
	case s.accessKey:
		s.accessGets++
	case s.sessionKey:
		s.sessionGets++
	}
	return data, err
}

// captureConsumption records the preflight budget and applies a configured storage interleaving. captureConsumption 记录预检开销，并执行配置的存储交错。
func (s *refreshCostStorage) captureConsumption() error {
	s.consumes++
	s.accessGetsAtConsume, s.sessionGetsAtConsume = s.accessGets, s.sessionGets
	if s.beforeConsume != nil {
		return s.beforeConsume()
	}
	return nil
}

// Delete counts basic consumption without introducing another read. Delete 统计普通消费，不额外读取。
func (s *refreshCostStorage) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if key == s.refreshKey {
			if err := s.captureConsumption(); err != nil {
				return err
			}
		}
	}
	return s.Storage.Delete(ctx, keys...)
}

type refreshAtomicCostStorage struct {
	*refreshCostStorage
	adapter.AtomicStorage
}

// GetAndDelete counts native consumption separately from ordinary GET. GetAndDelete 将原子消费与普通 GET 分开计数。
func (s *refreshAtomicCostStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	if key == s.refreshKey {
		if err := s.captureConsumption(); err != nil {
			return nil, err
		}
	}
	return s.AtomicStorage.GetAndDelete(ctx, key)
}

type refreshCostCodec struct {
	adapter.Codec
	refreshDecodes int
}

// Decode counts refresh metadata decoding, including native consumption results. Decode 统计刷新元数据解码，包括原子消费结果。
func (c *refreshCostCodec) Decode(data []byte, value any) error {
	if _, ok := value.(*refreshTokenRecord); ok {
		c.refreshDecodes++
	}
	return c.Codec.Decode(data, value)
}
