package migration

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// plainStorage hides atomic capabilities and allows deterministic storage failures. plainStorage 隐藏原子能力并模拟确定性的存储故障。
type plainStorage struct {
	adapter.Storage
	scan adapter.ScannerStorage
	keys func(context.Context, string) ([]string, error)
	get  func(context.Context, string) (any, error)
	ttl  func(context.Context, string) (time.Duration, error)
	set  func(context.Context, string, any, time.Duration) error
}

func (s *plainStorage) Keys(ctx context.Context, pattern string) ([]string, error) {
	if s.keys != nil {
		return s.keys(ctx, pattern)
	}
	return s.scan.Keys(ctx, pattern)
}

func (s *plainStorage) Get(ctx context.Context, key string) (any, error) {
	if s.get != nil {
		return s.get(ctx, key)
	}
	return s.Storage.Get(ctx, key)
}

func (s *plainStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if s.ttl != nil {
		return s.ttl(ctx, key)
	}
	return s.Storage.TTL(ctx, key)
}

func (s *plainStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.set != nil {
		return s.set(ctx, key, value, ttl)
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

type refusingStorage struct{ *memory.Storage }

func (s *refusingStorage) SetIfAbsent(context.Context, string, any, time.Duration) (bool, error) {
	return false, nil
}

func copyOptions() Options {
	return Options{AuthType: "user", SourcePrefix: "old:", TargetPrefix: "new:"}
}

func put(t *testing.T, store adapter.Storage, key string, value any, ttl time.Duration) {
	t.Helper()
	if err := store.Set(context.Background(), key, value, ttl); err != nil {
		t.Fatal(err)
	}
}

func TestCopyPreservesValuesAndLifetime(t *testing.T) {
	for _, atomic := range []bool{false, true} {
		name := "plain"
		if atomic {
			name = "atomic"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			base := memory.NewStorage()
			var store adapter.Storage = base
			if !atomic {
				store = &plainStorage{Storage: base, scan: base}
			}
			payload := []byte("encoded token")
			put(t, base, "old:user:token:a", payload, time.Hour)
			put(t, base, "old:user:session:1", "session", 0)
			put(t, base, "old:user:active:a", int64(123), time.Hour)
			put(t, base, "old:admin:token:b", "outside", 0)
			plan, err := Prepare(ctx, store, copyOptions())
			if err != nil {
				t.Fatal(err)
			}
			if got := plan.Summary(); got.Records != 3 || got.Permanent != 1 || got.Expiring != 2 || got.SourceNamespace != "old:user:" || got.TargetNamespace != "new:user:" {
				t.Fatalf("summary = %+v", got)
			}
			keys, err := base.Keys(ctx, "*")
			if err != nil || len(keys) != 4 {
				t.Fatalf("Prepare changed storage: keys=%v err=%v", keys, err)
			}

			// Mutating the borrowed source buffer must not mutate the prepared snapshot. 修改源缓冲区不能改变已捕获的快照。
			payload[0] = 'X'
			for i := range plan.records {
				if !plan.records[i].deadline.IsZero() {
					plan.records[i].deadline = time.Now().Add(30 * time.Minute)
				}
			}
			result, err := plan.Apply(ctx)
			if err != nil || result != (Result{Written: 3}) {
				t.Fatalf("Apply = %+v, %v", result, err)
			}
			for key, want := range map[string]any{"token:a": []byte("encoded token"), "session:1": "session", "active:a": int64(123)} {
				got, err := base.Get(ctx, "new:user:"+key)
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("%s = %v, %v", key, got, err)
				}
				if !base.Exists(ctx, "old:user:"+key) {
					t.Fatalf("source deleted: %s", key)
				}
			}
			if base.Exists(ctx, "new:admin:token:b") {
				t.Fatal("copied another auth type")
			}
			ttl, err := base.TTL(ctx, "new:user:token:a")
			if err != nil || ttl <= 0 || ttl > 30*time.Minute {
				t.Fatalf("remaining TTL = %v, %v", ttl, err)
			}
			ttl, err = base.TTL(ctx, "new:user:session:1")
			if err != nil || ttl != adapter.TTLNoExpire {
				t.Fatalf("permanent TTL = %v, %v", ttl, err)
			}
			if _, err := plan.Apply(ctx); !errors.Is(err, ErrPlanUsed) {
				t.Fatalf("replay = %v", err)
			}
		})
	}
}

