// Author records daixk as original author at 2026/1/22 17:33:00. Author 记录 daixk 为原始作者，创建时间为 2026/1/22 17:33:00。
package manager

import (
	"context"
	"sync"
	"time"
)

// lockLoginWrite locks write operations for one login ID lockLoginWrite 锁定指定账号的写操作
func (m *Manager) lockLoginWrite(loginID string) func() {
	if loginID == "" {
		return func() {}
	}

	value, _ := m.loginLocks.LoadOrStore(loginID, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
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

// expireTokenIfLimited renews current or legacy token key. expireTokenIfLimited 续期当前或历史 Token 键。
func (m *Manager) expireTokenIfLimited(ctx context.Context, tokenValue string, expiration time.Duration) error {
	if expiration <= 0 {
		return nil
	}
	for _, key := range m.getTokenStorageKeys(tokenValue) {
		if !m.storage.Exists(ctx, key) {
			continue
		}
		return m.expireIfLimited(ctx, key, expiration)
	}
	return nil
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
	if err := m.storage.Delete(ctx, append(m.getTokenStorageKeys(token), m.getRenewKey(token), m.getActiveKey(token))...); err != nil {
		m.logger.Errorf("manager.rollbackLogin: failed to delete token data, loginID=%s, token=%s, error=%v", loginID, token, err)
	}
}
