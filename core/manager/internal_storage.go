// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"fmt"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"strings"
	"time"
)

// getExpiration calculates token expiration duration from configuration. getExpiration 从配置中计算 Token 过期时长。
func (m *Manager) getExpiration() time.Duration {
	// Use configured timeout when limited 配置有限超时时使用配置值。
	if m.config.Timeout > 0 {
		return time.Duration(m.config.Timeout) * time.Second
	}
	// Return no expiration 返回不过期。
	return 0
}

// timeoutToSeconds converts duration to storage seconds timeoutToSeconds 将时长转换为存储层秒数
func (m *Manager) timeoutToSeconds(timeout time.Duration) int64 {
	// Map non-positive duration to no limit 非正时长映射为无限制。
	if timeout <= 0 {
		return config.NoLimit
	}

	// Convert duration to whole seconds 转换为整秒。
	seconds := int64(timeout / time.Second)
	// Round up partial second 不足一秒向上取整。
	if timeout%time.Second != 0 {
		seconds++
	}
	// Ensure positive second 保证至少一秒。
	if seconds <= 0 {
		return 1
	}
	return seconds
}

// resolveTokenExpiration resolves token expiration from token info resolveTokenExpiration 根据 token info 解析实际过期时长
func (m *Manager) resolveTokenExpiration(tokenInfo *TokenInfo) time.Duration {
	// Prefer token-specific timeout 优先使用 Token 自身超时。
	if tokenInfo != nil {
		switch {
		case tokenInfo.Timeout == config.NoLimit:
			return 0
		case tokenInfo.Timeout > 0:
			return time.Duration(tokenInfo.Timeout) * time.Second
		}
	}
	// Fallback to global expiration 回退到全局过期时间。
	return m.getExpiration()
}

// saveSessionWithMinTTL saves session while keeping the longer existing TTL saveSessionWithMinTTL 保存 session，并保留更长的现有 TTL
func (m *Manager) saveSessionWithMinTTL(
	ctx context.Context,
	key string,
	value any,
	expiration time.Duration,
) error {
	// Start with requested expiration 先使用请求的过期时间。
	finalExpiration := expiration
	// Preserve longer existing TTL 保留更长的已有 TTL。
	if expiration > 0 {
		currentTTL, err := m.storage.TTL(ctx, key)
		if err == nil {
			switch {
			case currentTTL == adapter.TTLNoExpire:
				finalExpiration = 0
			case currentTTL > expiration:
				finalExpiration = currentTTL
			}
		}
	}

	// Persist session 保存会话。
	return m.saveToStorage(ctx, key, value, finalExpiration)
}

// getDeviceAndDeviceId extracts device type and device ID from parameters. getDeviceAndDeviceId 获取设备类型和设备 ID。 规则：device 和 deviceId 是两个独立的过滤维度，互不影响
func (m *Manager) getDeviceAndDeviceId(deviceAndDeviceId ...string) (string, string) {
	// Initialize empty device fields 初始化空设备字段。
	device := ""
	deviceId := ""

	// Read device type 读取设备类型。
	if len(deviceAndDeviceId) > 0 {
		device = strings.TrimSpace(deviceAndDeviceId[0])
	}

	// Read device ID 读取设备 ID。
	if len(deviceAndDeviceId) > 1 {
		deviceId = strings.TrimSpace(deviceAndDeviceId[1])
	}

	// Return normalized device fields 返回规范化设备字段。
	return device, deviceId
}

// saveToStorage serializes and saves data to storage backend. saveToStorage 将指定类型的数据序列化并存储到存储后端。
func (m *Manager) saveToStorage(
	ctx context.Context,
	key string,
	value any,
	expiration ...time.Duration,
) error {

	// Serialize to bytes 序列化为字节
	bytesData, err := m.serializer.Encode(value)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Build expiration duration 构建过期时长
	duration := m.getExpiration()
	// Use explicit expiration when provided 存在显式过期时间时优先使用。
	if len(expiration) > 0 {
		duration = expiration[0]
	} else {
		// Keep existing TTL when possible 尽量保留已有 TTL。
		currentTTL, ttlErr := m.storage.TTL(ctx, key)
		if ttlErr == nil {
			switch {
			case currentTTL == adapter.TTLNoExpire:
				duration = 0
			case currentTTL > 0:
				duration = currentTTL
			}
		}
	}

	// Persist to storage 存储到后端
	if err = m.storage.Set(ctx, key, bytesData, duration); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Return save success 返回保存成功。
	return nil
}

// searchKeys searches storage keys by pattern with pagination. searchKeys 根据模式搜索存储键并分页。
func (m *Manager) searchKeys(ctx context.Context, pattern string, start, size int) ([]string, error) {
	// Require scanner storage capability 要求存储支持扫描能力。
	scanner, ok := m.storage.(adapter.ScannerStorage)
	if !ok {
		return nil, fmt.Errorf("%w: storage scanner capability is required", derror.ErrStorageUnavailable)
	}

	// Load matched keys 加载匹配键。
	keys, err := scanner.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Normalize pagination start 规范化分页起点。
	total := len(keys)
	if start < 0 {
		start = 0
	}
	// Return empty when start exceeds total 起点超过总数时返回空列表。
	if start >= total {
		return []string{}, nil
	}

	// Return all when size == -1 size == -1 表示返回全部
	// Calculate pagination end 计算分页终点。
	end := total
	if size >= 0 {
		end = start + size
		if end > total {
			end = total
		}
	}

	// Return key page 返回键分页。
	return keys[start:end], nil
}

// searchValues searches keys and strips storage prefix. searchValues 搜索存储键并裁剪为业务值。
func (m *Manager) searchValues(ctx context.Context, pattern, prefix string, start, size int) ([]string, error) {
	// Search storage keys 搜索存储键。
	keys, err := m.searchKeys(ctx, pattern, start, size)
	if err != nil {
		return nil, err
	}
	// Strip prefix from keys 从键中裁剪前缀。
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = strings.TrimPrefix(key, prefix)
	}
	// Return business values 返回业务值。
	return values, nil
}
