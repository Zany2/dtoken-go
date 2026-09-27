// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/utils"
)

// Disable disables an account for a specified duration, where zero means permanent. Disable 按指定时长封禁账号，时长为零表示永久封禁。
func (m *Manager) Disable(ctx context.Context, loginID string, duration time.Duration, reason ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Validate disable duration 校验封禁时长。
	if duration < 0 {
		return derror.ErrInvalidParam
	}

	// Lock account writes 锁定账号写操作。
	unlock := m.lockLoginWrite(loginID)

	// Release lock on function exit 函数退出时释放锁。
	defer func() { unlock() }()

	// Load session before disable 封禁前先尝试加载 Session，存储出错时在保存封禁信息前返回
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		// Ignore missing session and return other storage errors 如果只是 session 不存在，不算错误；其他存储错误则返回
		if !errors.Is(err, derror.ErrSessionNotFound) {
			return err
		}

		// Continue disable when sess is nil 否则 sess == nil，继续执行封禁操作（幂等。
	}

	// Build and save disable info 构建并保存封禁信。
	disableInfo := DisableInfo{
		DisableTime: time.Now().Unix(),
	}
	if len(reason) > 0 && reason[0] != "" {
		disableInfo.DisableReason = reason[0]
	}

	// Save account disable marker 保存账号封禁标记。
	if err = m.saveDisableMarker(ctx, m.getDisableKey(loginID), m.getLegacyDisableKey(loginID), disableMarker{DisableInfo: disableInfo, LoginID: loginID, Kind: "account"}, duration); err != nil {
		return err
	}

	// Retire the captured terminals before losing their index; cleanup must finish before untie. 丢失终端索引前使其失效，确保解封前清理完成。
	if sess != nil {
		if err = m.retireDisabledTerminals(ctx, sess); err != nil {
			return err
		}
	}

	// Delete the session only after its credentials have been retired. 凭证失效处理完成后再删除 Session。
	if err = m.storage.Delete(ctx, m.getSessionKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Release lock before events 触发事件前释放锁。
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
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Serialize account disable and untie operations. 串行化账号封禁与解封操作。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Resolve ownership before deleting current or legacy account markers. 删除新旧账号封禁标记前先核对归属。
	records, err := m.loadAccountDisableRecords(ctx, loginID, true)
	if err != nil {
		return err
	}
	changed := false
	for _, record := range records {
		deleted, deleteErr := m.deleteWithLegacyKey(ctx, record.key, record.key)
		if deleteErr != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, deleteErr)
		}
		changed = changed || deleted
	}
	if !changed {
		return nil
	}

	// Release the account lock before dispatching lifecycle events. 触发生命周期事件前释放账号锁。
	unlock()
	unlock = func() {}

	// Trigger untie event 触发解禁事件
	m.triggerEvent(listener.EventUntie, loginID, "", "", "", nil)

	return nil
}

// IsDisable checks if an account is disabled. IsDisable 检查账号是否被封禁。
func (m *Manager) IsDisable(ctx context.Context, loginID string) bool {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return false
	}

	// Check account disable marker 检查账号封禁标记。
	return m.isDisable(ctx, loginID)
}

// GetDisableInfo retrieves disable information for an account. GetDisableInfo 获取账号的封禁信息。
func (m *Manager) GetDisableInfo(ctx context.Context, loginID string) (*DisableInfo, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// Load only identity-matched account records. 仅加载身份匹配的账号封禁记录。
	records, err := m.loadAccountDisableRecords(ctx, loginID, false)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, derror.ErrAccountNotDisabled
	}

	// Return the unchanged public metadata shape. 返回保持原有结构的公开元数据。
	disableInfo := records[0].marker.DisableInfo
	return &disableInfo, nil
}

// GetDisableTTL returns remaining account disable seconds: -2 for enabled, -1 for permanent, and positive values for remaining time. GetDisableTTL 返回账号剩余封禁秒数：-2 表示未封禁，-1 表示永久封禁，正数表示剩余时间。
func (m *Manager) GetDisableTTL(ctx context.Context, loginID string) (int64, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	// Resolve ownership before reading the remaining lifetime. 读取剩余有效期前先核对归属。
	records, err := m.loadAccountDisableRecords(ctx, loginID, false)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return -2, nil
	}
	return m.getDisableTTL(ctx, records[0].key)
}

