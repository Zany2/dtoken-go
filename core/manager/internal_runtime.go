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

// loginMaintenanceState tracks one token maintenance generation and its latest activity. loginMaintenanceState 跟踪 Token 维护任务代次及最新活跃时间。
type loginMaintenanceState struct {
	generation uint64 // generation identifies the token lifecycle task. generation 标识 Token 生命周期任务。
	activeAt   int64  // activeAt stores the latest validation timestamp. activeAt 存储最近一次校验时间戳。
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

// submitAsync submits async work with a goroutine fallback and reports admission. submitAsync 提交异步任务并在池不可用时回退到 goroutine，同时返回是否已接收。
func (m *Manager) submitAsync(name string, task func()) bool {
	// Register the task unless manager shutdown has started. Manager 开始关闭后不再接收新任务。
	m.asyncMu.Lock()
	if m.asyncClosed {
		m.asyncMu.Unlock()
		return false
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
		return true
	}

	// Submit task to pool 提交任务到协程池。
	if err := pool.Submit(trackedTask); err != nil {
		m.logger.Errorf("manager.submitAsync: failed to submit async task, task=%s, error=%v", name, err)

		// Fallback when submit fails 提交失败时回退。
		go trackedTask()
	}
	return true
}

// beginLoginMaintenance reserves or replaces one in-flight maintenance generation for a token. beginLoginMaintenance 为 Token 登记或替换一个执行中的维护任务代次。
func (m *Manager) beginLoginMaintenance(tokenValue string, activeAt int64, replace bool) (uint64, bool) {
	if tokenValue == "" || m.closed.Load() {
		return 0, false
	}

	// Coalesce concurrent maintenance requests for the same token. 合并同一 Token 的并发维护请求。
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	if state, exists := m.maintenance[tokenValue]; exists && !replace {
		// Preserve the latest request activity while sharing the existing task. 复用已有任务时保留最近一次请求活跃时间。
		if activeAt > state.activeAt {
			state.activeAt = activeAt
			m.maintenance[tokenValue] = state
		}
		return 0, false
	}
	if m.maintenance == nil {
		m.maintenance = make(map[string]loginMaintenanceState)
	}
	m.maintenanceSeq++
	generation := m.maintenanceSeq
	m.maintenance[tokenValue] = loginMaintenanceState{generation: generation, activeAt: activeAt}
	return generation, true
}

// isLoginMaintenanceCurrent reports whether a token task still owns its generation. isLoginMaintenanceCurrent 判断 Token 维护任务是否仍持有当前代次。
func (m *Manager) isLoginMaintenanceCurrent(tokenValue string, generation uint64) bool {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	state, exists := m.maintenance[tokenValue]
	return exists && generation != 0 && state.generation == generation
}

// getLoginMaintenanceActiveAt returns the latest activity owned by a task generation. getLoginMaintenanceActiveAt 返回任务代次持有的最新活跃时间。
func (m *Manager) getLoginMaintenanceActiveAt(tokenValue string, generation uint64) (int64, bool) {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	state, exists := m.maintenance[tokenValue]
	if !exists || generation == 0 || state.generation != generation {
		return 0, false
	}
	return state.activeAt, true
}

// finishLoginMaintenanceActiveWrite completes a stable activity write. finishLoginMaintenanceActiveWrite 完成稳定的活跃写入。
func (m *Manager) finishLoginMaintenanceActiveWrite(tokenValue string, generation uint64, writtenAt int64) bool {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	state, exists := m.maintenance[tokenValue]
	if !exists || generation == 0 || state.generation != generation {
		return true
	}
	if state.activeAt > writtenAt {
		return false
	}

	// Delete atomically with the stable check so a later request creates a new task. 与稳定性检查原子删除，使后续请求能够创建新任务。
	delete(m.maintenance, tokenValue)
	return true
}

// finishLoginMaintenance releases a completed token maintenance generation. finishLoginMaintenance 释放已完成的 Token 维护任务代次。
func (m *Manager) finishLoginMaintenance(tokenValue string, generation uint64) {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	if state, exists := m.maintenance[tokenValue]; exists && state.generation == generation {
		delete(m.maintenance, tokenValue)
	}
}

// cancelLoginMaintenance invalidates pending maintenance for a token lifecycle. cancelLoginMaintenance 使 Token 当前生命周期的待执行维护任务失效。
func (m *Manager) cancelLoginMaintenance(tokenValue string) {
	if tokenValue == "" {
		return
	}
	m.maintenanceMu.Lock()
	delete(m.maintenance, tokenValue)
	m.maintenanceMu.Unlock()
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
