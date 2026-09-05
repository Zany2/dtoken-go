package manager

import (
	"context"
	"testing"

	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// TestManagerCredentialValidationEventsReportResult verifies Ticket and ShortKey validation events include both outcomes. TestManagerCredentialValidationEventsReportResult 验证 Ticket 与 ShortKey 校验事件包含成功和失败结果。
func TestManagerCredentialValidationEventsReportResult(t *testing.T) {
	ctx := context.Background()

	t.Run("ticket", func(t *testing.T) {
		mgr := newTestManagerWithTicket(t, nil)
		created, err := mgr.CreateTicket(ctx, ticket.CreateOptions{LoginID: "ticket-event-user", TargetApp: "admin"})
		if err != nil {
			t.Fatalf("CreateTicket() error = %v", err)
		}
		validationEvents := registerManagerValidationEventCollector(mgr, listener.EventTicketValidate)

		if _, err = mgr.ValidateTicket(ctx, created.Ticket, ticket.ValidateOptions{TargetApp: "admin"}); err != nil {
			t.Fatalf("ValidateTicket(success) error = %v", err)
		}
		if _, err = mgr.ValidateTicket(ctx, created.Ticket, ticket.ValidateOptions{TargetApp: "console"}); err == nil {
			t.Fatal("ValidateTicket(mismatch) error = nil")
		}

		assertManagerValidationResults(t, validationEvents(), created.Ticket, created.LoginID)
	})

	t.Run("short key", func(t *testing.T) {
		mgr := newTestManagerWithShortKey(t, nil)
		created, err := mgr.CreateShortKey(ctx, shortkey.CreateOptions{LoginID: "short-key-event-user", TargetApp: "admin"})
		if err != nil {
			t.Fatalf("CreateShortKey() error = %v", err)
		}
		validationEvents := registerManagerValidationEventCollector(mgr, listener.EventShortKeyValidate)

		if _, err = mgr.ValidateShortKey(ctx, created.Key, shortkey.ValidateOptions{TargetApp: "admin"}); err != nil {
			t.Fatalf("ValidateShortKey(success) error = %v", err)
		}
		if _, err = mgr.ValidateShortKey(ctx, created.Key, shortkey.ValidateOptions{TargetApp: "console"}); err == nil {
			t.Fatal("ValidateShortKey(mismatch) error = nil")
		}

		assertManagerValidationResults(t, validationEvents(), created.Key, created.LoginID)
	})
}

func registerManagerValidationEventCollector(mgr *Manager, event listener.Event) func() []*listener.EventData {
	events := make([]*listener.EventData, 0, 2)
	mgr.GetEventManager().RegisterFuncWithConfig(event, func(data *listener.EventData) {
		copyData := *data
		events = append(events, &copyData)
	}, listener.ListenerConfig{Async: false})
	return func() []*listener.EventData {
		return events
	}
}

func assertManagerValidationResults(t *testing.T, events []*listener.EventData, credential, loginID string) {
	t.Helper()
	if len(events) != 2 {
		t.Fatalf("validation events = %d, want 2", len(events))
	}
	for i, wantResult := range []bool{true, false} {
		if events[i].Token != credential || events[i].Extra[listener.ExtraKeyAction] != listener.ActionValidate || events[i].Extra[listener.ExtraKeyResult] != wantResult {
			t.Fatalf("validation event[%d] = %+v, want token %q and result %v", i, events[i], credential, wantResult)
		}
	}
	if events[0].LoginID != loginID {
		t.Fatalf("successful validation LoginID = %q, want %q", events[0].LoginID, loginID)
	}
	if events[1].LoginID != "" {
		t.Fatalf("failed validation LoginID = %q, want empty", events[1].LoginID)
	}
}
