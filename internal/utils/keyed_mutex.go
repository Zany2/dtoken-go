package utils

import "sync"

// KeyedMutex serializes local work per key and releases idle entries; its zero value is ready to use and it must not be copied after use. KeyedMutex 按键串行化本地操作并释放空闲条目；零值可用，使用后不可复制。
type KeyedMutex struct {
	mu      sync.Mutex // mu protects entry membership and reference counts. mu 保护条目注册与引用计数。
	entries map[string]*keyedMutexEntry
}

type keyedMutexEntry struct {
	mu   sync.Mutex
	refs int // refs includes holders and waiters. refs 包含持锁者与等待者。
}

// Lock acquires one key and returns an unlock function that must be called exactly once. Lock 获取指定键的锁，返回必须恰好调用一次的解锁函数。
func (m *KeyedMutex) Lock(key string) func() {
	m.mu.Lock()
	if m.entries == nil {
		m.entries = make(map[string]*keyedMutexEntry)
	}
	entry := m.entries[key]
	if entry == nil {
		entry = &keyedMutexEntry{}
		m.entries[key] = entry
	}
	entry.refs++
	m.mu.Unlock()

	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		m.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(m.entries, key)
		}
		m.mu.Unlock()
	}
}