// DisableService disables a specific service for an account. DisableService 封禁账号的指定服务。
func (m *Manager) DisableService(ctx context.Context, loginID, service string, duration time.Duration, reason ...string) error {
	return m.DisableServiceLevel(ctx, loginID, service, 0, duration, reason...)
}

// DisableServiceLevel disables a specific service for an account with a level. DisableServiceLevel 封禁账号的指定服务并设置封禁等级。
func (m *Manager) DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration, reason ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize service name 规范化服务名称。
	service = strings.TrimSpace(service)

	// Validate service name 校验服务名称。
	if service == "" {
		return derror.ErrInvalidParam
	}

	// Validate service level 校验服务封禁等级。
	if level < 0 {
		return derror.ErrInvalidParam
	}

	// Validate disable duration 校验封禁时长。
	if duration < 0 {
		return derror.ErrInvalidParam
	}

	// Serialize service disable writes with login and untie operations. 与登录及解封操作串行化服务封禁写入。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Build service disable info 构建服务封禁信息。
	info := ServiceDisableInfo{
		Service:     service,
		Level:       level,
		DisableTime: time.Now().Unix(),
	}

	// Fill disable reason 填充封禁原因。
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	// Persist the owner alongside the unchanged public service fields. 保存账号归属及原有公开服务字段。
	marker := disableMarker{
		DisableInfo: DisableInfo{DisableTime: info.DisableTime, DisableReason: info.DisableReason},
		LoginID:     loginID, Kind: "service", Service: service, Level: level,
	}
	if err := m.saveDisableMarker(ctx, m.getDisableServiceKey(loginID, service), m.getLegacyDisableServiceKey(loginID, service), marker, duration); err != nil {
		return err
	}

	// Release the account lock before dispatching lifecycle events. 触发生命周期事件前释放账号锁。
	unlock()
	unlock = func() {}

	// Trigger service disable event 触发服务封禁事件。
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
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize service name 规范化服务名称。
	service = strings.TrimSpace(service)

	// Validate service name 校验服务名称。
	if service == "" {
		return derror.ErrInvalidParam
	}

	// Serialize service untie writes with login and disable operations. 与登录及封禁操作串行化服务解封写入。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Resolve only markers whose embedded service matches the requested identity. 仅解析内嵌服务与请求身份一致的封禁标记。
	records, err := m.loadServiceDisableRecords(ctx, loginID, service, true)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	// Delete all matching current and legacy markers without touching colliding legacy records. 删除所有匹配的新旧标记，但不触碰发生碰撞的旧记录。
	keys := make([]string, len(records))
	for i := range records {
		keys[i] = records[i].key
	}
	if err = m.storage.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Release the account lock before dispatching lifecycle events. 触发生命周期事件前释放账号锁。
	unlock()
	unlock = func() {}

	// Trigger service untie event 触发服务解封事件。
	m.triggerEvent(listener.EventUntieService, loginID, "", "", "", map[string]any{
		listener.ExtraKeyService: service,
	})

	return nil
}

// IsDisableService checks if a specific service is disabled for an account. IsDisableService 检查账号的指定服务是否被封禁。
func (m *Manager) IsDisableService(ctx context.Context, loginID, service string) bool {
	// Normalize service name 规范化服务名称。
	service = strings.TrimSpace(service)

	// Validate required parameters 校验必要参数。
	if loginID == "" || service == "" {
		return false
	}

	// An escaped key can also contain another identity's legacy marker. 转义键也可能存有其他身份的旧标记。
	records, err := m.loadServiceDisableRecords(ctx, loginID, service, false)
	return err == nil && len(records) > 0
}

// IsDisableServiceLevel checks if a specific service is disabled at or above the given level. IsDisableServiceLevel 检查账号的指定服务是否达到指定封禁等级。
func (m *Manager) IsDisableServiceLevel(ctx context.Context, loginID, service string, level int) bool {
	// Reject invalid service level 拒绝非法服务封禁等级。
	if level < 0 {
		return false
	}

	// Load service disable info 加载服务封禁信息。
	info, err := m.GetDisableServiceInfo(ctx, loginID, service)
	if err != nil {
		return false
	}

	// Compare disable level 比较封禁等级。
	return info.Level >= level
}

