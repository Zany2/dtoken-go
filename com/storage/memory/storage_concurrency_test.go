package memory

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestMutationsRespectAtomicLockAndCancellation verifies every mutation waits for compound operations and rechecks cancellation. TestMutationsRespectAtomicLockAndCancellation 验证所有修改均等待复合操作并重新检查取消状态。
func TestMutationsRespectAtomicLockAndCancellation(t *testing.T) {
	operations := []struct {
		name string
		run  func(context.Context, *Storage) error
	}{
		{"set", func(ctx context.Context, s *Storage) error { return s.Set(ctx, "key", "new", 0) }},
		{"delete", func(ctx context.Context, s *Storage) error { return s.Delete(ctx, "key") }},
		{"clear", func(ctx context.Context, s *Storage) error { return s.Clear(ctx) }},
		{"expire", func(ctx context.Context, s *Storage) error { return s.Expire(ctx, "key", 0) }},
		{"get and delete", func(ctx context.Context, s *Storage) error {
			_, err := s.GetAndDelete(ctx, "key")
			return err
		}},
		{"set if absent", func(ctx context.Context, s *Storage) error {
			_, err := s.SetIfAbsent(ctx, "new-key", "new", 0)
			return err
		}},
	}
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			s := NewStorage()
			if err := s.Set(context.Background(), "key", "original", 0); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s.mu.Lock()
			locked := true
			defer func() {
				if locked {
					s.mu.Unlock()
				}
			}()
			started := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				close(started)
				done <- op.run(ctx, s)
			}()
			<-started
			select {
			case err := <-done:
				t.Fatalf("mutation bypassed an in-flight compound operation: %v", err)
			case <-time.After(25 * time.Millisecond):
			}

			cancel()
			s.mu.Unlock()
			locked = false
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("mutation error = %v, want context.Canceled", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("mutation did not finish after lock release")
			}
			if got, err := s.Get(context.Background(), "key"); err != nil || got != "original" {
				t.Fatalf("canceled mutation changed value: %v, %v", got, err)
			}
			if s.Exists(context.Background(), "new-key") {
				t.Fatal("canceled mutation created a key")
			}
		})
	}
}

// TestGetAndDeleteWithConcurrentSet verifies only serializable results are observable when a value is replaced. TestGetAndDeleteWithConcurrentSet 验证替换值时原子读删只产生可串行解释的结果。
func TestGetAndDeleteWithConcurrentSet(t *testing.T) {
	s := NewStorage()
	ctx := context.Background()
	for i := 0; i < 512; i++ {
		if err := s.Set(ctx, "key", "old", 0); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		written := make(chan error, 1)
		go func() {
			<-start
			written <- s.Set(ctx, "key", "new", 0)
		}()
		close(start)
		consumed, err := s.GetAndDelete(ctx, "key")
		if err != nil {
			t.Fatal(err)
		}
		if err := <-written; err != nil {
			t.Fatal(err)
		}
		remaining, err := s.Get(ctx, "key")
		if err != nil {
			t.Fatal(err)
		}
		if !((consumed == "old" && remaining == "new") || (consumed == "new" && remaining == nil)) {
			t.Fatalf("non-atomic result: consumed=%v remaining=%v", consumed, remaining)
		}
	}
}
