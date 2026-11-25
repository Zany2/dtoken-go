package ticket

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNamespaceSeparatorCompatibility verifies both existing payload forms share one namespace without accepting distinct names. TestNamespaceSeparatorCompatibility 验证两种已有载荷形式共享同一命名空间，且不接受不同名称。
func TestNamespaceSeparatorCompatibility(t *testing.T) {
	for _, source := range []string{"test", "test:"} {
		t.Run(source, func(t *testing.T) {
			base := newTestTicketManager(t, time.Minute)
			writer, err := NewDefaultManager(source, base.keyPrefix, base.storage, base.serializer)
			if err != nil {
				t.Fatal(err)
			}
			target := "test:"
			if source == target {
				target = "test"
			}
			reader, err := NewDefaultManager(target, base.keyPrefix, base.storage, base.serializer)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			value, err := writer.Create(ctx, CreateOptions{LoginID: "user"})
			if err != nil {
				t.Fatal(err)
			}
			if value.AuthType != source {
				t.Fatal("creation changed the public AuthType representation")
			}
			if _, err := reader.Validate(ctx, value.Ticket); err != nil {
				t.Fatalf("equivalent namespace rejected: %v", err)
			}
			if _, err := reader.Consume(ctx, value.Ticket); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Validate(ctx, value.Ticket); !errors.Is(err, ErrTicketConsumed) {
				t.Fatalf("consumption not shared: %v", err)
			}
			for _, foreign := range []string{"other", "test::", " test"} {
				value.AuthType = foreign
				encoded, err := base.serializer.Encode(value)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := reader.decode(encoded, value.Ticket); !errors.Is(err, ErrInvalidTicket) {
					t.Fatalf("foreign namespace %q accepted: %v", foreign, err)
				}
			}
		})
	}
}
