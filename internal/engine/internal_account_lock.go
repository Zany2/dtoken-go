package engine

import "sync"

// managerAccounts owns account lock entries independently of token maintenance. managerAccounts 独立持有账号锁条目。
type managerAccounts struct {
	loginLocksMu sync.Mutex                 // loginLocksMu protects the login lock registry. loginLocksMu 保护登录锁注册表。
	loginLocks   map[string]*loginLockEntry // loginLocks serializes writes per login ID. loginLocks 按登录 ID 串行化写操作。
}

// loginLockEntry tracks one login lock and its active users. loginLockEntry 跟踪单个登录锁及其活跃使用者。
type loginLockEntry struct {
	mu   sync.Mutex // mu serializes writes for one login ID. mu 按登录 ID 串行化写操作。
	refs int        // refs tracks lock holders and waiters. refs 跟踪持锁者和等待者。
}

// lockLoginWrite locks write operations for one login ID lockLoginWrite 锁定指定账号的写操作
func (m *Manager) lockLoginWrite(loginID string) func() {
	// Return no-op unlock for empty ID ID 为空时返回空解锁函数
	if loginID == "" {
		return func() {}
	}

	// Get or create one shared lock entry. 获取或创建共享锁条目。
	m.loginLocksMu.Lock()
	if m.loginLocks == nil {
		m.loginLocks = make(map[string]*loginLockEntry)
	}
	entry, ok := m.loginLocks[loginID]
	if !ok {
		entry = &loginLockEntry{}
		m.loginLocks[loginID] = entry
	}
	entry.refs++
	m.loginLocksMu.Unlock()

	// Lock account entry 锁定账号条目。
	entry.mu.Lock()
	return func() {
		// Unlock account entry 解锁账号条目。
		entry.mu.Unlock()

		// Release registry entry after the last waiter leaves. 最后一个等待者离开后释放注册表条目。
		m.loginLocksMu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(m.loginLocks, loginID)
		}
		m.loginLocksMu.Unlock()
	}
}
