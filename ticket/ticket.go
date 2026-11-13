// @Author daixk 2026/06/01
package ticket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	capability "github.com/Zany2/dtoken-go/contract/ticket"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

var (
	// ErrInvalidTicket indicates an invalid or missing ticket. ErrInvalidTicket 表示 Ticket 无效或不存在。
	ErrInvalidTicket = derror.ErrInvalidTicket
	// ErrTicketConsumed indicates a consumed ticket. ErrTicketConsumed 表示 Ticket 已消费。
	ErrTicketConsumed = derror.ErrTicketConsumed
	// ErrTicketRevoked indicates a revoked ticket. ErrTicketRevoked 表示 Ticket 已撤销。
	ErrTicketRevoked = derror.ErrTicketRevoked
	// ErrTicketExpired indicates an expired ticket. ErrTicketExpired 表示 Ticket 已过期。
	ErrTicketExpired = derror.ErrTicketExpired
	// ErrTicketMismatch indicates ticket constraints do not match. ErrTicketMismatch 表示 Ticket 约束不匹配。
	ErrTicketMismatch = derror.ErrTicketMismatch
)

// Config defines ticket manager config. Config 定义 Ticket 管理器配置。
type Config struct {
	// TTL stores default ticket ttl. TTL 存储 Ticket 默认有效期。
	TTL time.Duration
}

// DefaultConfig returns default ticket config. DefaultConfig 返回默认 Ticket 配置。
func DefaultConfig() *Config {
	return &Config{TTL: DefaultTicketTTL}
}

// Validate validates ticket config. Validate 校验 Ticket 配置。
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	if c.TTL <= 0 {
		return fmt.Errorf("TicketConfig.TTL must be a positive duration")
	}
	return nil
}

// Clone returns a deep copy of ticket config. Clone 返回 Ticket 配置副本。
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	copyCfg := *c
	return &copyCfg
}

// Ticket is the public capability data contract. Ticket 是公开能力数据契约。
type Ticket = capability.Ticket

// CreateOptions is the public capability data contract. CreateOptions 是公开能力数据契约。
type CreateOptions = capability.CreateOptions

// ValidateOptions is the public capability data contract. ValidateOptions 是公开能力数据契约。
type ValidateOptions = capability.ValidateOptions

// ConsumeResult is the public capability data contract. ConsumeResult 是公开能力数据契约。
type ConsumeResult = capability.ConsumeResult

// Manager handles temporary ticket operations. Manager 处理临时 Ticket 操作。
type Manager struct {
	authType   string
	keyPrefix  string
	ttl        time.Duration
	storage    adapter.Storage
	serializer adapter.Codec
	// stateMu serializes fallback consumption and revocation within this manager. stateMu 在当前管理器内串行化回退消费与撤销。
	stateMu sync.Mutex
}

// NewDefaultManager creates ticket manager with default config. NewDefaultManager 使用默认配置创建 Ticket 管理器。
func NewDefaultManager(authType, prefix string, storage adapter.Storage, serializer adapter.Codec) *Manager {
	return NewManagerWithConfig(authType, prefix, storage, serializer, DefaultConfig())
}

// NewManagerWithConfig creates ticket manager with config. NewManagerWithConfig 使用配置创建 Ticket 管理器。
func NewManagerWithConfig(authType, prefix string, storage adapter.Storage, serializer adapter.Codec, cfg *Config) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultTicketTTL
	}
	return &Manager{
		authType:   authType,
		keyPrefix:  prefix,
		ttl:        ttl,
		storage:    storage,
		serializer: serializer,
	}
}

// Create creates a ticket with the default TTL. Create 使用默认有效期创建 Ticket。
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Ticket, error) {
	return m.CreateWithTimeout(ctx, opts, opts.Timeout)
}

// CreateWithTimeout creates a ticket with the specified timeout. CreateWithTimeout 使用指定有效期创建 Ticket。
func (m *Manager) CreateWithTimeout(ctx context.Context, opts CreateOptions, timeout time.Duration) (*Ticket, error) {
	if timeout <= 0 {
		timeout = m.ttl
	}
	value, err := generateRandomValue(TicketLength)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	ticket := &Ticket{
		Ticket:     value,
		AuthType:   m.authType,
		LoginID:    opts.LoginID,
		Device:     opts.Device,
		DeviceID:   opts.DeviceID,
		Source:     opts.Source,
		SourceApp:  opts.SourceApp,
		TargetApp:  opts.TargetApp,
		Scopes:     append([]string(nil), opts.Scopes...),
		Extra:      cloneMap(opts.Extra),
		CreateTime: now.Unix(),
		ExpiresIn:  durationSeconds(timeout),
		ExpiresAt:  now.Add(timeout),
		Status:     StatusValid,
	}
	if err = m.save(ctx, ticket, timeout); err != nil {
		return nil, err
	}
	return ticket, nil
}

