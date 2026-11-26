package utils

import (
	"sync"
	"testing"
)

// TestKeyedMutexSerializesAndReleasesEntries checks shared-key exclusion and cleanup after concurrent callers. TestKeyedMutexSerializesAndReleasesEntries 验证同键互斥及并发调用后的条目回收。
func TestKeyedMutexSerializesAndReleasesEntries(t *testing.T) {
	var locks KeyedMutex
	var wg sync.WaitGroup
	count := 0
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				unlock := locks.Lock("shared")
				count++
				unlock()
			}
		}()
	}
	wg.Wait()
	if count != 640 || len(locks.entries) != 0 {
		t.Fatalf("count/remaining locks = %d/%d, want 640/0", count, len(locks.entries))
	}
}
