// Author records daixk as original author at 2026/1/22 17:33:00. Author 记录 daixk 为原始作者，创建时间为 2026/1/22 17:33:00。
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/utils"
	"time"
)

// Disable disables an account for a specified duration. Disable 封禁账号指定时长。
func (m *Manager) Disable(ctx context.Context, loginID string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer unlock()

	// Load session before disable 先尝试加载 Session（如果存储出错，在保存封禁信息前就返回，保证原子性）
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session and return other storage errors 如果只是 session 不存在，不算错误；其他存储错误则返回
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return err
		}
		// Continue disable when sess is nil 否则 sess == nil，继续执行封禁操作（幂等）
	}

	// Build and save disable info 构建并保存封禁信息
	disableInfo := DisableInfo{
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		disableInfo.DisableReason = reason[0]
	}

	if err = m.saveToStorage(ctx, m.getDisableKey(loginID), disableInfo, duration); err != nil {
		return err
	}

	// Delete session 删除 Session
	if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Clean related token data when terminals exist 如果有终端信息，清理所有相关 token 数据
	if sess != nil && len(sess.TerminalInfos) > 0 {
		tokens := make([]string, len(sess.TerminalInfos))
		tokenKeys := make([]string, 0, len(sess.TerminalInfos)*2)
		for i, info := range sess.TerminalInfos {
			tokens[i] = info.Token
			tokenKeys = append(tokenKeys, m.getTokenStorageKeys(info.Token)...)
		}

		// Delete primary token keys 删除主 token keys
		if err = m.storage.Delete(ctx, tokenKeys...); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}

		// Clean token metadata 清理附属 metadata（续期、活跃时间）
		if err = m.cleanTokenMetadata(ctx, tokens); err != nil {
			return err
		}
	}

	unlock()
	unlock = func() {}

	if sess != nil {
		// Trigger session destroy event 触发销毁 Session 事件
		m.triggerEvent(listener.EventDestroySession, loginID, "", "", "", nil)
	}

	// Trigger disable event 触发封禁事件
	m.triggerEvent(listener.EventDisable, loginID, "", "", "", map[string]any{
		"reason":   disableInfo.DisableReason,
		"duration": duration.Seconds(),
	})

	return nil
}