// CheckDisableService checks if any of the specified services are disabled, returns error if disabled. CheckDisableService 校验账号的指定服务是否被封禁，被封禁则返回 error。
func (m *Manager) CheckDisableService(ctx context.Context, loginID string, services ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Check each service 逐个校验服务。
	for _, service := range services {
		// Normalize service name 规范化服务名称。
		service = strings.TrimSpace(service)

		// Validate service name 校验服务名称。
		if service == "" {
			return derror.ErrInvalidParam
		}

		// Load service disable info to propagate storage errors 加载服务封禁信息以传递存储错误。
		if _, err := m.GetDisableServiceInfo(ctx, loginID, service); err == nil {
			return fmt.Errorf("%w: service=%s", derror.ErrServiceDisabled, service)
		} else if !errors.Is(err, derror.ErrServiceNotDisabled) {
			return err
		}
	}

	return nil
}

// CheckDisableServiceLevel checks if a service is disabled at or above the given level, returns error if so. CheckDisableServiceLevel 校验账号的指定服务是否达到指定封禁等级，达到则返回 error。
func (m *Manager) CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize service name 规范化服务名称。
	service = strings.TrimSpace(service)

	// Validate service name 校验服务名称。
	if service == "" {
		return derror.ErrInvalidParam
	}

	// Validate service level 校验服务封禁等级。
	if level < 0 {
		return derror.ErrInvalidParam
	}

	// Load service disable info to propagate storage errors 加载服务封禁信息以传播存储错误，避免存储异常时静默放。
	info, err := m.GetDisableServiceInfo(ctx, loginID, service)
	if err != nil {
		if errors.Is(err, derror.ErrServiceNotDisabled) {
			return nil
		}
		return err
	}

	// Reject disabled service level 拒绝达到等级的封禁服务。
	if info.Level >= level {
		return fmt.Errorf("%w: service=%s, level=%d", derror.ErrServiceDisabled, service, level)
	}
	return nil
}

// GetDisableServiceInfo retrieves the disable info for a specific service. GetDisableServiceInfo 获取账号指定服务的封禁信息。
func (m *Manager) GetDisableServiceInfo(ctx context.Context, loginID, service string) (*ServiceDisableInfo, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// Normalize service name 规范化服务名称。
	service = strings.TrimSpace(service)

	// Validate service name 校验服务名称。
	if service == "" {
		return nil, derror.ErrInvalidParam
	}

	// Load identity-matched service disable data. 加载身份匹配的服务封禁数据。
	records, err := m.loadServiceDisableRecords(ctx, loginID, service, false)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, derror.ErrServiceNotDisabled
	}
	info := records[0].info
	return &info, nil
}

// serviceDisableRecord binds decoded service state to its storage key. serviceDisableRecord 将解码后的服务封禁状态绑定到存储键。
type serviceDisableRecord struct {
	key  string
	info ServiceDisableInfo
}

// loadDisableMarker decodes one marker and reports whether it exists. loadDisableMarker 解码单个封禁标记并报告其是否存在。
func (m *Manager) loadDisableMarker(ctx context.Context, key string, info any) (bool, error) {
	data, err := m.storage.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return false, nil
	}

	bytesData, err := utils.ToBytes(data)
	if err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	if err = m.serializer.Decode(bytesData, info); err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	return true, nil
}

// uniqueStorageKeys removes empty and duplicate storage keys while preserving order. uniqueStorageKeys 按原顺序移除空存储键和重复存储键。
func uniqueStorageKeys(keys ...string) []string {
	unique := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}
	return unique
}

// loadServiceDisableRecords matches stored service fields, returning all matches only for untie operations. loadServiceDisableRecords 核对已存服务字段，仅解封操作需要返回所有匹配记录。
func (m *Manager) loadServiceDisableRecords(ctx context.Context, loginID, service string, all bool) ([]serviceDisableRecord, error) {
	matched, err := m.loadDisableRecords(ctx, disableMarker{LoginID: loginID, Kind: "service", Service: service}, m.getDisableServiceKey(loginID, service), m.getLegacyDisableServiceKey(loginID, service), all)
	if err != nil {
		return nil, err
	}
	records := make([]serviceDisableRecord, 0, len(matched))
	for _, record := range matched {
		marker := record.marker
		records = append(records, serviceDisableRecord{key: record.key, info: ServiceDisableInfo{
			Service: marker.Service, Level: marker.Level, DisableTime: marker.DisableTime, DisableReason: marker.DisableReason,
		}})
	}
	return records, nil
}

