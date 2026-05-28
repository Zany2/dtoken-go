// @Author daixk 2025/12/22 15:56:00
package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
)

var (
	// ErrInvalidTicket indicates an invalid or missing SSO ticket. ErrInvalidTicket 表示 SSO Ticket 无效或不存在。
	ErrInvalidTicket = errors.New("invalid SSO ticket")
	// ErrTicketUsed indicates a consumed SSO ticket. ErrTicketUsed 表示 SSO Ticket 已被消费。
	ErrTicketUsed = errors.New("SSO ticket has been used")
	// ErrTicketExpired indicates an expired SSO ticket. ErrTicketExpired 表示 SSO Ticket 已过期。
	ErrTicketExpired = errors.New("SSO ticket has expired")
	// ErrModeUnsupported indicates an unsupported SSO mode. ErrModeUnsupported 表示 SSO 模式不受支持。
	ErrModeUnsupported = errors.New("SSO mode unsupported")
)

// Config defines SSO server config. Config 定义 SSO 服务端配置。
type Config struct {
	// TicketExpiration stores ticket ttl. TicketExpiration 存储 Ticket 有效期。
	TicketExpiration time.Duration
}

// DefaultConfig returns default SSO config. DefaultConfig 返回默认 SSO 配置。
func DefaultConfig() *Config {
	return &Config{TicketExpiration: DefaultTicketExpiration}
}

// Validate validates SSO config. Validate 校验 SSO 配置。
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	if c.TicketExpiration <= 0 {
		return fmt.Errorf("SSOConfig.TicketExpiration must be a positive duration")
	}
	return nil
}

// Clone returns a deep copy of SSO config. Clone 返回 SSO 配置副本。
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	copyCfg := *c
	return &copyCfg
}

// Client defines an SSO client application. Client 定义 SSO 子应用客户端。
type Client struct {
	ClientID     string         `json:"clientId"`               // ClientID stores application id. ClientID 存储应用 ID。
	ClientSecret string         `json:"clientSecret,omitempty"` // ClientSecret stores application secret. ClientSecret 存储应用密钥。
	Name         string         `json:"name,omitempty"`         // Name stores display name. Name 存储展示名称。
	RedirectURIs []string       `json:"redirectUris,omitempty"` // RedirectURIs stores allowed callback URIs. RedirectURIs 存储允许的回调地址。
	AllowOrigins []string       `json:"allowOrigins,omitempty"` // AllowOrigins stores allowed front-end origins. AllowOrigins 存储允许的前端来源。
	Modes        []Mode         `json:"modes,omitempty"`        // Modes stores allowed SSO modes. Modes 存储允许的 SSO 模式。
	Scopes       []string       `json:"scopes,omitempty"`       // Scopes stores allowed scopes. Scopes 存储允许的权限范围。
	Extra        map[string]any `json:"extra,omitempty"`        // Extra stores extension data. Extra 存储扩展数据。
}

// Ticket defines an SSO one-time ticket. Ticket 定义 SSO 一次性票据。
type Ticket struct {
	Ticket      string         `json:"ticket"`           // Ticket stores ticket value. Ticket 存储票据值。
	Mode        Mode           `json:"mode"`             // Mode stores SSO mode. Mode 存储 SSO 模式。
	LoginID     string         `json:"loginId"`          // LoginID stores logged-in subject id. LoginID 存储登录主体 ID。
	ClientID    string         `json:"clientId"`         // ClientID stores target client id. ClientID 存储目标客户端 ID。
	RedirectURI string         `json:"redirectUri"`      // RedirectURI stores callback URI. RedirectURI 存储回调地址。
	Scopes      []string       `json:"scopes,omitempty"` // Scopes stores requested scopes. Scopes 存储请求范围。
	CreateTime  int64          `json:"createTime"`       // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	ExpiresIn   int64          `json:"expiresIn"`        // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Used        bool           `json:"used"`             // Used stores consume state. Used 存储消费状态。
	Extra       map[string]any `json:"extra,omitempty"`  // Extra stores extension data. Extra 存储扩展数据。
}

// Server handles SSO client and ticket operations. Server 处理 SSO 客户端与 Ticket 操作。
type Server struct {
	authType         string          // authType stores auth namespace. authType 存储认证命名空间。
	keyPrefix        string          // keyPrefix stores storage prefix. keyPrefix 存储键前缀。
	ticketExpiration time.Duration   // ticketExpiration stores ticket ttl. ticketExpiration 存储 Ticket 有效期。
	storage          adapter.Storage // storage stores storage adapter. storage 存储适配器。
	serializer       adapter.Codec   // serializer stores codec adapter. serializer 存储编解码适配器。
}

