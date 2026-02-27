package nonce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"sync"
	"time"
)

// NonceManager is the nonce manager.
// NonceManager Nonce管理器
type NonceManager struct {
	authType  string          // 认证体系类型
	keyPrefix string          // 可配置的前缀
	ttl       time.Duration   // Nonce有效期
	mu        sync.RWMutex    // 并发访问读写锁
	storage   adapter.Storage // 存储适配器
}

// NewNonceManager creates a new nonce manager.
// NewNonceManager 创建新的Nonce管理器
func NewNonceManager(authType, prefix string, storage adapter.Storage, ttl time.Duration) *NonceManager {
	if ttl == 0 {
		ttl = DefaultNonceTTL // 默认5分钟
	}
	if storage == nil {
		storage = memory.NewStorage() // 如果未提供使用内存存储
	}

	return &NonceManager{
		authType:  authType,
		keyPrefix: prefix,
		storage:   storage,
		ttl:       ttl,
	}
}

// Generate generates a new nonce and stores it (using default TTL).
// Generate 生成新的nonce并存储（使用默认有效期）。
func (nm *NonceManager) Generate(ctx context.Context) (string, error) {
	return nm.GenerateWithTimeout(ctx, nm.ttl)
}

// GenerateWithTimeout generates a new nonce with a custom timeout duration.
// GenerateWithTimeout 生成新的nonce并存储，使用指定的有效期。
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

// GetTTL returns the remaining TTL of a nonce in seconds.
// GetTTL 获取nonce的剩余有效时间（秒）。
func (nm *NonceManager) GetTTL(ctx context.Context, nonce string) (int64, error) {
	if nonce == "" {
		return -2, nil
	}

	key := nm.getNonceKey(nonce)
	ttl, err := nm.storage.TTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	seconds := int64(ttl)
	switch {
	case seconds == -2:
		return -2, nil
	case seconds == -1:
		return -1, nil
	default:
		return int64(ttl.Seconds()), nil
	}
}

// Verify verifies and consumes a nonce (one-time use), returns false if nonce does not exist or has been used.
// Verify 验证nonce并消费它（一次性使用），如果nonce不存在或已使用则返回false。
func (nm *NonceManager) Verify(ctx context.Context, nonce string) bool {
	if nonce == "" {
		return false
	}

	key := nm.getNonceKey(nonce)

	nm.mu.Lock()
	defer nm.mu.Unlock()

	_, err := nm.storage.GetAndDelete(ctx, key)

	return err == nil
}

// VerifyAndConsume verifies and consumes a nonce, returns error if invalid.
// VerifyAndConsume 验证并消费nonce，无效时返回错误。
func (nm *NonceManager) VerifyAndConsume(ctx context.Context, nonce string) error {
	if !nm.Verify(ctx, nonce) {
		return derror.ErrInvalidNonce
	}
	return nil
}

// IsValid checks if a nonce is valid without consuming it.
// IsValid 检查nonce是否有效（不消费）。
func (nm *NonceManager) IsValid(ctx context.Context, nonce string) bool {
	if nonce == "" {
		return false
	}

	key := nm.getNonceKey(nonce)

	nm.mu.RLock()
	defer nm.mu.RUnlock()

	return nm.storage.Exists(ctx, key)
}

// getNonceKey generates the storage key for a nonce.
// getNonceKey 获取nonce的存储键。
func (nm *NonceManager) getNonceKey(nonce string) string {
	return nm.keyPrefix + nm.authType + NonceKeySuffix + nonce
}
