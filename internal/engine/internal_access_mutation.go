package engine

import (
	"context"
	"errors"

	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// accessChange binds session mutation to its existing event contract. accessChange 将会话变更绑定到原有事件契约。
type accessChange struct {
	apply  func(*Session, ...string) []string // apply returns values actually changed. apply 返回实际变更值。
	event  listener.Event                     // event identifies permission or role changes. event 标识权限或角色变更。
	key    string                             // key identifies the changed values in event data. key 标识事件中的变更值。
	action string                             // action identifies addition or removal. action 标识新增或移除。
}

// mutateAccessByLoginID persists one account mutation and dispatches after unlocking. mutateAccessByLoginID 保存账号变更并在解锁后分发事件。
func (m *Manager) mutateAccessByLoginID(ctx context.Context, loginID string, values []string, change accessChange) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize before resolving account state. 解析账号状态前规范化变更值。
	values = normalizeAccessValues(values)
	if len(values) == 0 {
		return nil
	}

	// Serialize writes for the resolved account. 串行化目标账号的写操作。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Load and mutate the current session. 加载并修改当前会话。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}
	changed := change.apply(sess, values...)
	if len(changed) == 0 {
		return nil
	}

	// Persist only actual changes before reporting them. 仅持久化实际变更，完成后再报告。
	if err = m.saveToStorage(ctx, m.getSessionKey(loginID), *sess); err != nil {
		return err
	}

	// Call listeners outside the account lock. 在账号锁外调用监听器。
	unlock()
	unlock = func() {}
	m.triggerEvent(change.event, loginID, "", "", "", map[string]any{
		change.key:              changed,
		listener.ExtraKeyAction: change.action,
	})
	return nil
}

// mutateAccessByToken revalidates the token under its account lock before mutation. mutateAccessByToken 在账号锁内复核 Token 后执行变更。
func (m *Manager) mutateAccessByToken(ctx context.Context, tokenValue string, values []string, change accessChange) error {
	// Normalize before resolving account state. 解析账号状态前规范化变更值。
	values = normalizeAccessValues(values)

	// Capture the lifecycle before acquiring its account lock. 获取账号锁前捕获 Token 生命周期。
	expectedRecord, err := m.getTokenRecord(ctx, tokenValue)
	if err != nil {
		return err
	}

	// Serialize writes for the resolved account. 串行化目标账号的写操作。
	unlock := m.lockLoginWrite(expectedRecord.LoginID)

	// Schedule maintenance after unlocking on every return path. 所有返回路径均在解锁后提交维护任务。
	var submitMaintenance func()
	defer func() {
		unlock()
		if submitMaintenance != nil {
			submitMaintenance()
		}
	}()

	// Revalidate against the captured lifecycle while holding the lock. 持锁时按已捕获生命周期复核。
	sess, tokenInfo, err := m.checkLoginAndGetContextNoRenewLocked(ctx, tokenValue, expectedRecord)
	if err != nil {
		if errors.Is(err, derror.ErrActiveTimeout) && tokenInfo != nil {
			unlock()
			unlock = func() {}
			if sess != nil && len(sess.TerminalInfos) == 0 {
				m.triggerEvent(listener.EventDestroySession, tokenInfo.LoginID, "", "", "", nil)
			}
			m.triggerEvent(listener.EventActiveTimeout, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceID, tokenValue, nil)
		}
		return err
	}

	// Empty changes validate login without scheduling renewal. 空变更校验登录态后直接返回。
	if len(values) == 0 {
		return nil
	}

	submitMaintenance = m.prepareLoginMaintenance(ctx, tokenValue, expectedRecord, m.resolveActiveTimeoutFromSeconds(tokenInfo.ActiveTimeout))
	changed := change.apply(sess, values...)
	if len(changed) == 0 {
		return nil
	}

	// Persist only actual changes before reporting them. 仅持久化实际变更，完成后再报告。
	if err = m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess); err != nil {
		return err
	}

	// Call listeners outside the account lock. 在账号锁外调用监听器。
	unlock()
	unlock = func() {}
	m.triggerEvent(change.event, sess.LoginID, tokenInfo.Device, tokenInfo.DeviceID, tokenValue, map[string]any{
		change.key:              changed,
		listener.ExtraKeyAction: change.action,
	})
	return nil
}