// Validate validates a ticket without consuming it. Validate 校验 Ticket 但不消费。
func (m *Manager) Validate(ctx context.Context, ticketValue string, opts ...ValidateOptions) (*Ticket, error) {
	ticket, err := m.get(ctx, ticketValue)
	if err != nil {
		return nil, err
	}
	if err = m.checkAlive(ticket, time.Now()); err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		if err = checkConstraints(ticket, opts[0]); err != nil {
			return nil, err
		}
	}
	return ticket, nil
}

// Consume validates and consumes a one-time ticket; plain Storage serializes locally, while cross-instance consumption requires AtomicStorage. Consume 校验并消费一次性 Ticket，普通 Storage 仅在当前管理器内串行处理，跨实例消费需要 AtomicStorage。
func (m *Manager) Consume(ctx context.Context, ticketValue string, opts ...ValidateOptions) (*ConsumeResult, error) {
	atomicStorage, ok := m.storage.(adapter.AtomicStorage)
	if !ok {
		m.stateMu.Lock()
		defer m.stateMu.Unlock()
	}

	ticket, err := m.Validate(ctx, ticketValue, opts...)
	if err != nil {
		return nil, err
	}

	// Prefer atomic removal, and revalidate the value actually removed. 优先原子读删，并重新校验实际取出的载荷。
	if ok {
		value, err := atomicStorage.GetAndDelete(ctx, m.getTicketKey(ticketValue))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if value == nil {
			return nil, ErrInvalidTicket
		}
		ticket, err = m.decode(value, ticketValue)
		if err != nil {
			return nil, err
		}
		if err = m.checkAlive(ticket, time.Now()); err != nil {
			if ttl := remainingDuration(ticket); ttl > 0 {
				_ = m.save(ctx, ticket, ttl)
			}
			return nil, err
		}
		if len(opts) > 0 {
			if err = checkConstraints(ticket, opts[0]); err != nil {
				if ttl := remainingDuration(ticket); ttl > 0 {
					_ = m.save(ctx, ticket, ttl)
				}
				return nil, err
			}
		}
	} else if err = m.storage.Delete(ctx, m.getTicketKey(ticketValue)); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	} else if err = m.checkAlive(ticket, time.Now()); err != nil {
		return nil, err
	}

	ticket.Status = StatusConsumed
	if ttl := remainingDuration(ticket); ttl > 0 {
		if err = m.save(ctx, ticket, ttl); err != nil {
			return nil, err
		}
	}
	return &ConsumeResult{Ticket: ticket}, nil
}

