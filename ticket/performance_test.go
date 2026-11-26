package ticket

import (
	"context"
	"errors"
	"testing"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestServiceRevokeReadsOnce checks actual state transitions drive events without preflight reads. TestServiceRevokeReadsOnce 验证实际状态变更驱动事件，无需预读。
func TestServiceRevokeReadsOnce(t *testing.T) {
	for _, state := range []string{"valid", "consumed", "revoked", "missing", "write failure"} {
		t.Run(state, func(t *testing.T) {
			ctx := context.Background()
			mgr := newTestTicketManager(t, time.Minute)
			opts := CreateOptions{LoginID: "user"}

			created, err := mgr.Create(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}
			switch state {
			case "consumed":
				_, err = mgr.Consume(ctx, created.Ticket)
			case "revoked":
				err = mgr.Revoke(ctx, created.Ticket)
			case "missing":
				err = mgr.storage.Delete(ctx, mgr.getTicketKey(created.Ticket))
			}
			if err != nil {
				t.Fatal(err)
			}
			storage := &revokeCountingStorage{Storage: mgr.storage, failWrite: state == "write failure"}
			mgr.storage = storage
			calls := 0
			service := NewService(mgr, revokeEmitter(func(event listener.Event, loginID, token string, extra map[string]any) {
				calls++
				if event != listener.EventTicketRevoke || token != created.Ticket || extra[listener.ExtraKeyStatus] != StatusRevoked {
					t.Errorf("unexpected revoke event: %s, %s, %v", event, token, extra)
				}
				if loginID != opts.LoginID {
					t.Errorf("event identity = %q, want %q", loginID, opts.LoginID)
				}
			}))
			err = service.RevokeTicket(ctx, created.Ticket)
			if state == "write failure" {
				if !errors.Is(err, derror.ErrStorageUnavailable) {
					t.Fatalf("write error = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			wantEvents := 0
			if state == "valid" {
				wantEvents = 1
			}
			if storage.reads != 1 || calls != wantEvents {
				t.Fatalf("reads/events = %d/%d, want 1/%d", storage.reads, calls, wantEvents)
			}
			if wantEvents == 1 {
				if err = service.RevokeTicket(ctx, created.Ticket); err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Fatal("repeated revoke emitted another event")
				}
			}
		})
	}
}

type revokeCountingStorage struct {
	adapter.Storage
	reads     int
	failWrite bool
}

// Get counts credential reads. Get 统计凭据读取。
func (s *revokeCountingStorage) Get(ctx context.Context, key string) (any, error) {
	s.reads++
	return s.Storage.Get(ctx, key)
}

// Set injects failed state persistence. Set 注入状态持久化失败。
func (s *revokeCountingStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.failWrite {
		return errors.New("write unavailable")
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

type revokeEmitter func(listener.Event, string, string, map[string]any)

// Emit captures lifecycle metadata. Emit 捕获生命周期元数据。
func (f revokeEmitter) Emit(event listener.Event, loginID, device, deviceID, token string, extra map[string]any) {
	f(event, loginID, token, extra)
}

// TestUnrelatedCredentialsDoNotWaitForStorage verifies slow credential I/O does not hold an instance-wide lock. TestUnrelatedCredentialsDoNotWaitForStorage 验证单个凭据的慢存储操作不会持有实例级锁。
func TestUnrelatedCredentialsDoNotWaitForStorage(t *testing.T) {
	ctx := context.Background()
	mgr := newTestTicketManager(t, time.Minute)
	first, err := mgr.Create(ctx, CreateOptions{LoginID: "first"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := mgr.Create(ctx, CreateOptions{LoginID: "second"})
	if err != nil {
		t.Fatal(err)
	}
	storage := &blockedCredentialStorage{Storage: mgr.storage, key: mgr.getTicketKey(first.Ticket), entered: make(chan struct{}), release: make(chan struct{})}
	mgr.storage = storage
	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)
	defer func() {
		close(storage.release)
		select {
		case err := <-firstDone:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(time.Second):
			t.Error("blocked operation did not finish after release")
		}
	}()
	go func() { firstDone <- mgr.Revoke(ctx, first.Ticket) }()
	select {
	case <-storage.entered:
	case <-time.After(time.Second):
		t.Fatal("first operation did not reach storage")
	}
	go func() { _, err := mgr.Consume(ctx, second.Ticket); secondDone <- err }()
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("unrelated credential blocked behind slow storage")
	}
}

type blockedCredentialStorage struct {
	adapter.Storage
	key              string
	entered, release chan struct{}
}

// Get blocks only the selected credential without holding the backend lock. Get 仅阻塞指定凭据，不持有底层存储锁。
func (s *blockedCredentialStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.key {
		close(s.entered)
		<-s.release
	}
	return s.Storage.Get(ctx, key)
}
