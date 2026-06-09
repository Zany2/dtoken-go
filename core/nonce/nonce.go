// @Author daixk 2025/12/22 15:56:00
package nonce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
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
	mu        sync.RWMutex    // mu guards concurrent access mu 保护并发读写
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
		ttl = DefaultNonceTTL // Use default ttl 使用默认有效期
	}

	return &NonceManager{
		authType:  authType,
		keyPrefix: prefix,
		storage:   storage,
		ttl:       ttl,
	}
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

// GetTTL gets remaining nonce ttl GetTTL 获取 nonce 的剩余有效时间秒数
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
		return int64(ttl.Seconds()), nil
	default:
		return 0, nil
	}
}

// Verify verifies and consumes nonce Verify 验证并消费 nonce 且在不存在时返回 false
func (nm *NonceManager) Verify(ctx context.Context, nonce string) bool {
	return nm.VerifyAndConsume(ctx, nonce) == nil
}

// VerifyAndConsume verifies nonce with error VerifyAndConsume 验证并消费 nonce 且在无效时返回错误
func (nm *NonceManager) VerifyAndConsume(ctx context.Context, nonce string) error {
	if nonce == "" {
		return derror.ErrInvalidNonce
	}

	key := nm.getNonceKey(nonce)

	nm.mu.Lock()
	defer nm.mu.Unlock()

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

	nm.mu.RLock()
	defer nm.mu.RUnlock()

	return nm.storage.Exists(ctx, key)
}

// getNonceKey builds nonce storage key getNonceKey 获取 nonce 的存储键
func (nm *NonceManager) getNonceKey(nonce string) string {
	return nm.keyPrefix + nm.authType + NonceKeySuffix + nonce
}