// NewDefaultServer creates SSO server with default config. NewDefaultServer 使用默认配置创建 SSO 服务端。
func NewDefaultServer(authType, prefix string, storage adapter.Storage, serializer adapter.Codec) *Server {
	return NewServerWithConfig(authType, prefix, storage, serializer, DefaultConfig())
}

// NewServerWithConfig creates SSO server with config. NewServerWithConfig 使用配置创建 SSO 服务端。
func NewServerWithConfig(authType, prefix string, storage adapter.Storage, serializer adapter.Codec, cfg *Config) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	ticketExpiration := cfg.TicketExpiration
	if ticketExpiration <= 0 {
		ticketExpiration = DefaultTicketExpiration
	}
	return &Server{
		authType:         authType,
		keyPrefix:        prefix,
		ticketExpiration: ticketExpiration,
		storage:          storage,
		serializer:       serializer,
	}
}

// RegisterClient registers an SSO client. RegisterClient 注册 SSO 客户端。
func (s *Server) RegisterClient(client *Client) error {
	if client == nil || client.ClientID == "" {
		return derror.ErrClientOrClientIDEmpty
	}
	return s.saveClient(context.Background(), client)
}

// UnregisterClient unregisters an SSO client. UnregisterClient 注销 SSO 客户端。
func (s *Server) UnregisterClient(clientID string) error {
	if clientID == "" {
		return derror.ErrClientOrClientIDEmpty
	}
	return s.deleteClient(context.Background(), clientID)
}

// GetClient gets an SSO client by id. GetClient 根据 ID 获取 SSO 客户端。
func (s *Server) GetClient(clientID string) (*Client, error) {
	if clientID == "" {
		return nil, derror.ErrClientOrClientIDEmpty
	}
	return s.getClient(context.Background(), clientID)
}

// GenerateTicket generates an SSO ticket with default ttl. GenerateTicket 使用默认有效期生成 SSO Ticket。
func (s *Server) GenerateTicket(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any) (*Ticket, error) {
	return s.GenerateTicketWithTimeout(ctx, clientID, loginID, redirectURI, scopes, extra, s.ticketExpiration)
}

// GenerateTicketWithTimeout generates an SSO ticket with timeout. GenerateTicketWithTimeout 使用指定有效期生成 SSO Ticket。
func (s *Server) GenerateTicketWithTimeout(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any, timeout time.Duration) (*Ticket, error) {
	if loginID == "" {
		return nil, derror.ErrUserIDEmpty
	}
	if timeout <= 0 {
		timeout = s.ticketExpiration
	}

	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if !s.isModeAllowed(client, ModeTicket) {
		return nil, ErrModeUnsupported
	}
	if !s.isValidRedirectURI(client, redirectURI) {
		return nil, derror.ErrInvalidRedirectURI
	}
	if !s.isValidScopes(client, scopes) {
		return nil, derror.ErrInvalidScope
	}

	ticketValue, err := s.generateTicketValue()
	if err != nil {
		return nil, err
	}
	ticket := &Ticket{
		Ticket:      ticketValue,
		Mode:        ModeTicket,
		LoginID:     loginID,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		CreateTime:  time.Now().Unix(),
		ExpiresIn:   int64(timeout.Seconds()),
		Used:        false,
		Extra:       extra,
	}

	if err = s.saveTicket(ctx, ticket, timeout); err != nil {
		return nil, err
	}
	return ticket, nil
}

