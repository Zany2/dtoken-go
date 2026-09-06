// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"strings"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// SetSessionValue sets one session data value SetSessionValue 设置一个会话扩展数据
func (m *Manager) SetSessionValue(ctx context.Context, loginID, key string, value any) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize session data key before validation and storage 规范化会话扩展数据键后再校验和存储。
	key = strings.TrimSpace(key)
	if key == "" {
		return derror.ErrInvalidParam
	}
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}
	if sess.Data == nil {
		sess.Data = make(map[string]any)
	}
	sess.Data[key] = value
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// GetSessionValue gets one session data value GetSessionValue 获取一个会话扩展数据
func (m *Manager) GetSessionValue(ctx context.Context, loginID, key string) (any, bool, error) {
	if loginID == "" {
		return nil, false, derror.ErrIDIsEmpty
	}

	// Normalize session data key before lookup 规范化会话扩展数据键后再查询。
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false, derror.ErrInvalidParam
	}
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, false, err
	}
	if sess.Data == nil {
		return nil, false, nil
	}
	value, ok := sess.Data[key]
	return value, ok, nil
}

// DeleteSessionValue deletes one session data value DeleteSessionValue 删除一个会话扩展数据
func (m *Manager) DeleteSessionValue(ctx context.Context, loginID, key string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize session data key before deletion 规范化会话扩展数据键后再删除。
	key = strings.TrimSpace(key)
	if key == "" {
		return derror.ErrInvalidParam
	}
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}
	if sess.Data == nil {
		return nil
	}
	if _, exists := sess.Data[key]; !exists {
		return nil
	}
	delete(sess.Data, key)
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}

// SetSessionValueByToken sets session data after validating a token. SetSessionValueByToken 校验 Token 后设置会话扩展数据。
func (m *Manager) SetSessionValueByToken(ctx context.Context, tokenValue, key string, value any) error {
	// Validate and normalize token data key. 校验并规范化 Token 与会话数据键。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return derror.ErrInvalidParam
	}

	// Resolve the account before locking its session. 先解析账号，再锁定账号 Session。
	_, tokenInfo, err := m.checkLoginAndGetContextNoRenew(ctx, tokenValue)
	if err != nil {
		return err
	}
	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer func() { unlock() }()

	// Revalidate under the lock to prevent writes after logout. 锁内重新校验，避免登出后继续写入。
	sess, checkedInfo, err := m.checkLoginAndGetContextNoRenewLocked(ctx, tokenValue)
	if err != nil {
		// Publish a timeout discovered during locked revalidation only after unlocking. 锁内复核发现超时时，仅在解锁后发布事件。
		if errors.Is(err, derror.ErrActiveTimeout) && checkedInfo != nil {
			unlock()
			unlock = func() {}
			if sess != nil && len(sess.TerminalInfos) == 0 {
				m.triggerEvent(listener.EventDestroySession, checkedInfo.LoginID, "", "", "", nil)
			}
			m.triggerEvent(listener.EventActiveTimeout, checkedInfo.LoginID, checkedInfo.Device, checkedInfo.DeviceID, tokenValue, nil)
		}
		return err
	}
	if checkedInfo.LoginID != tokenInfo.LoginID {
		return derror.ErrInvalidToken
	}
	if sess.Data == nil {
		sess.Data = make(map[string]any)
	}
	sess.Data[key] = value
	return m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
}

// GetSessionValueByToken gets session data after validating a token. GetSessionValueByToken 校验 Token 后获取会话扩展数据。
func (m *Manager) GetSessionValueByToken(ctx context.Context, tokenValue, key string) (any, bool, error) {
	// Validate and normalize token data key. 校验并规范化 Token 与会话数据键。
	if tokenValue == "" {
		return nil, false, derror.ErrInvalidToken
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false, derror.ErrInvalidParam
	}

	// Use the session loaded by full token validation. 使用完整 Token 校验加载的 Session。
	sess, _, err := m.checkLoginAndGetContextNoRenew(ctx, tokenValue)
	if err != nil {
		return nil, false, err
	}
	if sess.Data == nil {
		return nil, false, nil
	}
	value, ok := sess.Data[key]
	return value, ok, nil
}

// DeleteSessionValueByToken deletes session data after validating a token. DeleteSessionValueByToken 校验 Token 后删除会话扩展数据。
func (m *Manager) DeleteSessionValueByToken(ctx context.Context, tokenValue, key string) error {
	// Validate and normalize token data key. 校验并规范化 Token 与会话数据键。
	if tokenValue == "" {
		return derror.ErrInvalidToken
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return derror.ErrInvalidParam
	}

	// Resolve the account before locking its session. 先解析账号，再锁定账号 Session。
	_, tokenInfo, err := m.checkLoginAndGetContextNoRenew(ctx, tokenValue)
	if err != nil {
		return err
	}
	unlock := m.lockLoginWrite(tokenInfo.LoginID)
	defer func() { unlock() }()

	// Revalidate under the lock to prevent writes after logout. 锁内重新校验，避免登出后继续写入。
	sess, checkedInfo, err := m.checkLoginAndGetContextNoRenewLocked(ctx, tokenValue)
	if err != nil {
		// Publish a timeout discovered during locked revalidation only after unlocking. 锁内复核发现超时时，仅在解锁后发布事件。
		if errors.Is(err, derror.ErrActiveTimeout) && checkedInfo != nil {
			unlock()
			unlock = func() {}
			if sess != nil && len(sess.TerminalInfos) == 0 {
				m.triggerEvent(listener.EventDestroySession, checkedInfo.LoginID, "", "", "", nil)
			}
			m.triggerEvent(listener.EventActiveTimeout, checkedInfo.LoginID, checkedInfo.Device, checkedInfo.DeviceID, tokenValue, nil)
		}
		return err
	}
	if checkedInfo.LoginID != tokenInfo.LoginID {
		return derror.ErrInvalidToken
	}
	if sess.Data == nil {
		return nil
	}
	if _, exists := sess.Data[key]; !exists {
		return nil
	}
	delete(sess.Data, key)
	return m.saveToStorage(ctx, m.getSessionKey(sess.LoginID), *sess)
}
