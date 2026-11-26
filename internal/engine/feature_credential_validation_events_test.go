package engine

import (
	"testing"

	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerCredentialValidationEventsReportResult verifies Ticket and ShortKey validation events include both outcomes. TestManagerCredentialValidationEventsReportResult 验证 Ticket 与 ShortKey 校验事件包含成功和失败结果。

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