// Untie removes the disable status from an account. Untie 解封账号。
func (m *Manager) Untie(ctx context.Context, loginID string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	if err := m.storage.Delete(ctx, m.getDisableKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Trigger untie event 触发解禁事件
	m.triggerEvent(listener.EventUntie, loginID, "", "", "", nil)

	return nil
}

// IsDisable checks if an account is disabled. IsDisable 检查账号是否被封禁。
func (m *Manager) IsDisable(ctx context.Context, loginID string) bool {
	return m.isDisable(ctx, loginID)
}

// GetDisableInfo retrieves disable information for an account. GetDisableInfo 获取账号的封禁信息。
func (m *Manager) GetDisableInfo(ctx context.Context, loginID string) (*DisableInfo, error) {
	disableInfoData, err := m.storage.Get(ctx, m.getDisableKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Return explicit error when disable key is missing 如果 key 不存在（用户未被封禁），返回明确的错误
	if disableInfoData == nil {
		return nil, derror.ErrAccountNotDisabled
	}

	bytesData, err := utils.ToBytes(disableInfoData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var disableInfo DisableInfo
	if err = m.serializer.Decode(bytesData, &disableInfo); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &disableInfo, nil
}

// GetDisableTTL retrieves the remaining disable time for an account in seconds. GetDisableTTL 获取账号剩余封禁时间（秒）。 Returns: -2: account is not disabled (未封禁) -1: account is permanently disabled (永久封禁) >0: remaining seconds until unban (剩余封禁秒数)
func (m *Manager) GetDisableTTL(ctx context.Context, loginID string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, m.getDisableKey(loginID))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Explain TTL semantics 存储适配器返回 time.Duration 类型，直接转换为 int64 即可，标准 Redis TTL 语义：-2 key 不存在，-1 key 无过期时间，>0 剩余秒数
	seconds := int64(ttl)

	switch {
	case ttl == adapter.TTLNotFound:
		return -2, nil // 未封禁
	case ttl == adapter.TTLNoExpire:
		return -1, nil // 永久封禁
	case seconds > 0:
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
}

// DisableService disables a specific service for an account. DisableService 封禁账号的指定服务。
func (m *Manager) DisableService(ctx context.Context, loginID, service string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if service == "" {
		return derror.ErrInvalidParam
	}

	info := ServiceDisableInfo{
		Service:     service,
		Level:       0,
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	if err := m.saveToStorage(ctx, m.getDisableServiceKey(loginID, service), info, duration); err != nil {
		return err
	}

	m.triggerEvent(listener.EventDisableService, loginID, "", "", "", map[string]any{
		listener.ExtraKeyService: service,
		"reason":                 info.DisableReason,
		"duration":               duration.Seconds(),
	})

	return nil
}

// DisableServiceLevel disables a specific service for an account with a level. DisableServiceLevel 封禁账号的指定服务并设置封禁等级。
func (m *Manager) DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if service == "" {
		return derror.ErrInvalidParam
	}

	info := ServiceDisableInfo{
		Service:     service,
		Level:       level,
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	if err := m.saveToStorage(ctx, m.getDisableServiceKey(loginID, service), info, duration); err != nil {
		return err
	}

	m.triggerEvent(listener.EventDisableService, loginID, "", "", "", map[string]any{
		listener.ExtraKeyService: service,
		listener.ExtraKeyLevel:   level,
		"reason":                 info.DisableReason,
		"duration":               duration.Seconds(),
	})

	return nil
}

// UntieService removes the disable status of a specific service for an account. UntieService 解封账号的指定服务。
func (m *Manager) UntieService(ctx context.Context, loginID, service string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if service == "" {
		return derror.ErrInvalidParam
	}

	if err := m.storage.Delete(ctx, m.getDisableServiceKey(loginID, service)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	m.triggerEvent(listener.EventUntieService, loginID, "", "", "", map[string]any{
		listener.ExtraKeyService: service,
	})

	return nil
}

// IsDisableService checks if a specific service is disabled for an account. IsDisableService 检查账号的指定服务是否被封禁。
func (m *Manager) IsDisableService(ctx context.Context, loginID, service string) bool {
	if loginID == "" || service == "" {
		return false
	}
	return m.storage.Exists(ctx, m.getDisableServiceKey(loginID, service))
}

// IsDisableServiceLevel checks if a specific service is disabled at or above the given level. IsDisableServiceLevel 检查账号的指定服务是否达到指定封禁等级。
func (m *Manager) IsDisableServiceLevel(ctx context.Context, loginID, service string, level int) bool {
	info, err := m.GetDisableServiceInfo(ctx, loginID, service)
	if err != nil {
		return false
	}
	return info.Level >= level
}

// CheckDisableService checks if any of the specified services are disabled, returns error if disabled. CheckDisableService 校验账号的指定服务是否被封禁，被封禁则返回 error。
func (m *Manager) CheckDisableService(ctx context.Context, loginID string, services ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	for _, service := range services {
		if m.IsDisableService(ctx, loginID, service) {
			return fmt.Errorf("%w: service=%s", derror.ErrServiceDisabled, service)
		}
	}
	return nil
}

// CheckDisableServiceLevel checks if a service is disabled at or above the given level, returns error if so. CheckDisableServiceLevel 校验账号的指定服务是否达到指定封禁等级，达到则返回 error。
func (m *Manager) CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if m.IsDisableServiceLevel(ctx, loginID, service, level) {
		return fmt.Errorf("%w: service=%s, level=%d", derror.ErrServiceDisabled, service, level)
	}
	return nil
}

// GetDisableServiceInfo retrieves the disable info for a specific service. GetDisableServiceInfo 获取账号指定服务的封禁信息。
func (m *Manager) GetDisableServiceInfo(ctx context.Context, loginID, service string) (*ServiceDisableInfo, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	if service == "" {
		return nil, derror.ErrInvalidParam
	}

	data, err := m.storage.Get(ctx, m.getDisableServiceKey(loginID, service))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrServiceNotDisabled
	}

	bytesData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var info ServiceDisableInfo
	if err = m.serializer.Decode(bytesData, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &info, nil
}

// GetDisableServiceTTL retrieves the remaining disable time for a specific service in seconds. GetDisableServiceTTL 获取账号指定服务的剩余封禁时间（秒）。
func (m *Manager) GetDisableServiceTTL(ctx context.Context, loginID, service string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, m.getDisableServiceKey(loginID, service))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	seconds := int64(ttl)
	switch {
	case ttl == adapter.TTLNotFound:
		return -2, nil
	case ttl == adapter.TTLNoExpire:
		return -1, nil
	default:
		return int64(ttl.Seconds()), nil
	}
}

func (m *Manager) CheckDisable(ctx context.Context, loginID string) error {
	if m.IsDisable(ctx, loginID) {
		return derror.ErrAccountDisabled
	}
	return nil
}

// isDisable checks if an account is disabled (internal method). isDisable 检查账号是否被封禁（内部方法）。
func (m *Manager) isDisable(ctx context.Context, loginID string) bool {
	return m.storage.Exists(ctx, m.getDisableKey(loginID))
}
