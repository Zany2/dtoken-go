package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/ticket"
)

func TestManagerTicketCreateValidateConsume(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithTicket(t, nil)

	created, err := mgr.CreateTicket(ctx, ticket.CreateOptions{
		LoginID:   "user-1001",
		Device:    "web",
		DeviceId:  "browser-1",
		Source:    "scan",
		SourceApp: "portal",
		TargetApp: "admin",
		Scopes:    []string{"profile"},
		Extra:     map[string]any{"scene": "qr"},
	})
	if err != nil {
		t.Fatalf("CreateTicket() error = %v", err)
	}
	if created.Ticket == "" || created.AuthType == "" || created.Status != ticket.StatusValid {
		t.Fatalf("CreateTicket() = %+v, want ticket/authType/valid status", created)
	}

	validated, err := mgr.ValidateTicket(ctx, created.Ticket, ticket.ValidateOptions{
		LoginID:   "user-1001",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("ValidateTicket() error = %v", err)
	}
	if validated.LoginID != "user-1001" || validated.TargetApp != "admin" {
		t.Fatalf("ValidateTicket() = %+v, want preserved constraints", validated)
	}

	result, err := mgr.ConsumeTicket(ctx, created.Ticket, ticket.ValidateOptions{
		LoginID:   "user-1001",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("ConsumeTicket() error = %v", err)
	}
	if result.Ticket.Status != ticket.StatusConsumed || result.Ticket.LoginID != "user-1001" {
		t.Fatalf("ConsumeTicket() = %+v, want consumed user ticket", result.Ticket)
	}

	status, err := mgr.GetTicketStatus(ctx, created.Ticket)
	if err != nil {
		t.Fatalf("GetTicketStatus() error = %v", err)
	}
	if status != ticket.StatusConsumed {
		t.Fatalf("GetTicketStatus() = %s, want consumed", status)
	}
	if _, err = mgr.ConsumeTicket(ctx, created.Ticket); !errors.Is(err, ticket.ErrTicketConsumed) {
		t.Fatalf("ConsumeTicket() second error = %v, want ErrTicketConsumed", err)
	}
}

func TestManagerTicketBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithTicket(t, nil)

	created, err := mgr.CreateTicket(ctx, ticket.CreateOptions{
		LoginID:   "user-1001",
		TargetApp: "admin",
	})
	if err != nil {
		t.Fatalf("CreateTicket() error = %v", err)
	}
	if _, err = mgr.ValidateTicket(ctx, created.Ticket, ticket.ValidateOptions{TargetApp: "console"}); !errors.Is(err, ticket.ErrTicketMismatch) {
		t.Fatalf("ValidateTicket() mismatch error = %v, want ErrTicketMismatch", err)
	}

	if err = mgr.RevokeTicket(ctx, created.Ticket); err != nil {
		t.Fatalf("RevokeTicket() error = %v", err)
	}
	status, err := mgr.GetTicketStatus(ctx, created.Ticket)
	if err != nil {
		t.Fatalf("GetTicketStatus() revoked error = %v", err)
	}
	if status != ticket.StatusRevoked {
		t.Fatalf("GetTicketStatus() = %s, want revoked", status)
	}
	if _, err = mgr.ValidateTicket(ctx, created.Ticket); !errors.Is(err, ticket.ErrTicketRevoked) {
		t.Fatalf("ValidateTicket() revoked error = %v, want ErrTicketRevoked", err)
	}

	expiring, err := mgr.CreateTicketWithTimeout(ctx, ticket.CreateOptions{LoginID: "user-1002"}, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("CreateTicketWithTimeout() error = %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	status, err = mgr.GetTicketStatus(ctx, expiring.Ticket)
	if err != nil {
		t.Fatalf("GetTicketStatus() expired error = %v", err)
	}
	if status != ticket.StatusInvalid {
		t.Fatalf("GetTicketStatus() after storage expiry = %s, want invalid", status)
	}
	ttl, err := mgr.GetTicketTTL(ctx, expiring.Ticket)
	if err != nil {
		t.Fatalf("GetTicketTTL() error = %v", err)
	}
	if ttl != -2 {
		t.Fatalf("GetTicketTTL() = %d, want -2", ttl)
	}
}

func newTestManagerWithTicket(t *testing.T, mutate func(*config.Config)) *Manager {
	t.Helper()
	mgr := newTestManager(t, mutate)
	WithTicketManager(ticket.NewDefaultManager(
		mgr.GetConfig().AuthType,
		mgr.GetConfig().KeyPrefix,
		mgr.GetStorage(),
		mgr.GetSerializer(),
	))(mgr)
	return mgr
}