func TestApplySkipsExpiredSnapshot(t *testing.T) {
	ctx := context.Background()
	base := memory.NewStorage()
	put(t, base, "old:user:token:a", "token", time.Hour)
	plan, err := Prepare(ctx, base, copyOptions())
	if err != nil {
		t.Fatal(err)
	}
	plan.records[0].deadline = time.Now().Add(-time.Second)
	result, err := plan.Apply(ctx)
	if err != nil || result != (Result{Expired: 1}) || base.Exists(ctx, "new:user:token:a") {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
}

func TestDestinationConflicts(t *testing.T) {
	ctx := context.Background()
	for _, stage := range []string{"prepare", "apply", "atomic", "plain"} {
		t.Run(stage, func(t *testing.T) {
			base := memory.NewStorage()
			put(t, base, "old:user:token:a", "source", 0)
			var store adapter.Storage = base
			if stage == "atomic" {
				store = &refusingStorage{base}
			}
			if stage == "plain" {
				store = &plainStorage{Storage: base, scan: base, get: func(ctx context.Context, key string) (any, error) {
					if key == "new:user:token:a" {
						return "occupied", nil
					}
					return base.Get(ctx, key)
				}}
			}
			if stage == "prepare" {
				put(t, base, "new:user:unexpected", "keep", 0)
			}
			plan, err := Prepare(ctx, store, copyOptions())
			if stage == "prepare" {
				if !errors.Is(err, ErrDestinationNotEmpty) {
					t.Fatalf("Prepare = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if stage == "apply" {
				put(t, base, "new:user:unexpected", "keep", 0)
			}
			result, err := plan.Apply(ctx)
			if !errors.Is(err, ErrDestinationNotEmpty) || result.Written != 0 {
				t.Fatalf("Apply = %+v, %v", result, err)
			}
			if base.Exists(ctx, "new:user:token:a") {
				t.Fatal("conflicting write accepted")
			}
			if stage == "apply" {
				value, _ := base.Get(ctx, "new:user:unexpected")
				if value != "keep" {
					t.Fatal("destination modified")
				}
			}
		})
	}
}

func TestPartialFailureCannotReplay(t *testing.T) {
	ctx := context.Background()
	base := memory.NewStorage()
	failure := errors.New("write failed")
	put(t, base, "old:user:a", "first", 0)
	put(t, base, "old:user:b", "second", 0)
	store := &plainStorage{Storage: base, scan: base, set: func(ctx context.Context, key string, value any, ttl time.Duration) error {
		if key == "new:user:b" {
			return failure
		}
		return base.Set(ctx, key, value, ttl)
	}}
	plan, err := Prepare(ctx, store, copyOptions())
	if err != nil {
		t.Fatal(err)
	}
	result, err := plan.Apply(ctx)
	if !errors.Is(err, failure) || result.Written != 1 {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
	if !base.Exists(ctx, "new:user:a") || !base.Exists(ctx, "old:user:a") || !base.Exists(ctx, "old:user:b") {
		t.Fatal("unexpected rollback or source mutation")
	}
	if _, err := plan.Apply(ctx); !errors.Is(err, ErrPlanUsed) {
		t.Fatalf("retry = %v", err)
	}
	if plan.records != nil {
		t.Fatal("snapshot retained after failure")
	}
}

func TestPrepareSkipsMissingAndRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ttl     time.Duration
		value   any
		invalid bool
	}{
		{"missing", adapter.TTLNotFound, "unused", false},
		{"zero", 0, "unused", false},
		{"vanished", time.Hour, nil, false},
		{"elapsed", time.Millisecond, "unused", false},
		{"invalid ttl", -3, "unused", true},
		{"mutable", adapter.TTLNoExpire, map[string]string{"a": "b"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := memory.NewStorage()
			store := &plainStorage{Storage: base, scan: base,
				keys: func(context.Context, string) ([]string, error) { return []string{"old:user:a", "old:user:a"}, nil },
				ttl:  func(context.Context, string) (time.Duration, error) { return tc.ttl, nil },
				get: func(context.Context, string) (any, error) {
					// Simulate a read that outlives the reported TTL. 模拟读取耗时超过返回的 TTL。
					if tc.name == "elapsed" {
						time.Sleep(tc.ttl)
					}
					return tc.value, nil
				},
			}
			plan, err := Prepare(context.Background(), store, copyOptions())
			if tc.invalid {
				if !errors.Is(err, derror.ErrInvalidParam) {
					t.Fatalf("Prepare = %v", err)
				}
			} else if err != nil || plan.Summary().Skipped != 1 || plan.Summary().Records != 0 {
				t.Fatalf("Prepare = %+v, %v", plan, err)
			}
		})
	}
}

func TestPrepareValidationAndCancellation(t *testing.T) {
	ctx := context.Background()
	base := memory.NewStorage()
	for _, opts := range []Options{{}, {SourcePrefix: "old", TargetPrefix: " old: "}} {
		if _, err := Prepare(ctx, base, opts); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("invalid options = %v", err)
		}
	}
	if _, err := Prepare(ctx, nil, copyOptions()); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("nil storage = %v", err)
	}
	if _, err := Prepare(ctx, struct{ adapter.Storage }{base}, copyOptions()); !errors.Is(err, derror.ErrStorageCapabilityUnsupported) {
		t.Fatalf("missing scanner = %v", err)
	}
	plan, err := Prepare(ctx, base, copyOptions())
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Prepare(cancelled, base, copyOptions()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled prepare = %v", err)
	}
	if result, err := plan.Apply(cancelled); !errors.Is(err, context.Canceled) || result.Written != 0 {
		t.Fatalf("cancelled Apply = %+v, %v", result, err)
	}
	if _, err := new(Plan).Apply(ctx); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("zero plan = %v", err)
	}
}

func TestLiteralNamespaceAndDefaultAuthType(t *testing.T) {
	ctx := context.Background()
	base := memory.NewStorage()
	put(t, base, "old*:auth:token:a", "selected", 0)
	put(t, base, "oldX:auth:token:b", "outside", 0)
	plan, err := Prepare(ctx, base, Options{SourcePrefix: " old*: ", TargetPrefix: "new:"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Summary().Records != 1 {
		t.Fatalf("summary = %+v", plan.Summary())
	}
	if result, err := plan.Apply(ctx); err != nil || result.Written != 1 {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
	if base.Exists(ctx, "new:auth:token:b") {
		t.Fatal("glob expanded namespace")
	}
}