// Revoke revokes a ticket. Revoke 撤销 Ticket。
func (m *Manager) Revoke(ctx context.Context, ticketValue string) error {
	if ticketValue == "" {
		return nil
	}
	if _, ok := m.storage.(adapter.AtomicStorage); !ok {
		m.stateMu.Lock()
		defer m.stateMu.Unlock()
	}

	ticket, err := m.get(ctx, ticketValue)
	if err != nil {
		if errors.Is(err, ErrInvalidTicket) {
			return nil
		}
		return err
	}
	if err = m.checkAlive(ticket, time.Now()); err != nil {
		switch {
		case errors.Is(err, ErrTicketConsumed), errors.Is(err, ErrTicketRevoked), errors.Is(err, ErrTicketExpired):
			return nil
		default:
			return err
		}
	}
	ticket.Status = StatusRevoked
	ttl := remainingDuration(ticket)
	if ttl <= 0 {
		if err = m.storage.Delete(ctx, m.getTicketKey(ticketValue)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return nil
	}
	if err = m.save(ctx, ticket, ttl); err != nil {
		return err
	}
	return nil
}

// Status returns ticket status. Status 返回 Ticket 状态。
func (m *Manager) Status(ctx context.Context, ticketValue string) (Status, error) {
	ticket, err := m.get(ctx, ticketValue)
	if err != nil {
		if errors.Is(err, ErrInvalidTicket) {
			return StatusInvalid, nil
		}
		return StatusInvalid, err
	}
	if err = m.checkAlive(ticket, time.Now()); err != nil {
		switch {
		case errors.Is(err, ErrTicketConsumed):
			return StatusConsumed, nil
		case errors.Is(err, ErrTicketRevoked):
			return StatusRevoked, nil
		case errors.Is(err, ErrTicketExpired):
			return StatusExpired, nil
		default:
			return StatusInvalid, nil
		}
	}
	return StatusValid, nil
}

// GetTTL returns ticket ttl in seconds. GetTTL 返回 Ticket 剩余有效秒数。
func (m *Manager) GetTTL(ctx context.Context, ticketValue string) (int64, error) {
	if ticketValue == "" {
		return -2, nil
	}
	ttl, err := m.storage.TTL(ctx, m.getTicketKey(ticketValue))
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

// save serializes and persists ticket state. save 序列化并持久化 Ticket 状态。
func (m *Manager) save(ctx context.Context, ticket *Ticket, timeout time.Duration) error {
	encoded, err := m.serializer.Encode(ticket)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}

	// Clamp storage TTL after encoding so state writes cannot extend the precise deadline. 编码后按精确截止时间限制存储 TTL，避免状态回写延长有效期。
	if !ticket.ExpiresAt.IsZero() {
		ttl := remainingDuration(ticket)
		if ttl <= 0 {
			return ErrTicketExpired
		}
		if timeout <= 0 || ttl < timeout {
			timeout = ttl
		}
	}

	if err = m.storage.Set(ctx, m.getTicketKey(ticket.Ticket), encoded, timeout); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// get loads a ticket by its credential value. get 根据凭证值加载 Ticket。
func (m *Manager) get(ctx context.Context, ticketValue string) (*Ticket, error) {
	if ticketValue == "" {
		return nil, ErrInvalidTicket
	}
	data, err := m.storage.Get(ctx, m.getTicketKey(ticketValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrInvalidTicket
	}
	return m.decode(data, ticketValue)
}

// decode converts a stored value and verifies its storage identity. decode 转换存储值并校验其存储身份。
func (m *Manager) decode(value any, ticketValue string) (*Ticket, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var ticket Ticket
	if err = m.serializer.Decode(rawData, &ticket); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if ticket.Ticket != ticketValue || ticket.AuthType != m.authType {
		return nil, ErrInvalidTicket
	}
	return &ticket, nil
}

// checkAlive validates ticket state and expiration. checkAlive 校验 Ticket 状态与有效期。
func (m *Manager) checkAlive(ticket *Ticket, now time.Time) error {
	if ticket == nil || ticket.Ticket == "" {
		return ErrInvalidTicket
	}
	switch ticket.Status {
	case StatusValid:
	case StatusConsumed:
		return ErrTicketConsumed
	case StatusRevoked:
		return ErrTicketRevoked
	case StatusExpired:
		return ErrTicketExpired
	default:
		return ErrInvalidTicket
	}
	if expiresAt := expirationTime(ticket); !expiresAt.IsZero() && !now.Before(expiresAt) {
		return ErrTicketExpired
	}
	return nil
}

// getTicketKey builds the storage key for a ticket. getTicketKey 构建 Ticket 存储键。
func (m *Manager) getTicketKey(ticketValue string) string {
	return utils.StorageNamespace(m.keyPrefix, m.authType) + TicketKeySuffix + ticketValue
}

// checkConstraints validates ticket binding constraints. checkConstraints 校验 Ticket 绑定约束。
func checkConstraints(ticket *Ticket, opts ValidateOptions) error {
	if opts.LoginID != "" && ticket.LoginID != opts.LoginID {
		return ErrTicketMismatch
	}
	if opts.Device != "" && ticket.Device != opts.Device {
		return ErrTicketMismatch
	}
	if opts.DeviceID != "" && ticket.DeviceID != opts.DeviceID {
		return ErrTicketMismatch
	}
	if opts.Source != "" && ticket.Source != opts.Source {
		return ErrTicketMismatch
	}
	if opts.SourceApp != "" && ticket.SourceApp != opts.SourceApp {
		return ErrTicketMismatch
	}
	if opts.TargetApp != "" && ticket.TargetApp != opts.TargetApp {
		return ErrTicketMismatch
	}
	return nil
}

// expirationTime reads the precise deadline with compatibility for legacy payloads. expirationTime 读取精确截止时间，并兼容旧版载荷。
func expirationTime(ticket *Ticket) time.Time {
	if ticket == nil {
		return time.Time{}
	}
	if !ticket.ExpiresAt.IsZero() {
		return ticket.ExpiresAt
	}
	if ticket.ExpiresIn > 0 {
		return time.Unix(ticket.CreateTime+ticket.ExpiresIn, 0)
	}
	return time.Time{}
}

// remainingDuration calculates the remaining ticket lifetime. remainingDuration 计算 Ticket 剩余有效期。
func remainingDuration(ticket *Ticket) time.Duration {
	ttl := time.Until(expirationTime(ticket))
	if ttl <= 0 {
		return 0
	}
	return ttl
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

// generateRandomValue creates a cryptographically secure hexadecimal value. generateRandomValue 生成密码学安全的十六进制随机值。
func generateRandomValue(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
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