// GetDisableServiceTTL retrieves the remaining disable time for a specific service in seconds. GetDisableServiceTTL 获取账号指定服务的剩余封禁时间（秒）。
func (m *Manager) GetDisableServiceTTL(ctx context.Context, loginID, service string) (int64, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	// Normalize service name 规范化服务名称。
	service = strings.TrimSpace(service)

	// Validate service name 校验服务名称。
	if service == "" {
		return 0, derror.ErrInvalidParam
	}

	// Resolve a metadata-matched record before reading its TTL. 读取 TTL 前先确定字段匹配的记录。
	records, err := m.loadServiceDisableRecords(ctx, loginID, service, false)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return -2, nil
	}
	return m.getDisableTTL(ctx, records[0].key)
}

// getDisableTTL loads and normalizes a disable marker TTL. getDisableTTL 加载并归一化封禁标记的剩余时间。
func (m *Manager) getDisableTTL(ctx context.Context, key string) (int64, error) {
	ttl, err := m.storage.TTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return normalizeTTLSeconds(ttl), nil
}

// DisableDevice disables a device type for an account. DisableDevice 封禁账号的指定设备类型。
func (m *Manager) DisableDevice(ctx context.Context, loginID, device string, duration time.Duration, reason ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate device type 校验设备类型。
	if device == "" {
		return derror.ErrInvalidParam
	}

	// Validate disable duration 校验封禁时长。
	if duration < 0 {
		return derror.ErrInvalidParam
	}

	// Serialize device disable writes with login and untie operations. 与登录及解封操作串行化设备封禁写入。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Build device disable info 构建设备封禁信息。
	info := DeviceDisableInfo{
		Device:      device,
		DisableTime: time.Now().Unix(),
	}

	// Fill disable reason 填充封禁原因。
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	// Persist the complete device identity with its account owner. 保存完整设备身份及所属账号。
	marker := disableMarker{
		DisableInfo: DisableInfo{DisableTime: info.DisableTime, DisableReason: info.DisableReason},
		LoginID:     loginID, Kind: "device", Device: info.Device, DeviceID: info.DeviceID,
	}
	key := m.getDisableDeviceKey(loginID, device)
	if err := m.saveDisableMarker(ctx, key, m.getLegacyDisableDeviceKey(loginID, device), marker, duration); err != nil {
		return err
	}

	// Release the account lock before dispatching lifecycle events. 触发生命周期事件前释放账号锁。
	unlock()
	unlock = func() {}

	// Trigger device disable event 触发设备封禁事件。
	m.triggerEvent(listener.EventDisableDevice, loginID, device, "", "", map[string]any{
		"reason":   info.DisableReason,
		"duration": duration.Seconds(),
	})

	return nil
}

// DisableDeviceAndDeviceID disables a concrete device for an account. DisableDeviceAndDeviceID 封禁账号的具体设备。
func (m *Manager) DisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string, duration time.Duration, reason ...string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return derror.ErrInvalidParam
	}

	// Validate disable duration 校验封禁时长。
	if duration < 0 {
		return derror.ErrInvalidParam
	}

	// Serialize concrete device disable writes with login and untie operations. 与登录及解封操作串行化具体设备封禁写入。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Build concrete device disable info 构建具体设备封禁信息。
	info := DeviceDisableInfo{
		Device:      device,
		DeviceID:    deviceID,
		DisableTime: time.Now().Unix(),
	}

	// Fill disable reason 填充封禁原因。
	if len(reason) > 0 && reason[0] != "" {
		info.DisableReason = reason[0]
	}

	// Persist the complete device identity with its account owner. 保存完整设备身份及所属账号。
	marker := disableMarker{
		DisableInfo: DisableInfo{DisableTime: info.DisableTime, DisableReason: info.DisableReason},
		LoginID:     loginID, Kind: "device", Device: info.Device, DeviceID: info.DeviceID,
	}
	key := m.getDisableDeviceAndDeviceIDKey(loginID, device, deviceID)
	if err := m.saveDisableMarker(ctx, key, m.getLegacyDisableDeviceAndDeviceIDKey(loginID, device, deviceID), marker, duration); err != nil {
		return err
	}

	// Release the account lock before dispatching lifecycle events. 触发生命周期事件前释放账号锁。
	unlock()
	unlock = func() {}

	// Trigger device disable event 触发设备封禁事件。
	m.triggerEvent(listener.EventDisableDevice, loginID, device, deviceID, "", map[string]any{
		"reason":   info.DisableReason,
		"duration": duration.Seconds(),
	})

	return nil
}

