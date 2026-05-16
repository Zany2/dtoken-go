// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"sync"
	"time"
)

// loginLockEntry tracks one login lock and its active users. loginLockEntry 跟踪单个登录锁及其活跃使用者。
type loginLockEntry struct {
	mu   sync.Mutex // mu serializes writes for one login ID. mu 按登录 ID 串行化写操作。
	refs int        // refs tracks lock holders and waiters. refs 跟踪持锁者和等待者。
}

// lockLoginWrite locks write operations for one login ID lockLoginWrite 锁定指定账号的写操作
func (m *Manager) lockLoginWrite(loginID string) func() {
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

	entry.mu.Lock()
	return func() {
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

// submitAsync submits async work with goroutine fallback submitAsync 提交异步任务并在池不可用时回退到 goroutine
func (m *Manager) submitAsync(name string, task func()) {
	if m.pool == nil {
		go task()
		return
	}

	if err := m.pool.Submit(task); err != nil {
		m.logger.Errorf("manager.submitAsync: failed to submit async task, task=%s, error=%v", name, err)
		go task()
	}
}

// expireIfLimited renews key only when duration is limited expireIfLimited 仅在有限过期时间下续期 key
func (m *Manager) expireIfLimited(ctx context.Context, key string, expiration time.Duration) error {
	if expiration <= 0 {
		return nil
	}
	return m.storage.Expire(ctx, key, expiration)
}

// expireTokenIfLimited renews token key when expiration is limited. expireTokenIfLimited 在存在过期时间时续期 Token 键。
func (m *Manager) expireTokenIfLimited(ctx context.Context, tokenValue string, expiration time.Duration) error {
	if expiration <= 0 {
		return nil
	}
	key := m.getTokenKey(tokenValue)
	if !m.storage.Exists(ctx, key) {
		return nil
	}
	return m.expireIfLimited(ctx, key, expiration)
}

// rollbackLogin removes data written by a failed login rollbackLogin 回滚失败登录已写入的数据
func (m *Manager) rollbackLogin(ctx context.Context, sess *Session, loginID, token string, expiration time.Duration) {
	if sess != nil {
		if _, ok := sess.removeTerminalByToken(token); ok {
			if len(sess.TerminalInfos) == 0 {
				if err := m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
					m.logger.Errorf("manager.rollbackLogin: failed to delete empty session, loginID=%s, token=%s, error=%v", loginID, token, err)
				}
			} else {
				if err := m.saveSessionWithMinTTL(ctx, m.getSessionKey(loginID), *sess, expiration); err != nil {
					m.logger.Errorf("manager.rollbackLogin: failed to save session, loginID=%s, token=%s, error=%v", loginID, token, err)
				}
			}
		}
	}
	if err := m.storage.Delete(ctx, m.getTokenKey(token), m.getRenewKey(token), m.getActiveKey(token)); err != nil {
		m.logger.Errorf("manager.rollbackLogin: failed to delete token data, loginID=%s, token=%s, error=%v", loginID, token, err)
	}
}
