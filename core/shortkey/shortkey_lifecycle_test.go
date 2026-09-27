package shortkey

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestShortKeyPreciseExpiration verifies sub-second deadlines survive serialization and retain legacy compatibility. TestShortKeyPreciseExpiration 验证亚秒截止时间在序列化后保留，并兼容旧版载荷。
func TestShortKeyPreciseExpiration(t *testing.T) {
	mgr := newTestShortKeyManager(time.Minute)
	createdAt := time.Unix(100, 900000000)
	for _, status := range []Status{StatusPending, StatusConfirmed} {
		for _, timeout := range []time.Duration{time.Second, 500 * time.Millisecond} {
			value := &ShortKey{Key: "precise", Status: status, CreateTime: createdAt.Unix(), ExpiresIn: 1, ExpiresAt: createdAt.Add(timeout)}
			encoded, err := mgr.serializer.Encode(value)
			if err != nil {
				t.Fatal(err)
			}
			var decoded ShortKey
			if err = mgr.serializer.Decode(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			for _, now := range []time.Time{time.Unix(101, 0), value.ExpiresAt.Add(-time.Nanosecond)} {
				if err = mgr.checkUsable(&decoded, now); err != nil {
					t.Fatalf("checkUsable(%s, %v, before deadline) = %v", status, timeout, err)
				}
			}
			if err = mgr.checkUsable(&decoded, value.ExpiresAt); !errors.Is(err, ErrShortKeyExpired) {
				t.Fatalf("checkUsable(at deadline) = %v, want ErrShortKeyExpired", err)
			}
		}
	}
	legacy := &ShortKey{Key: "legacy", Status: StatusConfirmed, CreateTime: 100, ExpiresIn: 1}
	if err := mgr.checkUsable(legacy, time.Unix(100, 999999999)); err != nil {
		t.Fatalf("checkUsable(legacy before deadline) = %v", err)
	}
	if err := mgr.checkUsable(legacy, time.Unix(101, 0)); !errors.Is(err, ErrShortKeyExpired) {
		t.Fatalf("checkUsable(legacy at deadline) = %v, want ErrShortKeyExpired", err)
	}
}

// TestShortKeyStateWritesPreserveDeadline verifies confirmation, consumption and revocation retain the precise TTL. TestShortKeyStateWritesPreserveDeadline 验证确认、消费及撤销保留精确 TTL。
func TestShortKeyStateWritesPreserveDeadline(t *testing.T) {
	for _, action := range []string{"confirm", "consume", "revoke"} {
		t.Run(action, func(t *testing.T) {
			ctx := context.Background()
			storage := &shortKeyTTLRecordingStorage{shortKeyTestStorage: newShortKeyTestStorage()}
			mgr := NewDefaultManager("test", "dt:", storage, shortKeyTestCodec{})
			opts := CreateOptions{}
			if action == "consume" {
				opts.LoginID = "user"
			}
			created, err := mgr.CreateWithTimeout(ctx, opts, time.Minute+500*time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			before := time.Now()
			wantStatus := StatusConfirmed
			switch action {
			case "confirm":
				_, err = mgr.Confirm(ctx, created.Key, ConfirmOptions{LoginID: "user"})
			case "consume":
				wantStatus = StatusConsumed
				_, err = mgr.Consume(ctx, created.Key)
			case "revoke":
				wantStatus = StatusRevoked
				err = mgr.Revoke(ctx, created.Key)
			}
			after := time.Now()
			if err != nil {
				t.Fatal(err)
			}
			if storage.lastTTL < created.ExpiresAt.Sub(after) || storage.lastTTL > created.ExpiresAt.Sub(before) {
				t.Fatalf("state write TTL = %v, want remaining precise lifetime", storage.lastTTL)
			}
			stored, err := mgr.get(ctx, created.Key)
			if err != nil || stored.Status != wantStatus || !stored.ExpiresAt.Equal(created.ExpiresAt) {
				t.Fatalf("stored short key = %+v, %v, want %s and original deadline", stored, err, wantStatus)
			}
		})
	}
}

// TestShortKeyPlainStorageLifecycle verifies pending and mismatched keys remain usable for later confirmation and consumption. TestShortKeyPlainStorageLifecycle 验证基础存储的待确认及约束不匹配短 Key 可继续确认和消费。
func TestShortKeyPlainStorageLifecycle(t *testing.T) {
	ctx := context.Background()
	storage := struct{ adapter.Storage }{newShortKeyTestStorage()}
	mgr := NewDefaultManager("test", "dt:", storage, shortKeyTestCodec{})
	created, err := mgr.Create(ctx, CreateOptions{Scene: "login"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mgr.Consume(ctx, created.Key); !errors.Is(err, ErrShortKeyPending) {
		t.Fatalf("Consume(pending) = %v, want ErrShortKeyPending", err)
	}
	if _, err = mgr.Confirm(ctx, created.Key, ConfirmOptions{LoginID: "user"}); err != nil {
		t.Fatal(err)
	}
	if _, err = mgr.Consume(ctx, created.Key, ValidateOptions{Scene: "other"}); !errors.Is(err, ErrShortKeyMismatch) {
		t.Fatalf("Consume(mismatch) = %v, want ErrShortKeyMismatch", err)
	}
	result, err := mgr.Consume(ctx, created.Key, ValidateOptions{Scene: "login", LoginID: "user"})
	if err != nil || result == nil || result.ShortKey.Status != StatusConsumed {
		t.Fatalf("Consume() = %+v, %v, want consumed key", result, err)
	}
	if _, err = mgr.Consume(ctx, created.Key); !errors.Is(err, ErrShortKeyConsumed) {
		t.Fatalf("Consume(replay) = %v, want ErrShortKeyConsumed", err)
	}
	if err = mgr.Revoke(ctx, created.Key); err != nil {
		t.Fatal(err)
	}
	if status, err := mgr.Status(ctx, created.Key); err != nil || status != StatusConsumed {
		t.Fatalf("Status(after revoke) = %s, %v, want consumed", status, err)
	}
}

// TestShortKeyConcurrentTransitions verifies a single confirmation and a single consumption succeed per key. TestShortKeyConcurrentTransitions 验证每个短 Key 仅有一次并发确认和消费成功。
func TestShortKeyConcurrentTransitions(t *testing.T) {
	for _, mode := range []string{"atomic", "plain"} {
		t.Run(mode, func(t *testing.T) {
			var storage adapter.Storage = newShortKeyTestStorage()
			if mode == "plain" {
				storage = struct{ adapter.Storage }{storage}
			}
			mgr := NewDefaultManager("test", "dt:", storage, shortKeyTestCodec{})
			created, err := mgr.Create(context.Background(), CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{"confirm", "consume"} {
				const workers = 16
				start := make(chan struct{})
				results := make(chan error, workers)
				var wg sync.WaitGroup
				for i := 0; i < workers; i++ {
					wg.Add(1)
					go func(id int) {
						defer wg.Done()
						<-start
						var err error
						if action == "confirm" {
							_, err = mgr.Confirm(context.Background(), created.Key, ConfirmOptions{LoginID: fmt.Sprintf("user-%d", id)})
						} else {
							_, err = mgr.Consume(context.Background(), created.Key)
						}
						results <- err
					}(i)
				}
				close(start)
				wg.Wait()
				close(results)
				successes := 0
				wantErr := ErrShortKeyMismatch
				if action == "consume" {
					wantErr = ErrShortKeyConsumed
				}
				for err := range results {
					if err == nil {
						successes++
					} else if !errors.Is(err, wantErr) {
						t.Errorf("%s error = %v, want %v", action, err, wantErr)
					}
				}
				if successes != 1 {
					t.Fatalf("successful %s calls = %d, want 1", action, successes)
				}
			}
		})
	}
}

// TestShortKeyCreationCollisions verifies retry exhaustion never overwrites occupied keys. TestShortKeyCreationCollisions 验证重试耗尽不会覆盖已占用短 Key。
func TestShortKeyCreationCollisions(t *testing.T) {
	for _, mode := range []string{"atomic", "plain"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			var storage adapter.Storage = newShortKeyTestStorage()
			if mode == "plain" {
				storage = struct{ adapter.Storage }{storage}
			}
			mgr := NewManagerWithConfig("test", "dt:", storage, shortKeyTestCodec{}, &Config{TTL: time.Minute, Length: 1, MaxGenerateRetries: 3})
			for _, char := range alphabet {
				if err := storage.Set(ctx, mgr.getKey(string(char)), "occupied", time.Minute); err != nil {
					t.Fatal(err)
				}
			}
			if created, err := mgr.Create(ctx, CreateOptions{}); err == nil || created != nil {
				t.Fatalf("Create(full key space) = %+v, %v, want collision error", created, err)
			}
			for _, char := range alphabet {
				if got, err := storage.Get(ctx, mgr.getKey(string(char))); err != nil || got != "occupied" {
					t.Fatalf("occupied key %c = %v, %v", char, got, err)
				}
			}
		})
	}
}

// TestShortKeyExpiredWrites verifies expired writes never become persistent storage entries. TestShortKeyExpiredWrites 验证过期写入不会成为永久存储记录。
func TestShortKeyExpiredWrites(t *testing.T) {
	for _, mode := range []string{"atomic", "plain"} {
		t.Run(mode, func(t *testing.T) {
			var storage adapter.Storage = newShortKeyTestStorage()
			if mode == "plain" {
				storage = struct{ adapter.Storage }{storage}
			}
			mgr := NewDefaultManager("test", "dt:", storage, shortKeyTestCodec{})
			value := &ShortKey{Key: "expired", Status: StatusPending, ExpiresAt: time.Now().Add(-time.Second)}
			ctx := context.Background()
			if err := mgr.save(ctx, value, time.Minute); !errors.Is(err, ErrShortKeyExpired) {
				t.Fatalf("save(expired) = %v, want ErrShortKeyExpired", err)
			}
			if saved, err := mgr.saveIfAbsent(ctx, value, time.Minute); saved || !errors.Is(err, ErrShortKeyExpired) {
				t.Fatalf("saveIfAbsent(expired) = %v, %v, want false and ErrShortKeyExpired", saved, err)
			}
			if storage.Exists(ctx, mgr.getKey(value.Key)) {
				t.Fatal("expired write created a storage key")
			}
		})
	}
}

// TestShortKeyConsumptionStorageErrors verifies atomic preference and fallback error propagation. TestShortKeyConsumptionStorageErrors 验证原子能力优先及回退错误传递。
func TestShortKeyConsumptionStorageErrors(t *testing.T) {
	for _, mode := range []string{"atomic", "plain delete failure", "plain save failure"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			deleteFailure := &shortKeyDeleteErrorStorage{newShortKeyTestStorage()}
			saveFailure := &shortKeyConsumeSaveFailStorage{shortKeyTestStorage: newShortKeyTestStorage()}
			var storage adapter.Storage = deleteFailure
			switch mode {
			case "plain delete failure":
				storage = struct{ adapter.Storage }{deleteFailure}
			case "plain save failure":
				storage = struct{ adapter.Storage }{saveFailure}
			}
			mgr := NewDefaultManager("test", "dt:", storage, shortKeyTestCodec{})
			created, err := mgr.Create(ctx, CreateOptions{LoginID: "user"})
			if err != nil {
				t.Fatal(err)
			}
			saveFailure.failSets = true
			_, err = mgr.Consume(ctx, created.Key)
			if mode == "atomic" {
				if err != nil {
					t.Fatalf("Consume(atomic) = %v, must bypass ordinary Delete", err)
				}
			} else if !errors.Is(err, derror.ErrStorageUnavailable) {
				t.Fatalf("Consume(%s) = %v, want ErrStorageUnavailable", mode, err)
			}
			if mode == "plain delete failure" {
				if _, err = mgr.Validate(ctx, created.Key); err != nil {
					t.Fatalf("Validate(after failed deletion) = %v", err)
				}
			}
		})
	}
}

// shortKeyTTLRecordingStorage records state-write lifetimes. shortKeyTTLRecordingStorage 记录状态回写的有效期。
type shortKeyTTLRecordingStorage struct {
	*shortKeyTestStorage
	lastTTL time.Duration
}

func (s *shortKeyTTLRecordingStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	s.lastTTL = expiration
	return s.shortKeyTestStorage.Set(ctx, key, value, expiration)
}

// shortKeyDeleteErrorStorage rejects ordinary deletion while preserving atomic operations. shortKeyDeleteErrorStorage 拒绝普通删除，同时保留原子操作。
type shortKeyDeleteErrorStorage struct {
	*shortKeyTestStorage
}

func (s *shortKeyDeleteErrorStorage) Delete(context.Context, ...string) error {
	return errors.New("delete failed")
}
