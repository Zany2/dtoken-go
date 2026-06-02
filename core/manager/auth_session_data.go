// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
)

// SetSessionValue sets one session data value SetSessionValue 设置一个会话扩展数据
func (m *Manager) SetSessionValue(ctx context.Context, loginID, key string, value any) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
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
	if key == "" {
		return derror.ErrInvalidParam
	}
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return err
	}
	if sess.Data != nil {
		delete(sess.Data, key)
	}
	return m.saveToStorage(ctx, m.getSessionKey(loginID), *sess)
}
