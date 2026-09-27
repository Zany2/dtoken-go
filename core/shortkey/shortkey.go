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

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
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

// ShortKey stores an interactive short credential payload. ShortKey 存储交互式短凭证载荷。
type ShortKey struct {
	Key        string         `json:"key"`                 // Key stores short credential value. Key 存储短凭证值。
	AuthType   string         `json:"authType,omitempty"`  // AuthType stores auth namespace. AuthType 存储认证命名空间。
	LoginID    string         `json:"loginId,omitempty"`   // LoginID stores confirmed subject id. LoginID 存储确认后的主体 ID。
	Device     string         `json:"device,omitempty"`    // Device stores device type. Device 存储设备类型。
	DeviceID   string         `json:"deviceId,omitempty"`  // DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	Scene      string         `json:"scene,omitempty"`     // Scene stores business scene. Scene 存储业务场景。
	SourceApp  string         `json:"sourceApp,omitempty"` // SourceApp stores issuing application. SourceApp 存储签发应用。
	TargetApp  string         `json:"targetApp,omitempty"` // TargetApp stores consuming application. TargetApp 存储目标应用。
	Scopes     []string       `json:"scopes,omitempty"`    // Scopes stores granted scopes. Scopes 存储授权范围。
	Extra      map[string]any `json:"extra,omitempty"`     // Extra stores extension data. Extra 存储扩展数据。
	CreateTime int64          `json:"createTime"`          // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	UpdateTime int64          `json:"updateTime"`          // UpdateTime stores update unix time. UpdateTime 存储更新时间戳。
	ExpiresIn  int64          `json:"expiresIn"`           // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	ExpiresAt  time.Time      `json:"expiresAt,omitempty"` // ExpiresAt stores the precise deadline; zero denotes a legacy short key. ExpiresAt 存储精确截止时间，零值表示旧版短 Key。
	Status     Status         `json:"status"`              // Status stores lifecycle state. Status 存储生命周期状态。
}

// CreateOptions defines short key creation options. CreateOptions 定义短 Key 创建选项。
type CreateOptions struct {
	LoginID   string         // LoginID stores subject id and confirms the key when set. LoginID 存储主体 ID，非空时直接确认短 Key。
	Device    string         // Device stores device type. Device 存储设备类型。
	DeviceID  string         // DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	Scene     string         // Scene stores business scene. Scene 存储业务场景。
	SourceApp string         // SourceApp stores issuing application. SourceApp 存储签发应用。
	TargetApp string         // TargetApp stores consuming application. TargetApp 存储目标应用。
	Scopes    []string       // Scopes stores granted scopes. Scopes 存储授权范围。
	Extra     map[string]any // Extra stores extension data. Extra 存储扩展数据。
	Timeout   time.Duration  // Timeout overrides default short key ttl. Timeout 覆盖默认短 Key 有效期。
}

// ConfirmOptions defines short key confirmation data. ConfirmOptions 定义短 Key 确认数据。
type ConfirmOptions struct {
	LoginID  string         // LoginID stores confirmed subject id. LoginID 存储确认后的主体 ID。
	Device   string         // Device stores confirmed device type. Device 存储确认后的设备类型。
	DeviceID string         // DeviceID stores confirmed concrete device id. DeviceID 存储确认后的具体设备 ID。
	Scopes   []string       // Scopes replaces granted scopes when set. Scopes 非空时替换授权范围。
	Extra    map[string]any // Extra replaces extension data when set. Extra 非空时替换扩展数据。
}

// ValidateOptions defines short key validation constraints. ValidateOptions 定义短 Key 校验约束。
type ValidateOptions struct {
	LoginID   string // LoginID requires matching subject id when set. LoginID 非空时要求主体 ID 匹配。
	Device    string // Device requires matching device type when set. Device 非空时要求设备类型匹配。
	DeviceID  string // DeviceID requires matching concrete device id when set. DeviceID 非空时要求具体设备 ID 匹配。
	Scene     string // Scene requires matching business scene when set. Scene 非空时要求业务场景匹配。
	SourceApp string // SourceApp requires matching issuing application when set. SourceApp 非空时要求签发应用匹配。
	TargetApp string // TargetApp requires matching consuming application when set. TargetApp 非空时要求目标应用匹配。
}

// ConsumeResult stores consumed short key data. ConsumeResult 存储短 Key 消费结果。
type ConsumeResult struct {
	ShortKey *ShortKey // ShortKey stores consumed short key payload. ShortKey 存储已消费的短 Key 载荷。
}

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

// Consume validates and consumes a confirmed short key. Consume 校验并消费已确认短 Key。
// Plain Storage provides only instance-local serialized consumption. 普通 Storage 仅提供当前实例内的串行消费。
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
