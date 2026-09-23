package memory

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
)

// TestStorageRejectsExpirationOverflow verifies oversized lifetimes cannot become permanent or overwrite existing data. TestStorageRejectsExpirationOverflow 验证过大的有效期不会变成永久或覆盖已有数据。
func TestStorageRejectsExpirationOverflow(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"set", "set if absent", "expire"} {
		t.Run(operation, func(t *testing.T) {
			s := NewStorage()
			if operation != "set if absent" {
				if err := s.Set(ctx, "key", "original", time.Hour); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			switch operation {
			case "set":
				err = s.Set(ctx, "key", "replacement", time.Duration(math.MaxInt64))
			case "set if absent":
				var stored bool
				stored, err = s.SetIfAbsent(ctx, "key", "replacement", time.Duration(math.MaxInt64))
				if stored {
					t.Fatal("SetIfAbsent accepted an overflowing expiration")
				}
			case "expire":
				err = s.Expire(ctx, "key", time.Duration(math.MaxInt64))
			}
			if !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("expiration error = %v, want ErrInvalidParam", err)
			}
			value, err := s.Get(ctx, "key")
			if err != nil {
				t.Fatal(err)
			}
			if operation == "set if absent" {
				if value != nil || s.Exists(ctx, "key") {
					t.Fatal("rejected expiration created a key")
				}
			} else {
				if value != "original" {
					t.Fatalf("rejected expiration changed the original value: %v", value)
				}
				if ttl, err := s.TTL(ctx, "key"); err != nil || ttl <= 0 || ttl > time.Hour {
					t.Fatalf("rejected expiration changed original TTL: %v, %v", ttl, err)
				}
			}
		})
	}
}

// TestStorageAcceptsLongFiniteExpiration verifies long representable deadlines remain finite. TestStorageAcceptsLongFiniteExpiration 验证可表示的长有效期仍保持有限。
func TestStorageAcceptsLongFiniteExpiration(t *testing.T) {
	s := NewStorage()
	ctx := context.Background()
	lifetime := 100 * 365 * 24 * time.Hour
	if err := s.Set(ctx, "key", "value", lifetime); err != nil {
		t.Fatal(err)
	}
	if ttl, err := s.TTL(ctx, "key"); err != nil || ttl <= 0 || ttl > lifetime {
		t.Fatalf("long finite TTL = %v, %v", ttl, err)
	}
}