// UntieDevice removes device type disable state. UntieDevice 解除设备类型封禁状态。
func (m *Manager) UntieDevice(ctx context.Context, loginID, device string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate device type 校验设备类型。
	if device == "" {
		return derror.ErrInvalidParam
	}

	// Serialize device untie writes with login and disable operations. 与登录及封禁操作串行化设备解封写入。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Resolve only markers whose embedded device identity matches the request. 仅解析内嵌设备身份与请求一致的封禁标记。
	records, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		m.getDisableDeviceKey(loginID, device),
		m.getLegacyDisableDeviceKey(loginID, device),
		device,
		"",
		true,
	)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	// Delete all matching current and legacy markers without touching colliding legacy records. 删除所有匹配的新旧标记，但不触碰发生碰撞的旧记录。
	keys := make([]string, len(records))
	for i := range records {
		keys[i] = records[i].key
	}
	if err = m.storage.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Release the account lock before dispatching lifecycle events. 触发生命周期事件前释放账号锁。
	unlock()
	unlock = func() {}

	// Trigger device untie event 触发设备解封事件。
	m.triggerEvent(listener.EventUntieDevice, loginID, device, "", "", nil)

	return nil
}

// UntieDeviceAndDeviceID removes concrete device disable state. UntieDeviceAndDeviceID 解除具体设备封禁状态。
func (m *Manager) UntieDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return derror.ErrInvalidParam
	}

	// Serialize concrete device untie writes with login and disable operations. 与登录及封禁操作串行化具体设备解封写入。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Resolve only markers whose embedded device identity matches the request. 仅解析内嵌设备身份与请求一致的封禁标记。
	records, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		m.getDisableDeviceAndDeviceIDKey(loginID, device, deviceID),
		m.getLegacyDisableDeviceAndDeviceIDKey(loginID, device, deviceID),
		device,
		deviceID,
		true,
	)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	// Delete all matching current and legacy markers without touching colliding legacy records. 删除所有匹配的新旧标记，但不触碰发生碰撞的旧记录。
	keys := make([]string, len(records))
	for i := range records {
		keys[i] = records[i].key
	}
	if err = m.storage.Delete(ctx, keys...); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Release the account lock before dispatching lifecycle events. 触发生命周期事件前释放账号锁。
	unlock()
	unlock = func() {}

	// Trigger device untie event 触发设备解封事件。
	m.triggerEvent(listener.EventUntieDevice, loginID, device, deviceID, "", nil)

	return nil
}

// IsDisableDevice checks device type disable state. IsDisableDevice 检查设备类型封禁状态。
func (m *Manager) IsDisableDevice(ctx context.Context, loginID, device string) bool {
	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate required parameters 校验必要参数。
	if loginID == "" || device == "" {
		return false
	}

	currentKey := m.getDisableDeviceKey(loginID, device)
	legacyKey := m.getLegacyDisableDeviceKey(loginID, device)

	// An escaped key can also contain another identity's legacy marker. 转义键也可能存有其他身份的旧标记。
	records, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		currentKey,
		legacyKey,
		device,
		"",
		false,
	)
	return err == nil && len(records) > 0
}

// IsDisableDeviceAndDeviceID checks concrete device disable state. IsDisableDeviceAndDeviceID 检查具体设备封禁状态。
func (m *Manager) IsDisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) bool {
	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate required parameters 校验必要参数。
	if loginID == "" || device == "" || deviceID == "" {
		return false
	}

	// Match both device type and concrete device disable rules 同时匹配设备类型与具体设备封禁规则。
	return m.isDisableDeviceMatch(ctx, loginID, device, deviceID)
}

// CheckDisableDevice validates device type disable state. CheckDisableDevice 校验设备类型封禁状态。
func (m *Manager) CheckDisableDevice(ctx context.Context, loginID, device string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate device type 校验设备类型。
	if device == "" {
		return derror.ErrInvalidParam
	}

	// Load device disable info to propagate storage errors 加载设备封禁信息以传递存储错误。
	if _, err := m.GetDisableDeviceInfo(ctx, loginID, device); err == nil {
		return derror.ErrDeviceDisabled
	} else if !errors.Is(err, derror.ErrDeviceNotDisabled) {
		return err
	}
	return nil
}

