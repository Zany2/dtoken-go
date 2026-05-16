// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/utils"
	"strings"
	"time"
)

// Disable disables an account for a specified duration. Disable 封禁账号指定时长。
func (m *Manager) Disable(ctx context.Context, loginID string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

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

	// Keep token mapping for disabled-state checks and clear only metadata. 保留 token 映射以返回封禁状态，仅清理 metadata。
	if sess != nil && len(sess.TerminalInfos) > 0 {
		tokens := make([]string, len(sess.TerminalInfos))
		for i, info := range sess.TerminalInfos {
			tokens[i] = info.Token
		}

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
	if loginID == "" {
		return false
	}
	return m.isDisable(ctx, loginID)
}

// GetDisableInfo retrieves disable information for an account. GetDisableInfo 获取账号的封禁信息。
func (m *Manager) GetDisableInfo(ctx context.Context, loginID string) (*DisableInfo, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

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
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	ttl, err := m.storage.TTL(ctx, m.getDisableKey(loginID))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Explain TTL semantics 存储适配器返回 time.Duration 类型，按哨兵值和正数 TTL 语义转换为秒
	switch {
	case ttl == adapter.TTLNotFound:
		return -2, nil // 未封禁
	case ttl == adapter.TTLNoExpire:
		return -1, nil // 永久封禁
	case ttl > 0:
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
	service = strings.TrimSpace(service)
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
	service = strings.TrimSpace(service)
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
	service = strings.TrimSpace(service)
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
	service = strings.TrimSpace(service)
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
		service = strings.TrimSpace(service)
		if service == "" {
			return derror.ErrInvalidParam
		}
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
	service = strings.TrimSpace(service)
	if service == "" {
		return derror.ErrInvalidParam
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
	service = strings.TrimSpace(service)
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
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}
	service = strings.TrimSpace(service)
	if service == "" {
		return 0, derror.ErrInvalidParam
	}

	ttl, err := m.storage.TTL(ctx, m.getDisableServiceKey(loginID, service))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	switch {
	case ttl == adapter.TTLNotFound:
		return -2, nil
	case ttl == adapter.TTLNoExpire:
		return -1, nil
	case ttl > 0:
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
}

// DisableDevice disables a device type for an account. DisableDevice 封禁账号的指定设备类型。
func (m *Manager) DisableDevice(ctx context.Context, loginID, device string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}

	info := DeviceDisableInfo{
		Device:      device,
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	if err := m.saveToStorage(ctx, m.getDisableDeviceKey(loginID, device), info, duration); err != nil {
		return err
	}

	m.triggerEvent(listener.EventDisableDevice, loginID, device, "", "", map[string]any{
		"reason":   info.DisableReason,
		"duration": duration.Seconds(),
	})

	return nil
}

// DisableDeviceAndDeviceId disables a concrete device for an account. DisableDeviceAndDeviceId 封禁账号的具体设备。
func (m *Manager) DisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, duration time.Duration, reason ...string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}

	info := DeviceDisableInfo{
		Device:      device,
		DeviceId:    deviceId,
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	if err := m.saveToStorage(ctx, m.getDisableDeviceAndDeviceIdKey(loginID, device, deviceId), info, duration); err != nil {
		return err
	}

	m.triggerEvent(listener.EventDisableDevice, loginID, device, deviceId, "", map[string]any{
		"reason":   info.DisableReason,
		"duration": duration.Seconds(),
	})

	return nil
}

// UntieDevice removes device type disable state. UntieDevice 解除设备类型封禁状态。
func (m *Manager) UntieDevice(ctx context.Context, loginID, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}

	if err := m.storage.Delete(ctx, m.getDisableDeviceKey(loginID, device)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	m.triggerEvent(listener.EventUntieDevice, loginID, device, "", "", nil)

	return nil
}

// UntieDeviceAndDeviceId removes concrete device disable state. UntieDeviceAndDeviceId 解除具体设备封禁状态。
func (m *Manager) UntieDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}

	if err := m.storage.Delete(ctx, m.getDisableDeviceAndDeviceIdKey(loginID, device, deviceId)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	m.triggerEvent(listener.EventUntieDevice, loginID, device, deviceId, "", nil)

	return nil
}

// IsDisableDevice checks device type disable state. IsDisableDevice 检查设备类型封禁状态。
func (m *Manager) IsDisableDevice(ctx context.Context, loginID, device string) bool {
	device = strings.TrimSpace(device)
	if loginID == "" || device == "" {
		return false
	}
	return m.storage.Exists(ctx, m.getDisableDeviceKey(loginID, device))
}

// IsDisableDeviceAndDeviceId checks concrete device disable state. IsDisableDeviceAndDeviceId 检查具体设备封禁状态。
func (m *Manager) IsDisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) bool {
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if loginID == "" || device == "" || deviceId == "" {
		return false
	}
	return m.storage.Exists(ctx, m.getDisableDeviceAndDeviceIdKey(loginID, device, deviceId))
}

