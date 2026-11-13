// @Author daixk 2026/06/01
package shortkey

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	capability "github.com/Zany2/dtoken-go/contract/shortkey"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

var (
	// ErrInvalidShortKey indicates an invalid or missing short key. ErrInvalidShortKey 表示短 Key 无效或不存在。
	ErrInvalidShortKey = derror.ErrInvalidShortKey
	// ErrShortKeyPending indicates the short key is not confirmed yet. ErrShortKeyPending 表示短 Key 尚未确认。
	ErrShortKeyPending = derror.ErrShortKeyPending
	// ErrShortKeyConsumed indicates a consumed short key. ErrShortKeyConsumed 表示短 Key 已消费。
	ErrShortKeyConsumed = derror.ErrShortKeyConsumed
	// ErrShortKeyRevoked indicates a revoked short key. ErrShortKeyRevoked 表示短 Key 已撤销。
	ErrShortKeyRevoked = derror.ErrShortKeyRevoked
	// ErrShortKeyExpired indicates an expired short key. ErrShortKeyExpired 表示短 Key 已过期。
	ErrShortKeyExpired = derror.ErrShortKeyExpired
	// ErrShortKeyMismatch indicates short key constraints do not match. ErrShortKeyMismatch 表示短 Key 约束不匹配。
	ErrShortKeyMismatch = derror.ErrShortKeyMismatch
)

// Config defines short key manager config. Config 定义短 Key 管理器配置。
type Config struct {
	// TTL stores default short key ttl. TTL 存储短 Key 默认有效期。
	TTL time.Duration
	// Length stores generated key length. Length 存储生成的短 Key 长度。
	Length int
	// MaxGenerateRetries stores collision retry count. MaxGenerateRetries 存储碰撞重试次数。
	MaxGenerateRetries int
}

// DefaultConfig returns default short key config. DefaultConfig 返回默认短 Key 配置。
func DefaultConfig() *Config {
	return &Config{
		TTL:                DefaultTTL,
		Length:             DefaultLength,
		MaxGenerateRetries: 8,
	}
}

// Validate validates short key config. Validate 校验短 Key 配置。
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	if c.TTL <= 0 {
		return fmt.Errorf("ShortKeyConfig.TTL must be a positive duration")
	}
	if c.Length <= 0 {
		return fmt.Errorf("ShortKeyConfig.Length must be positive")
	}
	if c.MaxGenerateRetries <= 0 {
		return fmt.Errorf("ShortKeyConfig.MaxGenerateRetries must be positive")
	}
	return nil
}

// Clone returns a deep copy of short key config. Clone 返回短 Key 配置副本。
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	copyCfg := *c
	return &copyCfg
}

// ShortKey is the public capability data contract. ShortKey 是公开能力数据契约。
type ShortKey = capability.ShortKey

// CreateOptions is the public capability data contract. CreateOptions 是公开能力数据契约。
type CreateOptions = capability.CreateOptions

// ConfirmOptions is the public capability data contract. ConfirmOptions 是公开能力数据契约。
type ConfirmOptions = capability.ConfirmOptions

// ValidateOptions is the public capability data contract. ValidateOptions 是公开能力数据契约。
type ValidateOptions = capability.ValidateOptions

// ConsumeResult is the public capability data contract. ConsumeResult 是公开能力数据契约。
type ConsumeResult = capability.ConsumeResult

// Manager handles short key operations. Manager 处理短 Key 操作。
type Manager struct {
	authType           string
	keyPrefix          string
	ttl                time.Duration
	length             int
	maxGenerateRetries int
	storage            adapter.Storage
	serializer         adapter.Codec
	mu                 sync.Mutex // Serializes creation writes and state transitions within this manager. 在当前管理器内串行化创建写入与状态流转。
}

// NewDefaultManager creates short key manager with default config. NewDefaultManager 使用默认配置创建短 Key 管理器。
func NewDefaultManager(authType, prefix string, storage adapter.Storage, serializer adapter.Codec) *Manager {
	return NewManagerWithConfig(authType, prefix, storage, serializer, DefaultConfig())
}

// NewManagerWithConfig creates short key manager with config. NewManagerWithConfig 使用配置创建短 Key 管理器。
func NewManagerWithConfig(authType, prefix string, storage adapter.Storage, serializer adapter.Codec, cfg *Config) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	length := cfg.Length
	if length <= 0 {
		length = DefaultLength
	}
	retries := cfg.MaxGenerateRetries
	if retries <= 0 {
		retries = 8
	}
	return &Manager{
		authType:           authType,
		keyPrefix:          prefix,
		ttl:                ttl,
		length:             length,
		maxGenerateRetries: retries,
		storage:            storage,
		serializer:         serializer,
	}
}

