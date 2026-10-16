package ticket

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestTicketPreciseExpiration verifies validity across whole-second boundaries and codec round trips. TestTicketPreciseExpiration 验证跨整秒边界及编解码后的精确有效期。
func TestTicketPreciseExpiration(t *testing.T) {
	createdAt := time.Unix(100, 900000000)
	mgr := newTestTicketManager(time.Minute)
	for _, timeout := range []time.Duration{time.Second, 500 * time.Millisecond} {
		t.Run(timeout.String(), func(t *testing.T) {
			value := &Ticket{
				Ticket: "precise", Status: StatusValid,
				CreateTime: createdAt.Unix(), ExpiresIn: durationSeconds(timeout),
				ExpiresAt: createdAt.Add(timeout),
			}
			encoded, err := mgr.serializer.Encode(value)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Ticket
			if err = mgr.serializer.Decode(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			for _, now := range []time.Time{time.Unix(101, 0), value.ExpiresAt.Add(-time.Nanosecond)} {
				if err = mgr.checkAlive(&decoded, now); err != nil {
					t.Fatalf("checkAlive(before precise deadline) = %v", err)
				}
			}
			for _, now := range []time.Time{value.ExpiresAt, value.ExpiresAt.Add(time.Nanosecond)} {
				if err = mgr.checkAlive(&decoded, now); !errors.Is(err, ErrTicketExpired) {
					t.Fatalf("checkAlive(at/after precise deadline) = %v, want ErrTicketExpired", err)
				}
			}
		})
	}

	// Old payloads retain their existing whole-second expiration semantics. 旧版载荷保留原有整秒到期语义。
	legacy := &Ticket{Ticket: "legacy", Status: StatusValid, CreateTime: 100, ExpiresIn: 1}
	if err := mgr.checkAlive(legacy, time.Unix(100, 999999999)); err != nil {
		t.Fatalf("checkAlive(legacy before deadline) = %v", err)
	}
	if err := mgr.checkAlive(legacy, time.Unix(101, 0)); !errors.Is(err, ErrTicketExpired) {
		t.Fatalf("checkAlive(legacy at deadline) = %v, want ErrTicketExpired", err)
	}
}

// TestCreateTicketPreciseDeadline verifies creation preserves sub-second duration precision. TestCreateTicketPreciseDeadline 验证创建票据时保留亚秒有效期精度。
func TestCreateTicketPreciseDeadline(t *testing.T) {
	mgr := newTestTicketManager(time.Minute)
	for _, timeout := range []time.Duration{time.Second, 500 * time.Millisecond} {
		before := time.Now()
		created, err := mgr.CreateWithTimeout(context.Background(), CreateOptions{}, timeout)
		after := time.Now()
		if err != nil {
			t.Fatal(err)
		}
		if created.ExpiresAt.Before(before.Add(timeout)) || created.ExpiresAt.After(after.Add(timeout)) {
			t.Fatalf("ExpiresAt = %v, want creation time + %v", created.ExpiresAt, timeout)
		}
		if created.ExpiresIn != 1 {
			t.Fatalf("ExpiresIn = %d, want rounded metadata of 1 second", created.ExpiresIn)
		}
	}
}

// TestTicketStateWritesPreserveDeadline verifies consume/revoke keep the stored precise deadline and TTL. TestTicketStateWritesPreserveDeadline 验证消费和撤销保留精确截止时间及存储 TTL。
func TestTicketStateWritesPreserveDeadline(t *testing.T) {
	for _, action := range []string{"consume", "revoke"} {
		t.Run(action, func(t *testing.T) {
			ctx := context.Background()
			storage := &ticketTTLRecordingStorage{ticketTestStorage: newTicketTestStorage()}
			mgr := NewDefaultManager("test", "dt:", storage, ticketTestCodec{})
			created, err := mgr.CreateWithTimeout(ctx, CreateOptions{}, time.Minute+500*time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			before := time.Now()
			wantStatus := StatusConsumed
			if action == "consume" {
				_, err = mgr.Consume(ctx, created.Ticket)
			} else {
				wantStatus = StatusRevoked
				err = mgr.Revoke(ctx, created.Ticket)
			}
			after := time.Now()
			if err != nil {
				t.Fatal(err)
			}
			if storage.lastTTL < created.ExpiresAt.Sub(after) || storage.lastTTL > created.ExpiresAt.Sub(before) {
				t.Fatalf("state write TTL = %v, want remaining precise lifetime", storage.lastTTL)
			}
			stored, err := mgr.get(ctx, created.Ticket)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Status != wantStatus || !stored.ExpiresAt.Equal(created.ExpiresAt) {
				t.Fatalf("stored ticket = %+v, want %s and original deadline", stored, wantStatus)
			}
		})
	}
}

// TestSaveExpiredTicketDoesNotCreatePersistentKey verifies expired writes never use a non-positive storage TTL. TestSaveExpiredTicketDoesNotCreatePersistentKey 验证过期写入不会通过非正 TTL 创建永久键。
func TestSaveExpiredTicketDoesNotCreatePersistentKey(t *testing.T) {
	ctx := context.Background()
	mgr := newTestTicketManager(time.Minute)
	value := &Ticket{Ticket: "expired", Status: StatusValid, ExpiresAt: time.Now().Add(-time.Second)}
	if err := mgr.save(ctx, value, time.Minute); !errors.Is(err, ErrTicketExpired) {
		t.Fatalf("save(expired) = %v, want ErrTicketExpired", err)
	}
	if mgr.storage.Exists(ctx, mgr.getTicketKey(value.Ticket)) {
		t.Fatal("expired save created a storage key")
	}
}

// TestTicketPlainStorageLifecycle verifies basic storage supports constraints, consumption and revocation. TestTicketPlainStorageLifecycle 验证基础存储支持约束校验、消费及撤销。
func TestTicketPlainStorageLifecycle(t *testing.T) {
	ctx := context.Background()
	storage := struct{ adapter.Storage }{newTicketTestStorage()}
	mgr := NewDefaultManager("test", "dt:", storage, ticketTestCodec{})
	created, err := mgr.Create(ctx, CreateOptions{TargetApp: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = mgr.Consume(ctx, created.Ticket, ValidateOptions{TargetApp: "other"}); !errors.Is(err, ErrTicketMismatch) {
		t.Fatalf("Consume(mismatch) = %v, want ErrTicketMismatch", err)
	}
	if _, err = mgr.Validate(ctx, created.Ticket); err != nil {
		t.Fatalf("Validate(after mismatch) = %v", err)
	}
	result, err := mgr.Consume(ctx, created.Ticket, ValidateOptions{TargetApp: "admin"})
	if err != nil || result == nil || result.Ticket.Status != StatusConsumed {
		t.Fatalf("Consume() = %+v, %v, want consumed ticket", result, err)
	}
	if _, err = mgr.Consume(ctx, created.Ticket); !errors.Is(err, ErrTicketConsumed) {
		t.Fatalf("Consume(replay) = %v, want ErrTicketConsumed", err)
	}
	if err = mgr.Revoke(ctx, created.Ticket); err != nil {
		t.Fatal(err)
	}
	if status, err := mgr.Status(ctx, created.Ticket); err != nil || status != StatusConsumed {
		t.Fatalf("Status(after revoke) = %s, %v, want consumed", status, err)
	}
	created, err = mgr.Create(ctx, CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err = mgr.Revoke(ctx, created.Ticket); err != nil {
		t.Fatal(err)
	}
	if _, err = mgr.Consume(ctx, created.Ticket); !errors.Is(err, ErrTicketRevoked) {
		t.Fatalf("Consume(revoked) = %v, want ErrTicketRevoked", err)
	}
}

// TestTicketConcurrentConsumption verifies only one consumer succeeds with either storage capability. TestTicketConcurrentConsumption 验证两种存储能力下均只有一个并发消费者成功。
func TestTicketConcurrentConsumption(t *testing.T) {
	for _, mode := range []string{"atomic", "plain"} {
		t.Run(mode, func(t *testing.T) {
			var storage adapter.Storage = newTicketTestStorage()
			if mode == "plain" {
				storage = struct{ adapter.Storage }{storage}
			}
			mgr := NewDefaultManager("test", "dt:", storage, ticketTestCodec{})
			created, err := mgr.Create(context.Background(), CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			const consumers = 16
			start := make(chan struct{})
			results := make(chan error, consumers)
			var wg sync.WaitGroup
			for i := 0; i < consumers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					_, err := mgr.Consume(context.Background(), created.Ticket)
					results <- err
				}()
			}
			close(start)
			wg.Wait()
			close(results)
			successes := 0
			for err := range results {
				if err == nil {
					successes++
				} else if !errors.Is(err, ErrTicketConsumed) && !errors.Is(err, ErrInvalidTicket) {
					t.Errorf("Consume() = %v, want consumed or missing", err)
				}
			}
			if successes != 1 {
				t.Fatalf("successful consumers = %d, want 1", successes)
			}
		})
	}
}

// TestTicketConsumptionStorageErrors verifies atomic preference and fallback deletion/save errors. TestTicketConsumptionStorageErrors 验证优先使用原子能力及回退删除和保存错误。
func TestTicketConsumptionStorageErrors(t *testing.T) {
	for _, mode := range []string{"atomic", "plain delete failure", "plain save failure"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			deleteFailure := &ticketDeleteErrorStorage{newTicketTestStorage()}
			saveFailure := &ticketConsumeSaveFailStorage{ticketTestStorage: newTicketTestStorage()}
			var storage adapter.Storage = deleteFailure
			switch mode {
			case "plain delete failure":
				storage = struct{ adapter.Storage }{deleteFailure}
			case "plain save failure":
				storage = struct{ adapter.Storage }{saveFailure}
			}
			mgr := NewDefaultManager("test", "dt:", storage, ticketTestCodec{})
			created, err := mgr.Create(ctx, CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			saveFailure.failSets = true
			_, err = mgr.Consume(ctx, created.Ticket)
			if mode == "atomic" {
				if err != nil {
					t.Fatalf("Consume(atomic) = %v, must bypass ordinary Delete", err)
				}
			} else if !errors.Is(err, derror.ErrStorageUnavailable) {
				t.Fatalf("Consume(%s) = %v, want ErrStorageUnavailable", mode, err)
			}
			if mode == "plain delete failure" {
				if _, err = mgr.Validate(ctx, created.Ticket); err != nil {
					t.Fatalf("Validate(after failed deletion) = %v", err)
				}
			}
		})
	}
}

// ticketTTLRecordingStorage records state-write lifetimes. ticketTTLRecordingStorage 记录状态回写的有效期。
type ticketTTLRecordingStorage struct {
	*ticketTestStorage
	lastTTL time.Duration
}

func (s *ticketTTLRecordingStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	s.lastTTL = expiration
	return s.ticketTestStorage.Set(ctx, key, value, expiration)
}

// ticketDeleteErrorStorage rejects ordinary deletion while preserving atomic operations. ticketDeleteErrorStorage 拒绝普通删除，同时保留原子操作。
type ticketDeleteErrorStorage struct {
	*ticketTestStorage
}

func (s *ticketDeleteErrorStorage) Delete(context.Context, ...string) error {
	return errors.New("delete failed")
}
