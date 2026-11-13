// @Author daixk 2025/12/22 15:56:00
package nonce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// Config defines nonce manager config Config 定义 Nonce 管理器配置
type Config struct {
	// TTL stores nonce default ttl TTL 存储 Nonce 默认有效期
	TTL time.Duration
}

// DefaultConfig returns default nonce config DefaultConfig 返回默认 Nonce 配置
func DefaultConfig() *Config {
	return &Config{TTL: DefaultNonceTTL}
}

// Validate validates nonce config Validate 验证 Nonce 配置
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	if c.TTL <= 0 {
		return fmt.Errorf("NonceConfig.TTL must be a positive duration")
	}
	return nil
}

// Clone returns a deep copy of nonce config Clone 克隆 Nonce 配置
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	copyCfg := *c
	return &copyCfg
}

// NonceManager defines nonce manager NonceManager 定义 Nonce 管理器
type NonceManager struct {
	authType  string          // authType stores auth type authType 存储认证体系类型
	keyPrefix string          // keyPrefix stores key prefix keyPrefix 存储可配置前缀
	ttl       time.Duration   // ttl stores nonce ttl ttl 存储 Nonce 有效期
	storage   adapter.Storage // storage stores storage adapter storage 存储存储适配器
}

// NewDefaultNonceManager creates nonce manager with default config NewDefaultNonceManager 使用默认配置创建 Nonce 管理器
func NewDefaultNonceManager(authType, prefix string, storage adapter.Storage) *NonceManager {
	return NewNonceManagerWithConfig(authType, prefix, storage, DefaultConfig())
}

// NewNonceManagerWithConfig creates nonce manager with config NewNonceManagerWithConfig 使用配置创建 Nonce 管理器
func NewNonceManagerWithConfig(authType, prefix string, storage adapter.Storage, cfg *Config) *NonceManager {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return NewNonceManager(authType, prefix, storage, cfg.TTL)
}

// NewNonceManager creates nonce manager NewNonceManager 创建新的 Nonce 管理器
func NewNonceManager(authType, prefix string, storage adapter.Storage, ttl time.Duration) *NonceManager {
	if ttl <= 0 {
		// Use default ttl 使用默认有效期
		ttl = DefaultNonceTTL
	}

	return &NonceManager{
		authType:  authType,
		keyPrefix: prefix,
		storage:   storage,
		ttl:       ttl,
	}
}

// TTL returns the effective default nonce ttl. TTL 返回实际生效的默认 Nonce 有效期。
func (nm *NonceManager) TTL() time.Duration {
	return nm.ttl
}

// Generate creates nonce with default ttl Generate 使用默认有效期生成并存储 nonce
func (nm *NonceManager) Generate(ctx context.Context) (string, error) {
	return nm.GenerateWithTimeout(ctx, nm.ttl)
}

// GenerateWithTimeout creates nonce with timeout GenerateWithTimeout 使用指定有效期生成并存储 nonce
func (nm *NonceManager) GenerateWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = nm.ttl
	}

	bytes := make([]byte, NonceLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(bytes)

	key := nm.getNonceKey(nonce)
	if err := nm.storage.Set(ctx, key, time.Now().Unix(), timeout); err != nil {
		return "", fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	return nonce, nil
}

// GetTTL returns remaining whole seconds, -2 when missing, or -1 without expiry. GetTTL 返回向下取整的剩余秒数，不存在返回 -2，永不过期返回 -1。
func (nm *NonceManager) GetTTL(ctx context.Context, nonce string) (int64, error) {
	if nonce == "" {
		return -2, nil
	}

	key := nm.getNonceKey(nonce)
	ttl, err := nm.storage.TTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	switch {
	case ttl == adapter.TTLNotFound:
		return -2, nil
	case ttl == adapter.TTLNoExpire:
		return -1, nil
	case ttl > 0:
		return int64(ttl / time.Second), nil
	default:
		return 0, nil
	}
}

// Verify verifies and consumes nonce Verify 验证并消费 nonce 且在不存在时返回 false
func (nm *NonceManager) Verify(ctx context.Context, nonce string) bool {
	return nm.VerifyAndConsume(ctx, nonce) == nil
}

// VerifyAndConsume verifies and consumes a nonce using truly atomic AtomicStorage.GetAndDelete; it returns errors and has no process-local fallback for distributed replay protection. VerifyAndConsume 依赖真正原子的 AtomicStorage.GetAndDelete 验证并消费 Nonce，失败返回错误；为保证分布式防重放，不提供进程内回退。
func (nm *NonceManager) VerifyAndConsume(ctx context.Context, nonce string) error {
	if nonce == "" {
		return derror.ErrInvalidNonce
	}

	key := nm.getNonceKey(nonce)

	atomicStorage, ok := nm.storage.(adapter.AtomicStorage)
	if !ok {
		return derror.ErrStorageCapabilityUnsupported
	}

	value, err := atomicStorage.GetAndDelete(ctx, key)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if value == nil {
		return derror.ErrInvalidNonce
	}

	return nil
}

// IsValid checks nonce without consuming IsValid 检查 nonce 是否有效且不消费
func (nm *NonceManager) IsValid(ctx context.Context, nonce string) bool {
	if nonce == "" {
		return false
	}

	key := nm.getNonceKey(nonce)

	return nm.storage.Exists(ctx, key)
}

// getNonceKey builds nonce storage key getNonceKey 获取 nonce 的存储键
func (nm *NonceManager) getNonceKey(nonce string) string {
	return utils.StorageNamespace(nm.keyPrefix, nm.authType) + NonceKeySuffix + nonce
}