// Create creates a pending short key. Create 创建待确认短 Key。
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*ShortKey, error) {
	return m.CreateWithTimeout(ctx, opts, opts.Timeout)
}

// CreateWithTimeout creates a pending short key with timeout. CreateWithTimeout 使用指定有效期创建待确认短 Key。
func (m *Manager) CreateWithTimeout(ctx context.Context, opts CreateOptions, timeout time.Duration) (*ShortKey, error) {
	if timeout <= 0 {
		timeout = m.ttl
	}

	for i := 0; i < m.maxGenerateRetries; i++ {
		generated, err := generateKey(m.length)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		shortKey := &ShortKey{
			Key:        generated,
			AuthType:   m.authType,
			LoginID:    opts.LoginID,
			Device:     opts.Device,
			DeviceID:   opts.DeviceID,
			Scene:      opts.Scene,
			SourceApp:  opts.SourceApp,
			TargetApp:  opts.TargetApp,
			Scopes:     append([]string(nil), opts.Scopes...),
			Extra:      cloneMap(opts.Extra),
			CreateTime: now.Unix(),
			UpdateTime: now.Unix(),
			ExpiresIn:  durationSeconds(timeout),
			ExpiresAt:  now.Add(timeout),
			Status:     StatusPending,
		}
		if shortKey.LoginID != "" {
			shortKey.Status = StatusConfirmed
		}
		ok, err := m.saveIfAbsent(ctx, shortKey, timeout)
		if err != nil {
			return nil, err
		}
		if ok {
			return shortKey, nil
		}
	}
	return nil, fmt.Errorf("short key collision retry limit reached")
}

// Confirm confirms a pending short key. Confirm 确认待处理短 Key。
func (m *Manager) Confirm(ctx context.Context, key string, opts ConfirmOptions) (*ShortKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	shortKey, err := m.get(ctx, key)
	if err != nil {
		return nil, err
	}
	if err = m.checkUsable(shortKey, time.Now()); err != nil {
		return nil, err
	}
	if shortKey.Status != StatusPending {
		return nil, ErrShortKeyMismatch
	}
	if opts.LoginID != "" {
		shortKey.LoginID = opts.LoginID
	}
	if opts.Device != "" {
		shortKey.Device = opts.Device
	}
	if opts.DeviceID != "" {
		shortKey.DeviceID = opts.DeviceID
	}
	if opts.Scopes != nil {
		shortKey.Scopes = append([]string(nil), opts.Scopes...)
	}
	if opts.Extra != nil {
		shortKey.Extra = cloneMap(opts.Extra)
	}
	shortKey.Status = StatusConfirmed
	shortKey.UpdateTime = time.Now().Unix()
	ttl := remainingDuration(shortKey)
	if ttl <= 0 {
		return nil, ErrShortKeyExpired
	}
	if err = m.save(ctx, shortKey, ttl); err != nil {
		return nil, err
	}
	return shortKey, nil
}

// Validate validates a short key without consuming it. Validate 校验短 Key 但不消费。
func (m *Manager) Validate(ctx context.Context, key string, opts ...ValidateOptions) (*ShortKey, error) {
	shortKey, err := m.get(ctx, key)
	if err != nil {
		return nil, err
	}
	if err = m.checkUsable(shortKey, time.Now()); err != nil {
		return nil, err
	}
	if shortKey.Status == StatusPending {
		return nil, ErrShortKeyPending
	}
	if len(opts) > 0 {
		if err = checkConstraints(shortKey, opts[0]); err != nil {
			return nil, err
		}
	}
	return shortKey, nil
}

// Consume validates and consumes a confirmed short key; plain Storage serializes consumption only within this manager instance. Consume 校验并消费已确认的短 Key，普通 Storage 仅在当前管理器实例内串行消费。
func (m *Manager) Consume(ctx context.Context, key string, opts ...ValidateOptions) (*ConsumeResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	shortKey, err := m.Validate(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	// Prefer atomic removal and revalidate the value actually removed. 优先原子读删，并重新校验实际取出的载荷。
	if atomicStorage, ok := m.storage.(adapter.AtomicStorage); ok {
		value, err := atomicStorage.GetAndDelete(ctx, m.getKey(key))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if value == nil {
			return nil, ErrInvalidShortKey
		}
		shortKey, err = m.decode(value, key)
		if err != nil {
			return nil, err
		}
		if err = m.checkUsable(shortKey, time.Now()); err != nil {
			if ttl := remainingDuration(shortKey); ttl > 0 {
				_ = m.save(ctx, shortKey, ttl)
			}
			return nil, err
		}
		if shortKey.Status == StatusPending {
			if ttl := remainingDuration(shortKey); ttl > 0 {
				_ = m.save(ctx, shortKey, ttl)
			}
			return nil, ErrShortKeyPending
		}
		if len(opts) > 0 {
			if err = checkConstraints(shortKey, opts[0]); err != nil {
				if ttl := remainingDuration(shortKey); ttl > 0 {
					_ = m.save(ctx, shortKey, ttl)
				}
				return nil, err
			}
		}
	} else if err = m.storage.Delete(ctx, m.getKey(key)); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	} else if err = m.checkUsable(shortKey, time.Now()); err != nil {
		return nil, err
	}

	shortKey.Status = StatusConsumed
	shortKey.UpdateTime = time.Now().Unix()
	if ttl := remainingDuration(shortKey); ttl > 0 {
		if err = m.save(ctx, shortKey, ttl); err != nil {
			return nil, err
		}
	}
	return &ConsumeResult{ShortKey: shortKey}, nil
}

