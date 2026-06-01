// @Author daixk 2026/06/01
package shortkey

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
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
	DeviceId   string         `json:"deviceId,omitempty"`  // DeviceId stores concrete device id. DeviceId 存储具体设备 ID。
	Scene      string         `json:"scene,omitempty"`     // Scene stores business scene. Scene 存储业务场景。
	SourceApp  string         `json:"sourceApp,omitempty"` // SourceApp stores issuing application. SourceApp 存储签发应用。
	TargetApp  string         `json:"targetApp,omitempty"` // TargetApp stores consuming application. TargetApp 存储目标应用。
	Scopes     []string       `json:"scopes,omitempty"`    // Scopes stores granted scopes. Scopes 存储授权范围。
	Extra      map[string]any `json:"extra,omitempty"`     // Extra stores extension data. Extra 存储扩展数据。
	CreateTime int64          `json:"createTime"`          // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	UpdateTime int64          `json:"updateTime"`          // UpdateTime stores update unix time. UpdateTime 存储更新时间戳。
	ExpiresIn  int64          `json:"expiresIn"`           // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Status     Status         `json:"status"`              // Status stores lifecycle state. Status 存储生命周期状态。
}

// CreateOptions defines short key creation options. CreateOptions 定义短 Key 创建选项。
type CreateOptions struct {
	LoginID   string
	Device    string
	DeviceId  string
	Scene     string
	SourceApp string
	TargetApp string
	Scopes    []string
	Extra     map[string]any
	Timeout   time.Duration
}

// ConfirmOptions defines short key confirmation data. ConfirmOptions 定义短 Key 确认数据。
type ConfirmOptions struct {
	LoginID  string
	Device   string
	DeviceId string
	Scopes   []string
	Extra    map[string]any
}

// ValidateOptions defines short key validation constraints. ValidateOptions 定义短 Key 校验约束。
type ValidateOptions struct {
	LoginID   string
	Device    string
	DeviceId  string
	Scene     string
	SourceApp string
	TargetApp string
}

// ConsumeResult stores consumed short key data. ConsumeResult 存储短 Key 消费结果。
type ConsumeResult struct {
	ShortKey *ShortKey
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
	var key string
	for i := 0; i < m.maxGenerateRetries; i++ {
		generated, err := generateKey(m.length)
		if err != nil {
			return nil, err
		}
		if !m.storage.Exists(ctx, m.getKey(generated)) {
			key = generated
			break
		}
	}
	if key == "" {
		return nil, fmt.Errorf("short key collision retry limit reached")
	}
	now := time.Now().Unix()
	shortKey := &ShortKey{
		Key:        key,
		AuthType:   m.authType,
		LoginID:    opts.LoginID,
		Device:     opts.Device,
		DeviceId:   opts.DeviceId,
		Scene:      opts.Scene,
		SourceApp:  opts.SourceApp,
		TargetApp:  opts.TargetApp,
		Scopes:     append([]string(nil), opts.Scopes...),
		Extra:      cloneMap(opts.Extra),
		CreateTime: now,
		UpdateTime: now,
		ExpiresIn:  int64(timeout.Seconds()),
		Status:     StatusPending,
	}
	if shortKey.LoginID != "" {
		shortKey.Status = StatusConfirmed
	}
	if err := m.save(ctx, shortKey, timeout); err != nil {
		return nil, err
	}
	return shortKey, nil
}

// Confirm confirms a pending short key. Confirm 确认待处理短 Key。
func (m *Manager) Confirm(ctx context.Context, key string, opts ConfirmOptions) (*ShortKey, error) {
	shortKey, err := m.get(ctx, key)
	if err != nil {
		return nil, err
	}
	if err = m.checkUsable(shortKey); err != nil {
		return nil, err
	}
	if shortKey.Status != StatusPending && shortKey.Status != StatusConfirmed {
		return nil, ErrInvalidShortKey
	}
	if opts.LoginID != "" {
		shortKey.LoginID = opts.LoginID
	}
	if opts.Device != "" {
		shortKey.Device = opts.Device
	}
	if opts.DeviceId != "" {
		shortKey.DeviceId = opts.DeviceId
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
	if err = m.checkUsable(shortKey); err != nil {
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
func (m *Manager) Consume(ctx context.Context, key string, opts ...ValidateOptions) (*ConsumeResult, error) {
	if _, err := m.Validate(ctx, key, opts...); err != nil {
		return nil, err
	}
	atomicStorage, ok := m.storage.(adapter.AtomicStorage)
	if !ok {
		return nil, derror.ErrStorageCapabilityUnsupported
	}
	value, err := atomicStorage.GetAndDelete(ctx, m.getKey(key))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if value == nil {
		return nil, ErrInvalidShortKey
	}
	shortKey, err := m.decode(value)
	if err != nil {
		return nil, err
	}
	if err = m.checkUsable(shortKey); err != nil {
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
	shortKey.Status = StatusConsumed
	shortKey.UpdateTime = time.Now().Unix()
	if ttl := remainingDuration(shortKey); ttl > 0 {
		_ = m.save(ctx, shortKey, ttl)
	}
	return &ConsumeResult{ShortKey: shortKey}, nil
}

// Revoke revokes a short key. Revoke 撤销短 Key。
func (m *Manager) Revoke(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	shortKey, err := m.get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrInvalidShortKey) {
			return nil
		}
		return err
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
	if err = m.checkUsable(shortKey); err != nil {
		switch {
		case errors.Is(err, ErrShortKeyPending):
			return StatusPending, nil
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

func (m *Manager) save(ctx context.Context, shortKey *ShortKey, timeout time.Duration) error {
	encoded, err := m.serializer.Encode(shortKey)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if err = m.storage.Set(ctx, m.getKey(shortKey.Key), encoded, timeout); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

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
	return m.decode(data)
}

func (m *Manager) decode(value any) (*ShortKey, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var shortKey ShortKey
	if err = m.serializer.Decode(rawData, &shortKey); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	return &shortKey, nil
}

func (m *Manager) checkUsable(shortKey *ShortKey) error {
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
	if shortKey.ExpiresIn > 0 && time.Now().Unix() > shortKey.CreateTime+shortKey.ExpiresIn {
		return ErrShortKeyExpired
	}
	return nil
}

func (m *Manager) getKey(key string) string {
	return m.keyPrefix + m.authType + KeySuffix + key
}

func checkConstraints(shortKey *ShortKey, opts ValidateOptions) error {
	if opts.LoginID != "" && shortKey.LoginID != opts.LoginID {
		return ErrShortKeyMismatch
	}
	if opts.Device != "" && shortKey.Device != opts.Device {
		return ErrShortKeyMismatch
	}
	if opts.DeviceId != "" && shortKey.DeviceId != opts.DeviceId {
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

func remainingDuration(shortKey *ShortKey) time.Duration {
	if shortKey == nil || shortKey.ExpiresIn <= 0 {
		return 0
	}
	expiresAt := time.Unix(shortKey.CreateTime+shortKey.ExpiresIn, 0)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return 0
	}
	return ttl
}

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