// CheckDisableDeviceAndDeviceID validates concrete device disable state. CheckDisableDeviceAndDeviceID 校验具体设备封禁状态。
func (m *Manager) CheckDisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return derror.ErrInvalidParam
	}

	// Check device type disable first 先检查设备类型封禁。
	if _, err := m.GetDisableDeviceInfo(ctx, loginID, device); err == nil {
		return derror.ErrDeviceDisabled
	} else if !errors.Is(err, derror.ErrDeviceNotDisabled) {
		return err
	}

	// Check concrete device disable 再检查具体设备封禁。
	if _, err := m.GetDisableDeviceAndDeviceIDInfo(ctx, loginID, device, deviceID); err == nil {
		return derror.ErrDeviceDisabled
	} else if !errors.Is(err, derror.ErrDeviceNotDisabled) {
		return err
	}
	return nil
}

// deviceDisableRecord binds decoded device state to its storage key. deviceDisableRecord 将解码后的设备封禁状态绑定到存储键。
type deviceDisableRecord struct {
	key  string
	info DeviceDisableInfo
}

// loadDeviceDisableRecords matches stored device fields, returning all matches only for untie operations. loadDeviceDisableRecords 核对已存设备字段，仅解封操作需要返回所有匹配记录。
func (m *Manager) loadDeviceDisableRecords(ctx context.Context, loginID, key, legacyKey, device, deviceID string, all bool) ([]deviceDisableRecord, error) {
	matched, err := m.loadDisableRecords(ctx, disableMarker{LoginID: loginID, Kind: "device", Device: device, DeviceID: deviceID}, key, legacyKey, all)
	if err != nil {
		return nil, err
	}
	records := make([]deviceDisableRecord, 0, len(matched))
	for _, record := range matched {
		marker := record.marker
		records = append(records, deviceDisableRecord{key: record.key, info: DeviceDisableInfo{
			Device: marker.Device, DeviceID: marker.DeviceID, DisableTime: marker.DisableTime, DisableReason: marker.DisableReason,
		}})
	}
	return records, nil
}

// GetDisableDeviceInfo returns device type disable information. GetDisableDeviceInfo 获取设备类型封禁信息。
func (m *Manager) GetDisableDeviceInfo(ctx context.Context, loginID, device string) (*DeviceDisableInfo, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate device type 校验设备类型。
	if device == "" {
		return nil, derror.ErrInvalidParam
	}

	// Load identity-matched device disable info 加载身份匹配的设备封禁信息。
	records, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		m.getDisableDeviceKey(loginID, device),
		m.getLegacyDisableDeviceKey(loginID, device),
		device,
		"",
		false,
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, derror.ErrDeviceNotDisabled
	}
	info := records[0].info
	return &info, nil
}

// GetDisableDeviceAndDeviceIDInfo returns concrete device disable information. GetDisableDeviceAndDeviceIDInfo 获取具体设备封禁信息。
func (m *Manager) GetDisableDeviceAndDeviceIDInfo(ctx context.Context, loginID, device, deviceID string) (*DeviceDisableInfo, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}

	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return nil, derror.ErrInvalidParam
	}

	// Load identity-matched concrete device disable info 加载身份匹配的具体设备封禁信息。
	records, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		m.getDisableDeviceAndDeviceIDKey(loginID, device, deviceID),
		m.getLegacyDisableDeviceAndDeviceIDKey(loginID, device, deviceID),
		device,
		deviceID,
		false,
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, derror.ErrDeviceNotDisabled
	}
	info := records[0].info
	return &info, nil
}

// GetDisableDeviceTTL returns device type disable TTL in seconds. GetDisableDeviceTTL 获取设备类型封禁剩余秒数。
func (m *Manager) GetDisableDeviceTTL(ctx context.Context, loginID, device string) (int64, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	// Normalize device type 规范化设备类型。
	device = strings.TrimSpace(device)

	// Validate device type 校验设备类型。
	if device == "" {
		return 0, derror.ErrInvalidParam
	}

	currentKey := m.getDisableDeviceKey(loginID, device)
	legacyKey := m.getLegacyDisableDeviceKey(loginID, device)

	// Resolve a metadata-matched record before reading its TTL. 读取 TTL 前先确定字段匹配的记录。
	records, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		currentKey,
		legacyKey,
		device,
		"",
		false,
	)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return -2, nil
	}
	return m.getDisableTTL(ctx, records[0].key)
}