// Revoke revokes a short key. Revoke 撤销短 Key。
func (m *Manager) Revoke(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	shortKey, err := m.get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrInvalidShortKey) {
			return nil
		}
		return err
	}
	if err = m.checkUsable(shortKey, time.Now()); err != nil {
		switch {
		case errors.Is(err, ErrShortKeyConsumed), errors.Is(err, ErrShortKeyRevoked), errors.Is(err, ErrShortKeyExpired):
			return nil
		default:
			return err
		}
	}
	shortKey.Status = StatusRevoked
	shortKey.UpdateTime = time.Now().Unix()
	ttl := remainingDuration(shortKey)
	if ttl <= 0 {
		if err = m.storage.Delete(ctx, m.getKey(key)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return nil
	}
	return m.save(ctx, shortKey, ttl)
}

// Status returns short key lifecycle status. Status 返回短 Key 生命周期状态。
func (m *Manager) Status(ctx context.Context, key string) (Status, error) {
	shortKey, err := m.get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrInvalidShortKey) {
			return StatusInvalid, nil
		}
		return StatusInvalid, err
	}
	if err = m.checkUsable(shortKey, time.Now()); err != nil {
		switch {
		case errors.Is(err, ErrShortKeyConsumed):
			return StatusConsumed, nil
		case errors.Is(err, ErrShortKeyRevoked):
			return StatusRevoked, nil
		case errors.Is(err, ErrShortKeyExpired):
			return StatusExpired, nil
		default:
			return StatusInvalid, nil
		}
	}
	return shortKey.Status, nil
}

