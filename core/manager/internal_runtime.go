// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
)

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

// submitAsync submits async work with a goroutine fallback submitAsync 提交异步任务并在池不可用时回退到 goroutine
func (m *Manager) submitAsync(name string, task func()) {
	// Register the task unless manager shutdown has started. Manager 开始关闭后不再接收新任务。
	m.asyncMu.Lock()
	if m.asyncClosed {
		m.asyncMu.Unlock()
		return
	}
	m.asyncWG.Add(1)
	pool := m.pool
	m.asyncMu.Unlock()

	// Track task completion independently from pool ownership. 独立于协程池所有权跟踪任务完成状态。
	trackedTask := func() {
		defer m.asyncWG.Done()
		task()
	}

	// Fallback when pool is absent 协程池不存在时回退。
	if pool == nil {
		go trackedTask()
		return
	}

	// Submit task to pool 提交任务到协程池。
	if err := pool.Submit(trackedTask); err != nil {
		m.logger.Errorf("manager.submitAsync: failed to submit async task, task=%s, error=%v", name, err)

		// Fallback when submit fails 提交失败时回退。
		go trackedTask()
	}
}

// expireIfLimited renews a key only when duration is limited expireIfLimited 仅在有限过期时间下续期存储键
func (m *Manager) expireIfLimited(ctx context.Context, key string, expiration time.Duration) error {
	// Skip unlimited expiration 跳过无限有效期。
	if expiration <= 0 {
		return nil
	}

	// Renew key expiration 续期键过期时间。
	if err := m.storage.Expire(ctx, key, expiration); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// expireTokenIfLimited renews token key when expiration is limited. expireTokenIfLimited 在存在过期时间时续期 Token 键。
func (m *Manager) expireTokenIfLimited(ctx context.Context, tokenValue string, expiration time.Duration) error {
	// Skip unlimited expiration 跳过无限有效期。
	if expiration <= 0 {
		return nil
	}

	// Build token key 构建 Token 键。
	key := m.getTokenKey(tokenValue)

	// Skip missing token key 跳过不存在的 Token 键。
	if !m.storage.Exists(ctx, key) {
		return nil
	}

	// Renew token key expiration 续期 Token 键过期时间。
	return m.expireIfLimited(ctx, key, expiration)
}