// ValidateTicket validates a ticket without consuming it. ValidateTicket 校验 Ticket 但不消费。
func (s *Server) ValidateTicket(ctx context.Context, ticketValue string) (*Ticket, error) {
	ticket, err := s.getTicket(ctx, ticketValue)
	if err != nil {
		return nil, err
	}
	if err = s.checkTicketAlive(ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

// ConsumeTicket validates and consumes a one-time ticket. ConsumeTicket 校验并消费一次性 Ticket。
func (s *Server) ConsumeTicket(ctx context.Context, ticketValue, clientID, clientSecret, redirectURI string) (*Ticket, error) {
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client.ClientSecret != "" && client.ClientSecret != clientSecret {
		return nil, derror.ErrInvalidClientCredentials
	}
	if !s.isModeAllowed(client, ModeTicket) {
		return nil, ErrModeUnsupported
	}

	key := s.getTicketKey(ticketValue)
	current, err := s.getTicket(ctx, ticketValue)
	if err != nil {
		return nil, err
	}
	if current.ClientID != clientID {
		return nil, derror.ErrClientMismatch
	}
	if current.RedirectURI != redirectURI {
		return nil, derror.ErrRedirectURIMismatch
	}
	if err = s.checkTicketAlive(current); err != nil {
		return nil, err
	}

	atomicStorage, ok := s.storage.(adapter.AtomicStorage)
	if !ok {
		return nil, derror.ErrStorageCapabilityUnsupported
	}

	value, err := atomicStorage.GetAndDelete(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if value == nil {
		return nil, ErrInvalidTicket
	}

	ticket, err := s.decodeTicket(value)
	if err != nil {
		return nil, err
	}
	if ticket.Used {
		return nil, ErrTicketUsed
	}
	if ticket.ClientID != clientID {
		return nil, derror.ErrClientMismatch
	}
	if ticket.RedirectURI != redirectURI {
		return nil, derror.ErrRedirectURIMismatch
	}
	if err = s.checkTicketAlive(ticket); err != nil {
		return nil, err
	}

	ticket.Used = true
	return ticket, nil
}

// RevokeTicket revokes an SSO ticket. RevokeTicket 撤销 SSO Ticket。
func (s *Server) RevokeTicket(ctx context.Context, ticketValue string) error {
	if ticketValue == "" {
		return nil
	}
	if err := s.storage.Delete(ctx, s.getTicketKey(ticketValue)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// GetTicketTTL returns ticket TTL in seconds. GetTicketTTL 获取 Ticket 剩余秒数。
func (s *Server) GetTicketTTL(ctx context.Context, ticketValue string) (int64, error) {
	if ticketValue == "" {
		return -2, nil
	}
	ttl, err := s.storage.TTL(ctx, s.getTicketKey(ticketValue))
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

func (s *Server) getClientKey(clientID string) string {
	return s.keyPrefix + s.authType + ClientKeySuffix + clientID
}

func (s *Server) getTicketKey(ticket string) string {
	return s.keyPrefix + s.authType + TicketKeySuffix + ticket
}

func (s *Server) saveClient(ctx context.Context, client *Client) error {
	encodeData, err := s.serializer.Encode(client)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getClientKey(client.ClientID), encodeData, 0); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

func (s *Server) deleteClient(ctx context.Context, clientID string) error {
	if err := s.storage.Delete(ctx, s.getClientKey(clientID)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

func (s *Server) getClient(ctx context.Context, clientID string) (*Client, error) {
	data, err := s.storage.Get(ctx, s.getClientKey(clientID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrClientNotFound
	}
	rawData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var client Client
	if err = s.serializer.Decode(rawData, &client); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	return &client, nil
}

func (s *Server) saveTicket(ctx context.Context, ticket *Ticket, timeout time.Duration) error {
	encodeData, err := s.serializer.Encode(ticket)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getTicketKey(ticket.Ticket), encodeData, timeout); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

func (s *Server) getTicket(ctx context.Context, ticketValue string) (*Ticket, error) {
	if ticketValue == "" {
		return nil, ErrInvalidTicket
	}
	data, err := s.storage.Get(ctx, s.getTicketKey(ticketValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrInvalidTicket
	}
	return s.decodeTicket(data)
}

func (s *Server) decodeTicket(value any) (*Ticket, error) {
	rawData, err := utils.ToBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var ticket Ticket
	if err = s.serializer.Decode(rawData, &ticket); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	return &ticket, nil
}

func (s *Server) checkTicketAlive(ticket *Ticket) error {
	if ticket == nil || ticket.Ticket == "" {
		return ErrInvalidTicket
	}
	if ticket.Used {
		return ErrTicketUsed
	}
	if ticket.ExpiresIn > 0 && time.Now().Unix() > ticket.CreateTime+ticket.ExpiresIn {
		return ErrTicketExpired
	}
	return nil
}

func (s *Server) generateTicketValue() (string, error) {
	bytes := make([]byte, TicketLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate SSO ticket: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (s *Server) isValidRedirectURI(client *Client, redirectURI string) bool {
	if client == nil || redirectURI == "" {
		return false
	}
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			return true
		}
	}
	return false
}

func (s *Server) isValidScopes(client *Client, scopes []string) bool {
	if client == nil || len(scopes) == 0 || len(client.Scopes) == 0 {
		return true
	}
	allowedScopes := make(map[string]struct{}, len(client.Scopes))
	for _, scope := range client.Scopes {
		allowedScopes[scope] = struct{}{}
	}
	for _, scope := range scopes {
		if _, ok := allowedScopes[scope]; !ok {
			return false
		}
	}
	return true
}

func (s *Server) isModeAllowed(client *Client, mode Mode) bool {
	if client == nil {
		return false
	}
	if len(client.Modes) == 0 {
		return mode == ModeTicket
	}
	for _, item := range client.Modes {
		if item == mode {
			return true
		}
	}
	return false
}
