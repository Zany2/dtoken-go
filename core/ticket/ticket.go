// @Author daixk 2026/06/01
package ticket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
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

// Ticket stores a temporary credential payload. Ticket 存储临时凭证载荷。
type Ticket struct {
	Ticket     string         `json:"ticket"`              // Ticket stores the random credential value. Ticket 存储随机凭证值。
	AuthType   string         `json:"authType,omitempty"`  // AuthType stores auth namespace. AuthType 存储认证命名空间。
	LoginID    string         `json:"loginId,omitempty"`   // LoginID stores subject id. LoginID 存储登录主体 ID。
	Device     string         `json:"device,omitempty"`    // Device stores device type. Device 存储设备类型。
	DeviceId   string         `json:"deviceId,omitempty"`  // DeviceId stores concrete device id. DeviceId 存储具体设备 ID。
	Source     string         `json:"source,omitempty"`    // Source stores business source. Source 存储业务来源。
	SourceApp  string         `json:"sourceApp,omitempty"` // SourceApp stores issuing application. SourceApp 存储签发应用。
	TargetApp  string         `json:"targetApp,omitempty"` // TargetApp stores consuming application. TargetApp 存储目标应用。
	Scopes     []string       `json:"scopes,omitempty"`    // Scopes stores granted scopes. Scopes 存储授权范围。
	Extra      map[string]any `json:"extra,omitempty"`     // Extra stores extension data. Extra 存储扩展数据。
	CreateTime int64          `json:"createTime"`          // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	ExpiresIn  int64          `json:"expiresIn"`           // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Status     Status         `json:"status"`              // Status stores lifecycle state. Status 存储生命周期状态。
}

// CreateOptions defines ticket creation options. CreateOptions 定义 Ticket 创建选项。
type CreateOptions struct {
	LoginID   string
	Device    string
	DeviceId  string
	Source    string
	SourceApp string
	TargetApp string
	Scopes    []string
	Extra     map[string]any
	Timeout   time.Duration
}

// ValidateOptions defines ticket validation constraints. ValidateOptions 定义 Ticket 校验约束。
type ValidateOptions struct {
	LoginID   string
	Device    string
	DeviceId  string
	Source    string
	SourceApp string
	TargetApp string
}

// ConsumeResult stores consumed ticket data. ConsumeResult 存储 Ticket 消费结果。
type ConsumeResult struct {
	Ticket *Ticket
}

// Manager handles temporary ticket operations. Manager 处理临时 Ticket 操作。
type Manager struct {
	authType   string
	keyPrefix  string
	ttl        time.Duration
	storage    adapter.Storage
	serializer adapter.Codec
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

// Create creates a ticket with default ttl. Create 使用默认有效期创建 Ticket。
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Ticket, error) {
	return m.CreateWithTimeout(ctx, opts, opts.Timeout)
}

// CreateWithTimeout creates a ticket with timeout. CreateWithTimeout 使用指定有效期创建 Ticket。
func (m *Manager) CreateWithTimeout(ctx context.Context, opts CreateOptions, timeout time.Duration) (*Ticket, error) {
	if timeout <= 0 {
		timeout = m.ttl
	}
	value, err := generateRandomValue(TicketLength)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	ticket := &Ticket{
		Ticket:     value,
		AuthType:   m.authType,
		LoginID:    opts.LoginID,
		Device:     opts.Device,
		DeviceId:   opts.DeviceId,
		Source:     opts.Source,
		SourceApp:  opts.SourceApp,
		TargetApp:  opts.TargetApp,
		Scopes:     append([]string(nil), opts.Scopes...),
		Extra:      cloneMap(opts.Extra),
		CreateTime: now,
		ExpiresIn:  int64(timeout.Seconds()),
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
	if err = m.checkAlive(ticket); err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		if err = checkConstraints(ticket, opts[0]); err != nil {
			return nil, err
		}
	}
	return ticket, nil
}

// Consume validates and consumes a one-time ticket. Consume 校验并消费一次性 Ticket。
func (m *Manager) Consume(ctx context.Context, ticketValue string, opts ...ValidateOptions) (*ConsumeResult, error) {
	if _, err := m.Validate(ctx, ticketValue, opts...); err != nil {
		return nil, err
	}
	atomicStorage, ok := m.storage.(adapter.AtomicStorage)
	if !ok {
		return nil, derror.ErrStorageCapabilityUnsupported
	}
	value, err := atomicStorage.GetAndDelete(ctx, m.getTicketKey(ticketValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if value == nil {
		return nil, ErrInvalidTicket
	}
	ticket, err := m.decode(value)
	if err != nil {
		return nil, err
	}
	if err = m.checkAlive(ticket); err != nil {
		if ttl := remainingDuration(ticket); ttl > 0 {
			_ = m.save(ctx, ticket, ttl)
		}
		return nil, err
	}
	if len(opts) > 0 {
		if err = checkConstraints(ticket, opts[0]); err != nil {
			return nil, err
		}
	}
	ticket.Status = StatusConsumed
	if ttl := remainingDuration(ticket); ttl > 0 {
		_ = m.save(ctx, ticket, ttl)
	}
	return &ConsumeResult{Ticket: ticket}, nil
}

// Revoke revokes a ticket. Revoke 撤销 Ticket。
func (m *Manager) Revoke(ctx context.Context, ticketValue string) error {
	if ticketValue == "" {
		return nil
	}
	ticket, err := m.get(ctx, ticketValue)
	if err != nil {
		if errors.Is(err, ErrInvalidTicket) {
			return nil
		}
		return err
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
	if err = m.checkAlive(ticket); err != nil {
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

func (m *Manager) save(ctx context.Context, ticket *Ticket, timeout time.Duration) error {
	encoded, err := m.serializer.Encode(ticket)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if err = m.storage.Set(ctx, m.getTicketKey(ticket.Ticket), encoded, timeout); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

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
	return m.decode(data)
}

func (m *Manager) decode(value any) (*Ticket, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var ticket Ticket
	if err = m.serializer.Decode(rawData, &ticket); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	return &ticket, nil
}

func (m *Manager) checkAlive(ticket *Ticket) error {
	if ticket == nil || ticket.Ticket == "" {
		return ErrInvalidTicket
	}
	switch ticket.Status {
	case "", StatusValid:
	case StatusConsumed:
		return ErrTicketConsumed
	case StatusRevoked:
		return ErrTicketRevoked
	case StatusExpired:
		return ErrTicketExpired
	default:
		return ErrInvalidTicket
	}
	if ticket.ExpiresIn > 0 && time.Now().Unix() > ticket.CreateTime+ticket.ExpiresIn {
		return ErrTicketExpired
	}
	return nil
}

func (m *Manager) getTicketKey(ticketValue string) string {
	return m.keyPrefix + m.authType + TicketKeySuffix + ticketValue
}

func checkConstraints(ticket *Ticket, opts ValidateOptions) error {
	if opts.LoginID != "" && ticket.LoginID != opts.LoginID {
		return ErrTicketMismatch
	}
	if opts.Device != "" && ticket.Device != opts.Device {
		return ErrTicketMismatch
	}
	if opts.DeviceId != "" && ticket.DeviceId != opts.DeviceId {
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

func remainingDuration(ticket *Ticket) time.Duration {
	if ticket == nil || ticket.ExpiresIn <= 0 {
		return 0
	}
	expiresAt := time.Unix(ticket.CreateTime+ticket.ExpiresIn, 0)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return 0
	}
	return ttl
}

func generateRandomValue(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
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