// GetTTL returns short key ttl in seconds. GetTTL 返回短 Key 剩余有效秒数。
func (m *Manager) GetTTL(ctx context.Context, key string) (int64, error) {
	if key == "" {
		return -2, nil
	}
	ttl, err := m.storage.TTL(ctx, m.getKey(key))
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

// save serializes and persists short key state. save 序列化并持久化短 Key 状态。
func (m *Manager) save(ctx context.Context, shortKey *ShortKey, timeout time.Duration) error {
	encoded, err := m.serializer.Encode(shortKey)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Account for encoding time before writing the remaining lifetime. 写入剩余有效期前计入编码耗时。
	timeout, err = storageTimeout(shortKey, timeout)
	if err != nil {
		return err
	}

	if err = m.storage.Set(ctx, m.getKey(shortKey.Key), encoded, timeout); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// saveIfAbsent persists a short key only when its storage key is absent. saveIfAbsent 仅在存储键不存在时持久化短 Key。
func (m *Manager) saveIfAbsent(ctx context.Context, shortKey *ShortKey, timeout time.Duration) (bool, error) {
	encoded, err := m.serializer.Encode(shortKey)
	if err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Serialize creation with state transitions, including the consumption removal window. 将创建与状态流转串行化，覆盖消费读删期间的空键窗口。
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(shortKey.Key)
	if atomicStorage, ok := m.storage.(adapter.AtomicStorage); ok {
		timeout, err = storageTimeout(shortKey, timeout)
		if err != nil {
			return false, err
		}
		ok, err := atomicStorage.SetIfAbsent(ctx, key, encoded, timeout)
		if err != nil {
			return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return ok, nil
	}

	// Preserve collision-check errors in the ordinary Get/Set fallback. 普通 Get/Set 回退流程保留碰撞检查错误。
	existing, err := m.storage.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if existing != nil {
		return false, nil
	}
	timeout, err = storageTimeout(shortKey, timeout)
	if err != nil {
		return false, err
	}
	if err = m.storage.Set(ctx, key, encoded, timeout); err != nil {
		return false, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return true, nil
}

// get loads a short key by its credential value. get 根据凭证值加载短 Key。
func (m *Manager) get(ctx context.Context, key string) (*ShortKey, error) {
	if key == "" {
		return nil, ErrInvalidShortKey
	}
	data, err := m.storage.Get(ctx, m.getKey(key))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrInvalidShortKey
	}
	return m.decode(data, key)
}

// decode converts a stored value and verifies its storage identity. decode 转换存储值并校验其存储身份。
func (m *Manager) decode(value any, key string) (*ShortKey, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var shortKey ShortKey
	if err = m.serializer.Decode(rawData, &shortKey); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if shortKey.Key != key || shortKey.AuthType != m.authType {
		return nil, ErrInvalidShortKey
	}
	return &shortKey, nil
}

// checkUsable validates short key state and expiration. checkUsable 校验短 Key 状态与有效期。
func (m *Manager) checkUsable(shortKey *ShortKey, now time.Time) error {
	if shortKey == nil || shortKey.Key == "" {
		return ErrInvalidShortKey
	}
	switch shortKey.Status {
	case StatusPending, StatusConfirmed:
	case StatusConsumed:
		return ErrShortKeyConsumed
	case StatusRevoked:
		return ErrShortKeyRevoked
	case StatusExpired:
		return ErrShortKeyExpired
	default:
		return ErrInvalidShortKey
	}
	if expiresAt := expirationTime(shortKey); !expiresAt.IsZero() && !now.Before(expiresAt) {
		return ErrShortKeyExpired
	}
	return nil
}

// getKey builds the storage key for a short key. getKey 构建短 Key 存储键。
func (m *Manager) getKey(key string) string {
	return utils.StorageNamespace(m.keyPrefix, m.authType) + KeySuffix + key
}

// checkConstraints validates short key binding constraints. checkConstraints 校验短 Key 绑定约束。
func checkConstraints(shortKey *ShortKey, opts ValidateOptions) error {
	if opts.LoginID != "" && shortKey.LoginID != opts.LoginID {
		return ErrShortKeyMismatch
	}
	if opts.Device != "" && shortKey.Device != opts.Device {
		return ErrShortKeyMismatch
	}
	if opts.DeviceID != "" && shortKey.DeviceID != opts.DeviceID {
		return ErrShortKeyMismatch
	}
	if opts.Scene != "" && shortKey.Scene != opts.Scene {
		return ErrShortKeyMismatch
	}
	if opts.SourceApp != "" && shortKey.SourceApp != opts.SourceApp {
		return ErrShortKeyMismatch
	}
	if opts.TargetApp != "" && shortKey.TargetApp != opts.TargetApp {
		return ErrShortKeyMismatch
	}
	return nil
}

// expirationTime reads the precise deadline with compatibility for legacy payloads. expirationTime 读取精确截止时间，并兼容旧版载荷。
func expirationTime(shortKey *ShortKey) time.Time {
	if shortKey == nil {
		return time.Time{}
	}
	if !shortKey.ExpiresAt.IsZero() {
		return shortKey.ExpiresAt
	}
	if shortKey.ExpiresIn > 0 {
		return time.Unix(shortKey.CreateTime+shortKey.ExpiresIn, 0)
	}
	return time.Time{}
}

// remainingDuration calculates the remaining short key lifetime. remainingDuration 计算短 Key 剩余有效期。
func remainingDuration(shortKey *ShortKey) time.Duration {
	ttl := time.Until(expirationTime(shortKey))
	if ttl <= 0 {
		return 0
	}
	return ttl
}

// storageTimeout caps writes at the precise deadline and rejects expired writes. storageTimeout 按精确截止时间限制写入 TTL，并拒绝过期写入。
func storageTimeout(shortKey *ShortKey, timeout time.Duration) (time.Duration, error) {
	if !shortKey.ExpiresAt.IsZero() {
		ttl := remainingDuration(shortKey)
		if ttl <= 0 {
			return 0, ErrShortKeyExpired
		}
		if timeout <= 0 || ttl < timeout {
			return ttl, nil
		}
	}
	return timeout, nil
}

// durationSeconds rounds a positive duration up to whole seconds. durationSeconds 将正时长向上取整为秒。
func durationSeconds(duration time.Duration) int64 {
	seconds := duration / time.Second
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return int64(seconds)
}

// generateKey creates a cryptographically secure short key. generateKey 生成密码学安全的短 Key。
func generateKey(length int) (string, error) {
	result := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range result {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = alphabet[index.Int64()]
	}
	return string(result), nil
}

// toBytes converts supported storage values to bytes. toBytes 将受支持的存储值转换为字节切片。
func toBytes(value any) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case byte:
		return []byte{v}, nil
	case rune:
		return []byte(string(v)), nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", value)
	}
}

// cloneMap copies extension data to avoid shared mutation. cloneMap 复制扩展数据以避免共享修改。
func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]any, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
