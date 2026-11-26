package ticket

import (
	"context"
	"errors"
	"testing"
	"time"

	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// TestServiceMemoryLifecycle checks capability transitions, binding rejection and event outcomes using real Memory storage. TestServiceMemoryLifecycle 使用真实 Memory 存储验证能力状态流转、绑定拒绝和事件结果。
func TestServiceMemoryLifecycle(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewDefaultManager("service", "test", memory.NewStorage(), ticketTestCodec{})
	if err != nil {
		t.Fatal(err)
	}
	events := map[listener.Event]int{}
	results := []bool{}
	service := NewService(mgr, revokeEmitter(func(event listener.Event, loginID, token string, extra map[string]any) {
		events[event]++
		if token == "" {
			t.Error("event omitted credential")
		}
		if event == listener.EventTicketValidate {
			value, ok := extra[listener.ExtraKeyResult].(bool)
			if !ok {
				t.Error("validation result missing")
			}
			results = append(results, value)
		}
	}))
	created, err := service.CreateTicketWithTimeout(ctx, CreateOptions{LoginID: "user", TargetApp: "consumer"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	if status, err := service.GetTicketStatus(ctx, created.Ticket); err != nil || status != StatusValid {
		t.Fatalf("active status=%s,%v", status, err)
	}
	if ttl, err := service.GetTicketTTL(ctx, created.Ticket); err != nil || ttl <= 0 || ttl > 60 {
		t.Fatalf("ttl=%d,%v", ttl, err)
	}
	if _, err = service.ValidateTicket(ctx, created.Ticket, ValidateOptions{TargetApp: "wrong"}); !errors.Is(err, ErrTicketMismatch) {
		t.Fatalf("binding error=%v", err)
	}
	if _, err = service.ConsumeTicket(ctx, created.Ticket, ValidateOptions{TargetApp: "wrong"}); !errors.Is(err, ErrTicketMismatch) {
		t.Fatalf("consume binding error=%v", err)
	}
	for i := 0; i < 2; i++ {
		if value, err := service.ValidateTicket(ctx, created.Ticket, ValidateOptions{LoginID: "user", TargetApp: "consumer"}); err != nil || value.LoginID != "user" {
			t.Fatalf("validate=%+v,%v", value, err)
		}
	}
	result, err := service.ConsumeTicket(ctx, created.Ticket, ValidateOptions{TargetApp: "consumer"})
	if err != nil || result == nil || result.Ticket.LoginID != "user" {
		t.Fatalf("consume=%+v,%v", result, err)
	}
	if _, err = service.ConsumeTicket(ctx, created.Ticket); !errors.Is(err, ErrTicketConsumed) {
		t.Fatalf("replay=%v", err)
	}
	if status, err := service.GetTicketStatus(ctx, created.Ticket); err != nil || status != StatusConsumed {
		t.Fatalf("consumed status=%s,%v", status, err)
	}
	if _, err = service.ValidateTicket(ctx, created.Ticket); !errors.Is(err, ErrTicketConsumed) {
		t.Fatalf("consumed validation=%v", err)
	}
	if events[listener.EventTicketCreate] != 1 || events[listener.EventTicketConsume] != 1 {
		t.Fatalf("events=%v", events)
	}
	if len(results) != 4 || results[0] || !results[1] || !results[2] || results[3] {
		t.Fatalf("validation results=%v", results)
	}
	revoked, err := service.CreateTicket(ctx, CreateOptions{LoginID: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if err = service.RevokeTicket(ctx, revoked.Ticket); err != nil {
		t.Fatal(err)
	}
	if status, err := service.GetTicketStatus(ctx, revoked.Ticket); err != nil || status != StatusRevoked {
		t.Fatalf("revoke status=%s,%v", status, err)
	}
	if _, err = service.ValidateTicket(ctx, revoked.Ticket); !errors.Is(err, ErrTicketRevoked) {
		t.Fatalf("revoked validation=%v", err)
	}
}

// TestServiceDisabled rejects all operations without a capability manager. TestServiceDisabled 在未绑定能力管理器时拒绝全部操作。
func TestServiceDisabled(t *testing.T) {
	ctx := context.Background()
	service := NewService(nil, nil)
	checks := []func() error{
		func() error { _, err := service.CreateTicket(ctx, CreateOptions{}); return err },
		func() error { _, err := service.CreateTicketWithTimeout(ctx, CreateOptions{}, time.Minute); return err },

		func() error { _, err := service.ValidateTicket(ctx, "key"); return err },
		func() error { _, err := service.ConsumeTicket(ctx, "key"); return err },
		func() error { return service.RevokeTicket(ctx, "key") },
		func() error { _, err := service.GetTicketStatus(ctx, "key"); return err },
		func() error { _, err := service.GetTicketTTL(ctx, "key"); return err },
	}
	for i, check := range checks {
		if err := check(); !errors.Is(err, derror.ErrModuleNotEnabled) {
			t.Errorf("operation %d=%v", i, err)
		}
	}
}
