package shortkey

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNamespaceSeparatorCompatibility verifies creation, confirmation and consumption across equivalent namespace spellings. TestNamespaceSeparatorCompatibility 验证等价命名空间写法之间的创建、确认与消费。
func TestNamespaceSeparatorCompatibility(t *testing.T) {
	for _, source := range []string{"test", "test:"} {
		t.Run(source, func(t *testing.T) {
			base := newTestShortKeyManager(t, time.Minute)
			writer, err := NewDefaultManager(source, "dt:", base.storage, base.serializer)
			if err != nil {
				t.Fatal(err)
			}
			target := "test:"
			if source == target {
				target = "test"
			}
			reader, err := NewDefaultManager(target, "dt:", base.storage, base.serializer)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			value, err := writer.Create(ctx, CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if value.AuthType != source {
				t.Fatal("creation changed the public AuthType representation")
			}
			if _, err := reader.Confirm(ctx, value.Key, ConfirmOptions{LoginID: "user"}); err != nil {
				t.Fatalf("equivalent namespace rejected: %v", err)
			}
			if _, err := writer.Validate(ctx, value.Key); err != nil {
				t.Fatal(err)
			}
			if _, err := reader.Consume(ctx, value.Key); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Validate(ctx, value.Key); !errors.Is(err, ErrShortKeyConsumed) {
				t.Fatalf("consumption not shared: %v", err)
			}
			for _, foreign := range []string{"other", "test::", " test"} {
				value.AuthType = foreign
				encoded, err := base.serializer.Encode(value)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := reader.decode(encoded, value.Key); !errors.Is(err, ErrInvalidShortKey) {
					t.Fatalf("foreign namespace %q accepted: %v", foreign, err)
				}
			}
		})
	}
}