// GetDisableDeviceAndDeviceIDTTL returns concrete device disable TTL in seconds. GetDisableDeviceAndDeviceIDTTL 获取具体设备封禁剩余秒数。
func (m *Manager) GetDisableDeviceAndDeviceIDTTL(ctx context.Context, loginID, device, deviceID string) (int64, error) {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return 0, derror.ErrIDIsEmpty
	}

	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate device fields 校验设备字段。
	if device == "" || deviceID == "" {
		return 0, derror.ErrInvalidParam
	}

	currentKey := m.getDisableDeviceAndDeviceIDKey(loginID, device, deviceID)
	legacyKey := m.getLegacyDisableDeviceAndDeviceIDKey(loginID, device, deviceID)

	// Resolve a metadata-matched record before reading its TTL. 读取 TTL 前先确定字段匹配的记录。
	records, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		currentKey,
		legacyKey,
		device,
		deviceID,
		false,
	)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return -2, nil
	}
	return m.getDisableTTL(ctx, records[0].key)
}

// CheckDisable validates account disable state. CheckDisable 校验账号封禁状态。
func (m *Manager) CheckDisable(ctx context.Context, loginID string) error {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return derror.ErrIDIsEmpty
	}

	// Load account disable info to propagate storage errors 加载账号封禁信息以传递存储错误。
	if _, err := m.GetDisableInfo(ctx, loginID); err == nil {
		return derror.ErrAccountDisabled
	} else if !errors.Is(err, derror.ErrAccountNotDisabled) {
		return err
	}
	return nil
}

// checkLoginDisableState checks account and device disable states. checkLoginDisableState 检查账号和设备封禁状态。
func (m *Manager) checkLoginDisableState(ctx context.Context, loginID, device, deviceID string) error {
	// Use error-aware lookups so storage failures cannot become an implicit allow. 使用可返回错误的查询，避免存储故障被误判为允许访问。
	if err := m.CheckDisable(ctx, loginID); err != nil {
		return err
	}

	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)
	if device == "" {
		return nil
	}
	if deviceID != "" {
		return m.CheckDisableDeviceAndDeviceID(ctx, loginID, device, deviceID)
	}
	return m.CheckDisableDevice(ctx, loginID, device)
}

// isDisable checks if an account is disabled. isDisable 检查账号是否被封禁。
func (m *Manager) isDisable(ctx context.Context, loginID string) bool {
	// Validate login ID 校验登录 ID。
	if loginID == "" {
		return false
	}

	// Do not treat a colliding service or device marker as an account ban. 不将冲突的服务或设备记录视为账号封禁。
	_, err := m.GetDisableInfo(ctx, loginID)
	return err == nil
}

// isDisableDeviceMatch checks device disable state. isDisableDeviceMatch 检查设备封禁状态。
func (m *Manager) isDisableDeviceMatch(ctx context.Context, loginID, device, deviceID string) bool {
	// Normalize device fields 规范化设备字段。
	device = strings.TrimSpace(device)
	deviceID = strings.TrimSpace(deviceID)

	// Validate required parameters 校验必要参数。
	if loginID == "" || device == "" {
		return false
	}

	// Match device type disable 匹配设备类型封禁。
	if m.IsDisableDevice(ctx, loginID, device) {
		return true
	}

	// Match concrete device disable 匹配具体设备封禁。
	if deviceID == "" {
		return false
	}
	currentKey := m.getDisableDeviceAndDeviceIDKey(loginID, device, deviceID)
	legacyKey := m.getLegacyDisableDeviceAndDeviceIDKey(loginID, device, deviceID)

	// An escaped key can also contain another identity's legacy marker. 转义键也可能存有其他身份的旧标记。
	concreteRecords, err := m.loadDeviceDisableRecords(
		ctx,
		loginID,
		currentKey,
		legacyKey,
		device,
		deviceID,
		false,
	)
	return err == nil && len(concreteRecords) > 0
}
