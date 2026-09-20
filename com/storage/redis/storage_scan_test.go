package redis

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

// TestKeysDeduplicatesScanPages verifies duplicate keys and empty intermediate pages do not corrupt scan results. TestKeysDeduplicatesScanPages 验证重复键和中间空页不影响扫描结果。
func TestKeysDeduplicatesScanPages(t *testing.T) {
	client := redisv9.NewClient(&redisv9.Options{})
	t.Cleanup(func() { _ = client.Close() })
	storage := &Storage{client: client, operationTimeout: time.Second}
	calls := 0
	client.AddHook(scanResultHook{process: func(ctx context.Context, cmd redisv9.Cmder) error {
		scan, ok := cmd.(*redisv9.ScanCmd)
		if !ok {
			t.Fatalf("unexpected command %T", cmd)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("SCAN lost the operation deadline")
		}
		pages := []struct {
			cursor, next uint64
			keys         []string
		}{
			{0, 7, []string{"a", "a", "b"}},
			{7, 9, nil},
			{9, 0, []string{"b", "c"}},
		}
		if calls >= len(pages) {
			t.Fatal("SCAN continued after cursor zero")
		}
		page := pages[calls]
		calls++
		if got := scan.Args()[1]; got != page.cursor {
			t.Fatalf("cursor = %v, want %d", got, page.cursor)
		}
		scan.SetVal(page.keys, page.next)
		return nil
	}})
	keys, err := storage.Keys(context.Background(), "*")
	if err != nil || !reflect.DeepEqual(keys, []string{"a", "b", "c"}) || calls != 3 {
		t.Fatalf("Keys() = %v, %v; calls = %d", keys, err, calls)
	}
}

// TestKeysReportsLaterScanFailure verifies partial results are not returned as a successful complete scan. TestKeysReportsLaterScanFailure 验证后续扫描失败不会被部分结果掩盖。
func TestKeysReportsLaterScanFailure(t *testing.T) {
	client := redisv9.NewClient(&redisv9.Options{})
	t.Cleanup(func() { _ = client.Close() })
	storage := NewStorageFromClient(client)
	sentinel := errors.New("scan failed")
	calls := 0
	client.AddHook(scanResultHook{process: func(_ context.Context, cmd redisv9.Cmder) error {
		calls++
		if calls == 1 {
			cmd.(*redisv9.ScanCmd).SetVal([]string{"partial"}, 3)
			return nil
		}
		return sentinel
	}})
	keys, err := storage.Keys(context.Background(), "*")
	if keys != nil || !errors.Is(err, sentinel) || calls != 2 {
		t.Fatalf("Keys() = %v, %v; calls = %d", keys, err, calls)
	}
}

// scanResultHook supplies command results without contacting a Redis server. scanResultHook 提供命令结果，无需连接 Redis 服务端。
type scanResultHook struct {
	process redisv9.ProcessHook
}

func (h scanResultHook) DialHook(next redisv9.DialHook) redisv9.DialHook {
	return next
}

func (h scanResultHook) ProcessHook(redisv9.ProcessHook) redisv9.ProcessHook {
	return h.process
}

func (h scanResultHook) ProcessPipelineHook(next redisv9.ProcessPipelineHook) redisv9.ProcessPipelineHook {
	return next
}
