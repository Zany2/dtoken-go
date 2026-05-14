// Author records daixk as original author at 2026/1/22 17:33:00. Author 记录 daixk 为原始作者，创建时间为 2026/1/22 17:33:00。
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
	if m.config.Timeout > 0 {
		return time.Duration(m.config.Timeout) * time.Second
	}
	return 0
}

// timeoutToSeconds converts duration to storage seconds timeoutToSeconds 将时长转换为存储层秒数
func (m *Manager) timeoutToSeconds(timeout time.Duration) int64 {
	if timeout <= 0 {
		return config.NoLimit
	}

	seconds := int64(timeout / time.Second)
	if timeout%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return seconds
}

// resolveTokenExpiration resolves token expiration from token info resolveTokenExpiration 根据 token info 解析实际过期时长
func (m *Manager) resolveTokenExpiration(tokenInfo *TokenInfo) time.Duration {
	if tokenInfo != nil {
		switch {
		case tokenInfo.Timeout == config.NoLimit:
			return 0
		case tokenInfo.Timeout > 0:
			return time.Duration(tokenInfo.Timeout) * time.Second
		}
	}
	return m.getExpiration()
}

// saveSessionWithMinTTL saves session while keeping the longer existing TTL saveSessionWithMinTTL 保存 session，并保留更长的现有 TTL
func (m *Manager) saveSessionWithMinTTL(
	ctx context.Context,
	key string,
	value any,
	expiration time.Duration,
) error {
	finalExpiration := expiration
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

	return m.saveToStorage(ctx, key, value, finalExpiration)
}

// getDeviceAndDeviceId extracts device type and device ID from parameters. getDeviceAndDeviceId 获取设备类型和设备 ID。 规则：device 和 deviceId 是两个独立的过滤维度，互不影响
func (m *Manager) getDeviceAndDeviceId(deviceAndDeviceId ...string) (string, string) {
	device := ""
	deviceId := ""

	if len(deviceAndDeviceId) > 0 {
		device = strings.TrimSpace(deviceAndDeviceId[0])
	}

	if len(deviceAndDeviceId) > 1 {
		deviceId = strings.TrimSpace(deviceAndDeviceId[1])
	}

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
	if len(expiration) > 0 {
		duration = expiration[0]
	} else {
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

	return nil
}

// searchKeys searches storage keys by pattern with pagination (internal method). searchKeys 根据模式搜索存储键并分页（内部方法）。
func (m *Manager) searchKeys(ctx context.Context, pattern string, start, size int) ([]string, error) {
	scanner, ok := m.storage.(adapter.ScannerStorage)
	if !ok {
		return nil, fmt.Errorf("%w: storage scanner capability is required", derror.ErrStorageUnavailable)
	}

	keys, err := scanner.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	total := len(keys)
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []string{}, nil
	}

	// Return all when size == -1 size == -1 表示返回全部
	end := total
	if size >= 0 {
		end = start + size
		if end > total {
			end = total
		}
	}

	return keys[start:end], nil
}

// searchValues searches keys and strips storage prefix. searchValues 搜索存储键并裁剪为业务值。
func (m *Manager) searchValues(ctx context.Context, pattern, prefix string, start, size int) ([]string, error) {
	keys, err := m.searchKeys(ctx, pattern, start, size)
	if err != nil {
		return nil, err
	}
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = strings.TrimPrefix(key, prefix)
	}
	return values, nil
}