// CheckDisableDevice validates device type disable state. CheckDisableDevice 校验设备类型封禁状态。
func (m *Manager) CheckDisableDevice(ctx context.Context, loginID, device string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return derror.ErrInvalidParam
	}
	if m.IsDisableDevice(ctx, loginID, device) {
		return derror.ErrDeviceDisabled
	}
	return nil
}

// CheckDisableDeviceAndDeviceId validates concrete device disable state. CheckDisableDeviceAndDeviceId 校验具体设备封禁状态。
func (m *Manager) CheckDisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return derror.ErrInvalidParam
	}
	if m.isDisableDeviceMatch(ctx, loginID, device, deviceId) {
		return derror.ErrDeviceDisabled
	}
	return nil
}

// GetDisableDeviceInfo returns device type disable information. GetDisableDeviceInfo 获取设备类型封禁信息。
func (m *Manager) GetDisableDeviceInfo(ctx context.Context, loginID, device string) (*DeviceDisableInfo, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return nil, derror.ErrInvalidParam
	}
	return m.getDisableDeviceInfo(ctx, m.getDisableDeviceKey(loginID, device))
}

// GetDisableDeviceAndDeviceIdInfo returns concrete device disable information. GetDisableDeviceAndDeviceIdInfo 获取具体设备封禁信息。
func (m *Manager) GetDisableDeviceAndDeviceIdInfo(ctx context.Context, loginID, device, deviceId string) (*DeviceDisableInfo, error) {
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return nil, derror.ErrInvalidParam
	}
	return m.getDisableDeviceInfo(ctx, m.getDisableDeviceAndDeviceIdKey(loginID, device, deviceId))
}

// GetDisableDeviceTTL returns device type disable TTL in seconds. GetDisableDeviceTTL 获取设备类型封禁剩余秒数。
func (m *Manager) GetDisableDeviceTTL(ctx context.Context, loginID, device string) (int64, error) {
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	if device == "" {
		return 0, derror.ErrInvalidParam
	}
	return m.getDisableDeviceTTL(ctx, m.getDisableDeviceKey(loginID, device))
}

// GetDisableDeviceAndDeviceIdTTL returns concrete device disable TTL in seconds. GetDisableDeviceAndDeviceIdTTL 获取具体设备封禁剩余秒数。
func (m *Manager) GetDisableDeviceAndDeviceIdTTL(ctx context.Context, loginID, device, deviceId string) (int64, error) {
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if device == "" || deviceId == "" {
		return 0, derror.ErrInvalidParam
	}
	return m.getDisableDeviceTTL(ctx, m.getDisableDeviceAndDeviceIdKey(loginID, device, deviceId))
}

// getDisableDeviceInfo loads device disable info by key. getDisableDeviceInfo 按 key 加载设备封禁信息。
func (m *Manager) getDisableDeviceInfo(ctx context.Context, key string) (*DeviceDisableInfo, error) {
	data, err := m.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrDeviceNotDisabled
	}

	bytesData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	var info DeviceDisableInfo
	if err = m.serializer.Decode(bytesData, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	return &info, nil
}

// getDisableDeviceTTL loads device disable ttl by key. getDisableDeviceTTL 按 key 获取设备封禁剩余时间。
func (m *Manager) getDisableDeviceTTL(ctx context.Context, key string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	switch {
	case ttl == adapter.TTLNotFound:
		return -2, nil
	case ttl == adapter.TTLNoExpire:
		return -1, nil
	case ttl > 0:
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
}

func (m *Manager) CheckDisable(ctx context.Context, loginID string) error {
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}
	if m.IsDisable(ctx, loginID) {
		return derror.ErrAccountDisabled
	}
	return nil
}

// isDisable checks if an account is disabled (internal method). isDisable 检查账号是否被封禁（内部方法）。
func (m *Manager) isDisable(ctx context.Context, loginID string) bool {
	if loginID == "" {
		return false
	}
	return m.storage.Exists(ctx, m.getDisableKey(loginID))
}

// isDisableDeviceMatch checks device disable state. isDisableDeviceMatch 检查设备封禁状态。
func (m *Manager) isDisableDeviceMatch(ctx context.Context, loginID, device, deviceId string) bool {
	device = strings.TrimSpace(device)
	deviceId = strings.TrimSpace(deviceId)
	if loginID == "" || device == "" {
		return false
	}
	if m.storage.Exists(ctx, m.getDisableDeviceKey(loginID, device)) {
		return true
	}
	return deviceId != "" && m.storage.Exists(ctx, m.getDisableDeviceAndDeviceIdKey(loginID, device, deviceId))
}
