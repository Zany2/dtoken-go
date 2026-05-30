// @Author daixk 2025/12/22 15:56:00
package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

var (
	// ErrStorageUnavailable indicates storage access failed. ErrStorageUnavailable 表示存储访问失败。
	ErrStorageUnavailable = errors.New("storage unavailable")
	// ErrSerializeFailed indicates codec encode or decode failed. ErrSerializeFailed 表示编解码失败。
	ErrSerializeFailed = errors.New("serialize failed")
	// ErrTypeConvert indicates stored payload cannot be converted to bytes. ErrTypeConvert 表示存储载荷无法转换为字节。
	ErrTypeConvert = errors.New("type conversion failed")
	// ErrStorageCapabilityUnsupported indicates required storage capability is missing. ErrStorageCapabilityUnsupported 表示存储缺少必要能力。
	ErrStorageCapabilityUnsupported = errors.New("storage capability unsupported")
	// ErrServerNotInitialized indicates the SSO server is not initialized. ErrServerNotInitialized 表示 SSO 服务端未初始化。
	ErrServerNotInitialized = errors.New("SSO server not initialized")
	// ErrClientOrClientIDEmpty indicates empty client or client id. ErrClientOrClientIDEmpty 表示客户端或客户端 ID 为空。
	ErrClientOrClientIDEmpty = errors.New("client or client ID cannot be empty")
	// ErrClientNotFound indicates client not found. ErrClientNotFound 表示客户端不存在。
	ErrClientNotFound = errors.New("client not found")
	// ErrInvalidClientCredentials indicates invalid client credentials. ErrInvalidClientCredentials 表示客户端凭证无效。
	ErrInvalidClientCredentials = errors.New("invalid client credentials")
	// ErrInvalidRedirectURI indicates invalid redirect uri. ErrInvalidRedirectURI 表示回调 URI 无效。
	ErrInvalidRedirectURI = errors.New("invalid redirect URI")
	// ErrRedirectURIMismatch indicates redirect uri mismatch. ErrRedirectURIMismatch 表示回调 URI 不匹配。
	ErrRedirectURIMismatch = errors.New("redirect URI mismatch")
	// ErrInvalidScope indicates invalid scope. ErrInvalidScope 表示权限范围无效。
	ErrInvalidScope = errors.New("invalid scope")
	// ErrUserIDEmpty indicates empty user id. ErrUserIDEmpty 表示用户 ID 为空。
	ErrUserIDEmpty = errors.New("user ID cannot be empty")
	// ErrClientMismatch indicates client mismatch. ErrClientMismatch 表示客户端不匹配。
	ErrClientMismatch = errors.New("client mismatch")
	// ErrInvalidTicket indicates an invalid, missing, or already removed SSO ticket. ErrInvalidTicket 表示 SSO Ticket 无效、不存在或已被移除。
	ErrInvalidTicket = errors.New("invalid SSO ticket")
	// ErrTicketUsed indicates a consumed SSO ticket payload. ErrTicketUsed 表示 SSO Ticket 载荷已被消费。
	ErrTicketUsed = errors.New("SSO ticket has been used")
	// ErrTicketExpired indicates an expired SSO ticket. ErrTicketExpired 表示 SSO Ticket 已过期。
	ErrTicketExpired = errors.New("SSO ticket has expired")
	// ErrModeUnsupported indicates that the client does not allow the requested SSO mode. ErrModeUnsupported 表示客户端不允许当前 SSO 模式。
	ErrModeUnsupported = errors.New("SSO mode unsupported")
	// ErrInvalidSharedToken indicates an invalid or missing SSO shared token. ErrInvalidSharedToken 表示 SSO 共享 Token 无效或不存在。
	ErrInvalidSharedToken = errors.New("invalid SSO shared token")
	// ErrSharedTokenExpired indicates an expired SSO shared token. ErrSharedTokenExpired 表示 SSO 共享 Token 已过期。
	ErrSharedTokenExpired = errors.New("SSO shared token has expired")
	// ErrInvalidRemoteSession indicates an invalid or missing SSO remote session. ErrInvalidRemoteSession 表示 SSO 远程会话无效或不存在。
	ErrInvalidRemoteSession = errors.New("invalid SSO remote session")
	// ErrRemoteSessionExpired indicates an expired SSO remote session. ErrRemoteSessionExpired 表示 SSO 远程会话已过期。
	ErrRemoteSessionExpired = errors.New("SSO remote session has expired")
	// ErrInvalidOAuth2Code indicates an invalid or missing SSO OAuth2 code. ErrInvalidOAuth2Code 表示 SSO OAuth2 授权码无效或不存在。
	ErrInvalidOAuth2Code = errors.New("invalid SSO OAuth2 code")
	// ErrOAuth2CodeUsed indicates a consumed SSO OAuth2 code. ErrOAuth2CodeUsed 表示 SSO OAuth2 授权码已被消费。
	ErrOAuth2CodeUsed = errors.New("SSO OAuth2 code has been used")
	// ErrOAuth2CodeExpired indicates an expired SSO OAuth2 code. ErrOAuth2CodeExpired 表示 SSO OAuth2 授权码已过期。
	ErrOAuth2CodeExpired = errors.New("SSO OAuth2 code has expired")
	// ErrClientSessionNotFound indicates there is no SSO client session record. ErrClientSessionNotFound 表示 SSO 客户端会话记录不存在。
	ErrClientSessionNotFound = errors.New("client session not found")
	// ErrInvalidSign indicates the request signature is invalid. ErrInvalidSign 表示请求签名无效。
	ErrInvalidSign = errors.New("invalid sign")
	// ErrMethodNotAllowed indicates the request method is not allowed. ErrMethodNotAllowed 表示请求方法不允许。
	ErrMethodNotAllowed = errors.New("method not allowed")
	// ErrInvalidCallbackURL indicates logout callback URL is not allowed. ErrInvalidCallbackURL 表示注销回调地址不被允许。
	ErrInvalidCallbackURL = errors.New("invalid callback URL")
	// ErrCallbackExpired indicates logout callback timestamp is outside allowed window. ErrCallbackExpired 表示注销回调时间戳超过允许窗口。
	ErrCallbackExpired = errors.New("logout callback expired")
)

// Config defines SSO server config. Config 定义 SSO 服务端配置。
type Config struct {
	// TicketExpiration stores the default lifetime for generated tickets. TicketExpiration 存储生成 Ticket 的默认有效期。
	TicketExpiration time.Duration
	// SharedTokenExpiration stores the default lifetime for shared tokens. SharedTokenExpiration 存储共享 Token 的默认有效期。
	SharedTokenExpiration time.Duration
	// RemoteSessionExpiration stores the default lifetime for remote sessions. RemoteSessionExpiration 存储远程会话的默认有效期。
	RemoteSessionExpiration time.Duration
	// OAuth2CodeExpiration stores the default lifetime for OAuth2 codes. OAuth2CodeExpiration 存储 OAuth2 授权码的默认有效期。
	OAuth2CodeExpiration time.Duration
}

// DefaultConfig returns default SSO config. DefaultConfig 返回默认 SSO 配置。
func DefaultConfig() *Config {
	return &Config{
		TicketExpiration:        DefaultTicketExpiration,
		SharedTokenExpiration:   DefaultSharedTokenExpiration,
		RemoteSessionExpiration: DefaultRemoteSessionExpiration,
		OAuth2CodeExpiration:    DefaultOAuth2CodeExpiration,
	}
}

// Validate validates SSO config. Validate 校验 SSO 配置。
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}
	if c.TicketExpiration <= 0 {
		return fmt.Errorf("SSOConfig.TicketExpiration must be a positive duration")
	}
	if c.SharedTokenExpiration <= 0 {
		return fmt.Errorf("SSOConfig.SharedTokenExpiration must be a positive duration")
	}
	if c.RemoteSessionExpiration <= 0 {
		return fmt.Errorf("SSOConfig.RemoteSessionExpiration must be a positive duration")
	}
	if c.OAuth2CodeExpiration <= 0 {
		return fmt.Errorf("SSOConfig.OAuth2CodeExpiration must be a positive duration")
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

// Client defines an SSO client application registered by the login center. Client 定义统一登录中心注册的 SSO 子应用客户端。
type Client struct {
	ClientID     string         `json:"clientId"`               // ClientID stores the unique application id. ClientID 存储唯一应用 ID。
	ClientSecret string         `json:"clientSecret,omitempty"` // ClientSecret stores the optional application secret. ClientSecret 存储可选的应用密钥。
	Name         string         `json:"name,omitempty"`         // Name stores the display name for operators. Name 存储面向管理者的展示名称。
	RedirectURIs []string       `json:"redirectUris,omitempty"` // RedirectURIs stores allowed callback URIs. RedirectURIs 存储允许的回调地址。
	AllowOrigins []string       `json:"allowOrigins,omitempty"` // AllowOrigins stores allowed front-end origins for future browser flows. AllowOrigins 存储预留给浏览器流程的前端来源。
	Modes        []Mode         `json:"modes,omitempty"`        // Modes stores allowed SSO modes; empty means ticket mode only. Modes 存储允许的 SSO 模式，空值表示仅允许 Ticket 模式。
	Scopes       []string       `json:"scopes,omitempty"`       // Scopes stores allowed scopes; empty means all requested scopes are accepted. Scopes 存储允许的权限范围，空值表示接受所有请求范围。
	Extra        map[string]any `json:"extra,omitempty"`        // Extra stores custom application metadata. Extra 存储自定义应用元数据。
}

// Ticket defines an SSO one-time ticket exchanged between the login center and a client app. Ticket 定义统一登录中心和子应用之间交换的一次性 SSO 票据。
type Ticket struct {
	Ticket      string         `json:"ticket"`           // Ticket stores the random ticket value. Ticket 存储随机票据值。
	Mode        Mode           `json:"mode"`             // Mode stores SSO mode. Mode 存储 SSO 模式。
	LoginID     string         `json:"loginId"`          // LoginID stores logged-in subject id at the SSO center. LoginID 存储统一登录中心的登录主体 ID。
	ClientID    string         `json:"clientId"`         // ClientID stores target client id. ClientID 存储目标客户端 ID。
	RedirectURI string         `json:"redirectUri"`      // RedirectURI stores callback URI. RedirectURI 存储回调地址。
	Scopes      []string       `json:"scopes,omitempty"` // Scopes stores requested scopes. Scopes 存储请求范围。
	CreateTime  int64          `json:"createTime"`       // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	ExpiresIn   int64          `json:"expiresIn"`        // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Used        bool           `json:"used"`             // Used stores consume state after successful exchange. Used 存储成功换票后的消费状态。
	Extra       map[string]any `json:"extra,omitempty"`  // Extra stores extension data. Extra 存储扩展数据。
}

// SharedToken defines a reusable SSO token shared by trusted apps. SharedToken 定义可信应用间共享复用的 SSO Token。
type SharedToken struct {
	Token      string         `json:"token"`            // Token stores the random shared token value. Token 存储随机共享 Token 值。
	Mode       Mode           `json:"mode"`             // Mode stores SSO mode. Mode 存储 SSO 模式。
	LoginID    string         `json:"loginId"`          // LoginID stores logged-in subject id at the SSO center. LoginID 存储统一登录中心的登录主体 ID。
	ClientID   string         `json:"clientId"`         // ClientID stores the client receiving the shared token. ClientID 存储接收共享 Token 的客户端 ID。
	Scopes     []string       `json:"scopes,omitempty"` // Scopes stores granted scopes. Scopes 存储授权范围。
	CreateTime int64          `json:"createTime"`       // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	ExpiresIn  int64          `json:"expiresIn"`        // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Extra      map[string]any `json:"extra,omitempty"`  // Extra stores extension data. Extra 存储扩展数据。
}

// RemoteSession defines a centralized SSO session checked by client apps remotely. RemoteSession 定义由子应用远程校验的中心化 SSO 会话。
type RemoteSession struct {
	SessionID  string         `json:"sessionId"`        // SessionID stores the random remote session id. SessionID 存储随机远程会话 ID。
	Mode       Mode           `json:"mode"`             // Mode stores SSO mode. Mode 存储 SSO 模式。
	LoginID    string         `json:"loginId"`          // LoginID stores logged-in subject id at the SSO center. LoginID 存储统一登录中心的登录主体 ID。
	ClientID   string         `json:"clientId"`         // ClientID stores the client that owns this remote view. ClientID 存储拥有该远程视图的客户端 ID。
	Scopes     []string       `json:"scopes,omitempty"` // Scopes stores granted scopes. Scopes 存储授权范围。
	CreateTime int64          `json:"createTime"`       // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	ExpiresIn  int64          `json:"expiresIn"`        // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Extra      map[string]any `json:"extra,omitempty"`  // Extra stores extension data. Extra 存储扩展数据。
}

// OAuth2Code defines an SSO OAuth2 authorization code. OAuth2Code 定义 SSO OAuth2 授权码。
type OAuth2Code struct {
	Code        string         `json:"code"`             // Code stores the random authorization code. Code 存储随机授权码。
	Mode        Mode           `json:"mode"`             // Mode stores SSO mode. Mode 存储 SSO 模式。
	LoginID     string         `json:"loginId"`          // LoginID stores logged-in subject id at the SSO center. LoginID 存储统一登录中心的登录主体 ID。
	ClientID    string         `json:"clientId"`         // ClientID stores target client id. ClientID 存储目标客户端 ID。
	RedirectURI string         `json:"redirectUri"`      // RedirectURI stores callback URI. RedirectURI 存储回调地址。
	Scopes      []string       `json:"scopes,omitempty"` // Scopes stores requested scopes. Scopes 存储请求范围。
	CreateTime  int64          `json:"createTime"`       // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	ExpiresIn   int64          `json:"expiresIn"`        // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Used        bool           `json:"used"`             // Used stores consume state after exchange. Used 存储换取后的消费状态。
	Extra       map[string]any `json:"extra,omitempty"`  // Extra stores extension data. Extra 存储扩展数据。
}

// ClientSession stores one client login binding for single logout. ClientSession 存储用于单点注销的客户端登录绑定。
type ClientSession struct {
	LoginID           string `json:"loginId"`                     // LoginID stores subject id at the login center. LoginID 存储认证中心登录主体 ID。
	ClientID          string `json:"clientId"`                    // ClientID stores client application id. ClientID 存储客户端应用 ID。
	LogoutCallbackURL string `json:"logoutCallbackUrl,omitempty"` // LogoutCallbackURL stores client logout callback URL. LogoutCallbackURL 存储客户端注销回调地址。
	CreateTime        int64  `json:"createTime"`                  // CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	UpdateTime        int64  `json:"updateTime"`                  // UpdateTime stores update unix time. UpdateTime 存储更新时间戳。
}

// Server handles SSO client registration and ticket operations. Server 处理 SSO 客户端注册与 Ticket 操作。
type Server struct {
	authType                string          // authType stores auth namespace for key isolation. authType 存储用于键隔离的认证命名空间。
	keyPrefix               string          // keyPrefix stores storage prefix shared by the manager. keyPrefix 存储 Manager 共享的存储键前缀。
	ticketExpiration        time.Duration   // ticketExpiration stores default ticket ttl. ticketExpiration 存储默认 Ticket 有效期。
	sharedTokenExpiration   time.Duration   // sharedTokenExpiration stores default shared token ttl. sharedTokenExpiration 存储默认共享 Token 有效期。
	remoteSessionExpiration time.Duration   // remoteSessionExpiration stores default remote session ttl. remoteSessionExpiration 存储默认远程会话有效期。
	oauth2CodeExpiration    time.Duration   // oauth2CodeExpiration stores default OAuth2 code ttl. oauth2CodeExpiration 存储默认 OAuth2 授权码有效期。
	storage                 adapter.Storage // storage stores clients and SSO credentials. storage 存储客户端和 SSO 凭证。
	serializer              adapter.Codec   // serializer encodes clients and credentials before storage. serializer 在存储前编解码客户端和凭证。
}

// NewDefaultServer creates SSO server with default config. NewDefaultServer 使用默认配置创建 SSO 服务端。
func NewDefaultServer(authType, prefix string, storage adapter.Storage, serializer adapter.Codec) *Server {
	return NewServerWithConfig(authType, prefix, storage, serializer, DefaultConfig())
}

// NewServerWithConfig creates SSO server with config and falls back to defaults for invalid ttl. NewServerWithConfig 使用配置创建 SSO 服务端，并在有效期无效时回退默认值。
func NewServerWithConfig(authType, prefix string, storage adapter.Storage, serializer adapter.Codec, cfg *Config) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	ticketExpiration := cfg.TicketExpiration
	if ticketExpiration <= 0 {
		ticketExpiration = DefaultTicketExpiration
	}
	sharedTokenExpiration := cfg.SharedTokenExpiration
	if sharedTokenExpiration <= 0 {
		sharedTokenExpiration = DefaultSharedTokenExpiration
	}
	remoteSessionExpiration := cfg.RemoteSessionExpiration
	if remoteSessionExpiration <= 0 {
		remoteSessionExpiration = DefaultRemoteSessionExpiration
	}
	oauth2CodeExpiration := cfg.OAuth2CodeExpiration
	if oauth2CodeExpiration <= 0 {
		oauth2CodeExpiration = DefaultOAuth2CodeExpiration
	}
	return &Server{
		authType:                authType,
		keyPrefix:               prefix,
		ticketExpiration:        ticketExpiration,
		sharedTokenExpiration:   sharedTokenExpiration,
		remoteSessionExpiration: remoteSessionExpiration,
		oauth2CodeExpiration:    oauth2CodeExpiration,
		storage:                 storage,
		serializer:              serializer,
	}
}

// RegisterClient registers an SSO client. RegisterClient 注册 SSO 客户端。
func (s *Server) RegisterClient(client *Client) error {
	if client == nil || client.ClientID == "" {
		return ErrClientOrClientIDEmpty
	}
	// Store client metadata before any ticket can be issued. 先保存客户端元数据，后续签发 Ticket 时依赖它做校验。
	return s.saveClient(context.Background(), client)
}

// UnregisterClient unregisters an SSO client. UnregisterClient 注销 SSO 客户端。
func (s *Server) UnregisterClient(clientID string) error {
	if clientID == "" {
		return ErrClientOrClientIDEmpty
	}
	return s.deleteClient(context.Background(), clientID)
}

// GetClient gets an SSO client by id. GetClient 根据 ID 获取 SSO 客户端。
func (s *Server) GetClient(clientID string) (*Client, error) {
	if clientID == "" {
		return nil, ErrClientOrClientIDEmpty
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
		return nil, ErrUserIDEmpty
	}
	if timeout <= 0 {
		timeout = s.ticketExpiration
	}

	// Resolve registered client first because redirect URI, scope, and mode are client-scoped policies. 先解析已注册客户端，因为回调地址、权限范围和模式都是客户端级策略。
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	// Enforce the per-client mode allow-list before issuing a ticket. 签发 Ticket 前先校验客户端允许的模式。
	if !s.isModeAllowed(client, ModeTicket) {
		return nil, ErrModeUnsupported
	}
	// Prevent tickets from being delivered to unregistered callback addresses. 防止 Ticket 被投递到未登记的回调地址。
	if !s.isValidRedirectURI(client, redirectURI) {
		return nil, ErrInvalidRedirectURI
	}
	// Keep requested scopes within the client allow-list when one is configured. 配置客户端权限范围时，限制请求范围不能越权。
	if !s.isValidScopes(client, scopes) {
		return nil, ErrInvalidScope
	}

	// Generate an opaque random ticket instead of exposing login data in the URL. 生成不透明随机 Ticket，避免在 URL 中暴露登录数据。
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

	// Persist the ticket with TTL so storage can expire unused callbacks automatically. 写入带 TTL 的 Ticket，让未使用回调自动过期。
	if err = s.saveTicket(ctx, ticket, timeout); err != nil {
		return nil, err
	}
	return ticket, nil
}

// ValidateTicket validates a ticket without consuming it. ValidateTicket 校验 Ticket 但不消费。
func (s *Server) ValidateTicket(ctx context.Context, ticketValue string) (*Ticket, error) {
	// Read-only validation is useful for inspection pages and diagnostics. 只读校验适用于状态检查和诊断场景。
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
	// Authenticate the target client before touching ticket state. 先认证目标客户端，避免无效客户端影响 Ticket 状态。
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client.ClientSecret != "" && client.ClientSecret != clientSecret {
		return nil, ErrInvalidClientCredentials
	}
	if !s.isModeAllowed(client, ModeTicket) {
		return nil, ErrModeUnsupported
	}

	key := s.getTicketKey(ticketValue)
	// Pre-check all deterministic constraints before atomic deletion so a wrong redirect URI does not consume a valid ticket. 原子删除前先校验确定性约束，避免错误回调地址消耗合法 Ticket。
	current, err := s.getTicket(ctx, ticketValue)
	if err != nil {
		return nil, err
	}
	if current.ClientID != clientID {
		return nil, ErrClientMismatch
	}
	if current.RedirectURI != redirectURI {
		return nil, ErrRedirectURIMismatch
	}
	if err = s.checkTicketAlive(current); err != nil {
		return nil, err
	}

	// Consumption requires atomic get-and-delete to keep the one-time guarantee under concurrency. 消费必须使用原子读删，保证并发下的一次性语义。
	atomicStorage, ok := s.storage.(adapter.AtomicStorage)
	if !ok {
		return nil, ErrStorageCapabilityUnsupported
	}

	// Another request may have consumed the ticket between the pre-check and this atomic operation. 预校验到原子操作之间可能已有其它请求消费了 Ticket。
	value, err := atomicStorage.GetAndDelete(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if value == nil {
		return nil, ErrInvalidTicket
	}

	ticket, err := s.decodeTicket(value)
	if err != nil {
		return nil, err
	}
	// Re-validate the deleted payload because storage is the final source of truth. 对原子删除得到的载荷再次校验，因为存储中的值才是最终事实。
	if ticket.Used {
		return nil, ErrTicketUsed
	}
	if ticket.ClientID != clientID {
		return nil, ErrClientMismatch
	}
	if ticket.RedirectURI != redirectURI {
		return nil, ErrRedirectURIMismatch
	}
	if err = s.checkTicketAlive(ticket); err != nil {
		return nil, err
	}

	// Mark the returned payload as consumed for callers; the stored copy has already been removed. 返回给调用方时标记已消费，存储中的副本已经被删除。
	ticket.Used = true
	return ticket, nil
}

// RevokeTicket revokes an SSO ticket. RevokeTicket 撤销 SSO Ticket。
func (s *Server) RevokeTicket(ctx context.Context, ticketValue string) error {
	if ticketValue == "" {
		return nil
	}
	// Revocation is idempotent: deleting a missing key is treated as success by storage adapters. 撤销保持幂等：删除不存在的键由存储适配器按成功处理。
	if err := s.storage.Delete(ctx, s.getTicketKey(ticketValue)); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
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
		return 0, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	// Normalize storage TTL sentinel values to Redis-like seconds. 将存储 TTL 哨兵值统一转换为类似 Redis 的秒级返回值。
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

// GenerateSharedToken generates an SSO shared token with default ttl. GenerateSharedToken 使用默认有效期生成 SSO 共享 Token。
func (s *Server) GenerateSharedToken(ctx context.Context, clientID, loginID string, scopes []string, extra map[string]any) (*SharedToken, error) {
	return s.GenerateSharedTokenWithTimeout(ctx, clientID, loginID, scopes, extra, s.sharedTokenExpiration)
}

// GenerateSharedTokenWithTimeout generates an SSO shared token with timeout. GenerateSharedTokenWithTimeout 使用指定有效期生成 SSO 共享 Token。
func (s *Server) GenerateSharedTokenWithTimeout(ctx context.Context, clientID, loginID string, scopes []string, extra map[string]any, timeout time.Duration) (*SharedToken, error) {
	if loginID == "" {
		return nil, ErrUserIDEmpty
	}
	if timeout <= 0 {
		timeout = s.sharedTokenExpiration
	}

	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if !s.isModeAllowed(client, ModeSharedToken) {
		return nil, ErrModeUnsupported
	}
	if !s.isValidScopes(client, scopes) {
		return nil, ErrInvalidScope
	}

	tokenValue, err := s.generateRandomValue(SharedTokenLength, "SSO shared token")
	if err != nil {
		return nil, err
	}
	token := &SharedToken{
		Token:      tokenValue,
		Mode:       ModeSharedToken,
		LoginID:    loginID,
		ClientID:   clientID,
		Scopes:     scopes,
		CreateTime: time.Now().Unix(),
		ExpiresIn:  int64(timeout.Seconds()),
		Extra:      extra,
	}
	if err = s.saveSharedToken(ctx, token, timeout); err != nil {
		return nil, err
	}
	return token, nil
}

// ValidateSharedToken validates an SSO shared token. ValidateSharedToken 校验 SSO 共享 Token。
func (s *Server) ValidateSharedToken(ctx context.Context, tokenValue, clientID string) (*SharedToken, error) {
	token, err := s.getSharedToken(ctx, tokenValue)
	if err != nil {
		return nil, err
	}
	if token.ClientID != clientID {
		return nil, ErrClientMismatch
	}
	if err = s.checkSharedTokenAlive(token); err != nil {
		return nil, err
	}
	return token, nil
}

// RevokeSharedToken revokes an SSO shared token. RevokeSharedToken 撤销 SSO 共享 Token。
func (s *Server) RevokeSharedToken(ctx context.Context, tokenValue string) error {
	if tokenValue == "" {
		return nil
	}
	if err := s.storage.Delete(ctx, s.getSharedTokenKey(tokenValue)); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// GetSharedTokenTTL returns shared token TTL in seconds. GetSharedTokenTTL 获取共享 Token 剩余秒数。
func (s *Server) GetSharedTokenTTL(ctx context.Context, tokenValue string) (int64, error) {
	return s.getTTLSeconds(ctx, s.getSharedTokenKey(tokenValue), tokenValue)
}

// CreateRemoteSession creates a centralized SSO session with default ttl. CreateRemoteSession 使用默认有效期创建中心化 SSO 会话。
func (s *Server) CreateRemoteSession(ctx context.Context, clientID, loginID string, scopes []string, extra map[string]any) (*RemoteSession, error) {
	return s.CreateRemoteSessionWithTimeout(ctx, clientID, loginID, scopes, extra, s.remoteSessionExpiration)
}

// CreateRemoteSessionWithTimeout creates a centralized SSO session with timeout. CreateRemoteSessionWithTimeout 使用指定有效期创建中心化 SSO 会话。
func (s *Server) CreateRemoteSessionWithTimeout(ctx context.Context, clientID, loginID string, scopes []string, extra map[string]any, timeout time.Duration) (*RemoteSession, error) {
	if loginID == "" {
		return nil, ErrUserIDEmpty
	}
	if timeout <= 0 {
		timeout = s.remoteSessionExpiration
	}

	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if !s.isModeAllowed(client, ModeRemoteSession) {
		return nil, ErrModeUnsupported
	}
	if !s.isValidScopes(client, scopes) {
		return nil, ErrInvalidScope
	}

	sessionID, err := s.generateRandomValue(RemoteSessionLength, "SSO remote session")
	if err != nil {
		return nil, err
	}
	session := &RemoteSession{
		SessionID:  sessionID,
		Mode:       ModeRemoteSession,
		LoginID:    loginID,
		ClientID:   clientID,
		Scopes:     scopes,
		CreateTime: time.Now().Unix(),
		ExpiresIn:  int64(timeout.Seconds()),
		Extra:      extra,
	}
	if err = s.saveRemoteSession(ctx, session, timeout); err != nil {
		return nil, err
	}
	return session, nil
}

// ValidateRemoteSession validates a centralized SSO session. ValidateRemoteSession 校验中心化 SSO 会话。
func (s *Server) ValidateRemoteSession(ctx context.Context, sessionID, clientID string) (*RemoteSession, error) {
	session, err := s.getRemoteSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.ClientID != clientID {
		return nil, ErrClientMismatch
	}
	if err = s.checkRemoteSessionAlive(session); err != nil {
		return nil, err
	}
	return session, nil
}

// RenewRemoteSession renews a centralized SSO session. RenewRemoteSession 续期中心化 SSO 会话。
func (s *Server) RenewRemoteSession(ctx context.Context, sessionID string, timeout time.Duration) error {
	if sessionID == "" {
		return ErrInvalidRemoteSession
	}
	if timeout <= 0 {
		timeout = s.remoteSessionExpiration
	}
	if err := s.storage.Expire(ctx, s.getRemoteSessionKey(sessionID), timeout); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// RevokeRemoteSession revokes a centralized SSO session. RevokeRemoteSession 撤销中心化 SSO 会话。
func (s *Server) RevokeRemoteSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	if err := s.storage.Delete(ctx, s.getRemoteSessionKey(sessionID)); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// GetRemoteSessionTTL returns remote session TTL in seconds. GetRemoteSessionTTL 获取远程会话剩余秒数。
func (s *Server) GetRemoteSessionTTL(ctx context.Context, sessionID string) (int64, error) {
	return s.getTTLSeconds(ctx, s.getRemoteSessionKey(sessionID), sessionID)
}

// GenerateOAuth2Code generates an SSO OAuth2 authorization code with default ttl. GenerateOAuth2Code 使用默认有效期生成 SSO OAuth2 授权码。
func (s *Server) GenerateOAuth2Code(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any) (*OAuth2Code, error) {
	return s.GenerateOAuth2CodeWithTimeout(ctx, clientID, loginID, redirectURI, scopes, extra, s.oauth2CodeExpiration)
}

// GenerateOAuth2CodeWithTimeout generates an SSO OAuth2 authorization code with timeout. GenerateOAuth2CodeWithTimeout 使用指定有效期生成 SSO OAuth2 授权码。
func (s *Server) GenerateOAuth2CodeWithTimeout(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any, timeout time.Duration) (*OAuth2Code, error) {
	if loginID == "" {
		return nil, ErrUserIDEmpty
	}
	if timeout <= 0 {
		timeout = s.oauth2CodeExpiration
	}

	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if !s.isModeAllowed(client, ModeOAuth2) {
		return nil, ErrModeUnsupported
	}
	if !s.isValidRedirectURI(client, redirectURI) {
		return nil, ErrInvalidRedirectURI
	}
	if !s.isValidScopes(client, scopes) {
		return nil, ErrInvalidScope
	}

	codeValue, err := s.generateRandomValue(OAuth2CodeLength, "SSO OAuth2 code")
	if err != nil {
		return nil, err
	}
	code := &OAuth2Code{
		Code:        codeValue,
		Mode:        ModeOAuth2,
		LoginID:     loginID,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		CreateTime:  time.Now().Unix(),
		ExpiresIn:   int64(timeout.Seconds()),
		Used:        false,
		Extra:       extra,
	}
	if err = s.saveOAuth2Code(ctx, code, timeout); err != nil {
		return nil, err
	}
	return code, nil
}

// ConsumeOAuth2Code validates and consumes an SSO OAuth2 authorization code. ConsumeOAuth2Code 校验并消费 SSO OAuth2 授权码。
func (s *Server) ConsumeOAuth2Code(ctx context.Context, codeValue, clientID, clientSecret, redirectURI string) (*OAuth2Code, error) {
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client.ClientSecret != "" && client.ClientSecret != clientSecret {
		return nil, ErrInvalidClientCredentials
	}
	if !s.isModeAllowed(client, ModeOAuth2) {
		return nil, ErrModeUnsupported
	}

	current, err := s.getOAuth2Code(ctx, codeValue)
	if err != nil {
		return nil, err
	}
	if current.ClientID != clientID {
		return nil, ErrClientMismatch
	}
	if current.RedirectURI != redirectURI {
		return nil, ErrRedirectURIMismatch
	}
	if err = s.checkOAuth2CodeAlive(current); err != nil {
		return nil, err
	}

	atomicStorage, ok := s.storage.(adapter.AtomicStorage)
	if !ok {
		return nil, ErrStorageCapabilityUnsupported
	}
	value, err := atomicStorage.GetAndDelete(ctx, s.getOAuth2CodeKey(codeValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if value == nil {
		return nil, ErrInvalidOAuth2Code
	}

	code, err := s.decodeOAuth2Code(value)
	if err != nil {
		return nil, err
	}
	if code.ClientID != clientID {
		return nil, ErrClientMismatch
	}
	if code.RedirectURI != redirectURI {
		return nil, ErrRedirectURIMismatch
	}
	if err = s.checkOAuth2CodeAlive(code); err != nil {
		return nil, err
	}
	code.Used = true
	return code, nil
}

// RevokeOAuth2Code revokes an SSO OAuth2 authorization code. RevokeOAuth2Code 撤销 SSO OAuth2 授权码。
func (s *Server) RevokeOAuth2Code(ctx context.Context, codeValue string) error {
	if codeValue == "" {
		return nil
	}
	if err := s.storage.Delete(ctx, s.getOAuth2CodeKey(codeValue)); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// GetOAuth2CodeTTL returns OAuth2 code TTL in seconds. GetOAuth2CodeTTL 获取 OAuth2 授权码剩余秒数。
func (s *Server) GetOAuth2CodeTTL(ctx context.Context, codeValue string) (int64, error) {
	return s.getTTLSeconds(ctx, s.getOAuth2CodeKey(codeValue), codeValue)
}

// RegisterClientSession records a client login binding for single logout. RegisterClientSession 记录用于单点注销的客户端登录绑定。
func (s *Server) RegisterClientSession(ctx context.Context, loginID, clientID, logoutCallbackURL string) (*ClientSession, error) {
	if loginID == "" {
		return nil, ErrUserIDEmpty
	}
	if clientID == "" {
		return nil, ErrClientOrClientIDEmpty
	}
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if !s.isValidLogoutCallbackURL(client, logoutCallbackURL) {
		return nil, ErrInvalidCallbackURL
	}
	now := time.Now().Unix()
	session := &ClientSession{
		LoginID:           loginID,
		ClientID:          clientID,
		LogoutCallbackURL: logoutCallbackURL,
		CreateTime:        now,
		UpdateTime:        now,
	}
	if existing, err := s.getClientSession(ctx, loginID, clientID); err == nil && existing != nil {
		session.CreateTime = existing.CreateTime
	}
	if err := s.saveClientSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// GetClientSessions returns recorded client sessions for login id. GetClientSessions 返回指定登录主体的客户端会话记录。
func (s *Server) GetClientSessions(ctx context.Context, loginID string) ([]ClientSession, error) {
	if loginID == "" {
		return nil, ErrUserIDEmpty
	}
	data, err := s.storage.Get(ctx, s.getClientSessionIndexKey(loginID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, nil
	}
	rawData, err := toBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTypeConvert, err)
	}
	var sessions []ClientSession
	if err = s.serializer.Decode(rawData, &sessions); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	return sessions, nil
}

// ClearClientSessions removes recorded client sessions for login id. ClearClientSessions 删除指定登录主体的客户端会话记录。
func (s *Server) ClearClientSessions(ctx context.Context, loginID string) error {
	if loginID == "" {
		return ErrUserIDEmpty
	}
	return s.deleteClientSessionIndex(ctx, loginID)
}

// getClientKey builds the storage key for an SSO client. getClientKey 构建 SSO 客户端的存储键。
func (s *Server) getClientKey(clientID string) string {
	return s.keyPrefix + s.authType + ClientKeySuffix + clientID
}

// getTicketKey builds the storage key for an SSO ticket. getTicketKey 构建 SSO Ticket 的存储键。
func (s *Server) getTicketKey(ticket string) string {
	return s.keyPrefix + s.authType + TicketKeySuffix + ticket
}

// getSharedTokenKey builds the storage key for an SSO shared token. getSharedTokenKey 构建 SSO 共享 Token 的存储键。
func (s *Server) getSharedTokenKey(token string) string {
	return s.keyPrefix + s.authType + SharedTokenKeySuffix + token
}

// getRemoteSessionKey builds the storage key for an SSO remote session. getRemoteSessionKey 构建 SSO 远程会话的存储键。
func (s *Server) getRemoteSessionKey(sessionID string) string {
	return s.keyPrefix + s.authType + RemoteSessionKeySuffix + sessionID
}

// getOAuth2CodeKey builds the storage key for an SSO OAuth2 code. getOAuth2CodeKey 构建 SSO OAuth2 授权码的存储键。
func (s *Server) getOAuth2CodeKey(code string) string {
	return s.keyPrefix + s.authType + OAuth2CodeKeySuffix + code
}

// getClientSessionIndexKey builds the storage key for SSO client sessions. getClientSessionIndexKey 构建 SSO 客户端会话索引键。
func (s *Server) getClientSessionIndexKey(loginID string) string {
	return s.keyPrefix + s.authType + ClientSessionKeySuffix + loginID
}

// saveClient serializes and persists a client without expiration. saveClient 序列化并永久保存客户端配置。
func (s *Server) saveClient(ctx context.Context, client *Client) error {
	encodeData, err := s.serializer.Encode(client)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getClientKey(client.ClientID), encodeData, 0); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// deleteClient removes a registered client. deleteClient 删除已注册客户端。
func (s *Server) deleteClient(ctx context.Context, clientID string) error {
	if err := s.storage.Delete(ctx, s.getClientKey(clientID)); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// getClient loads and decodes a registered client from storage. getClient 从存储加载并解码已注册客户端。
func (s *Server) getClient(ctx context.Context, clientID string) (*Client, error) {
	data, err := s.storage.Get(ctx, s.getClientKey(clientID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrClientNotFound
	}
	rawData, err := toBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTypeConvert, err)
	}
	var client Client
	if err = s.serializer.Decode(rawData, &client); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	return &client, nil
}

// saveTicket serializes and persists a ticket with the supplied ttl. saveTicket 按指定有效期序列化并保存 Ticket。
func (s *Server) saveTicket(ctx context.Context, ticket *Ticket, timeout time.Duration) error {
	encodeData, err := s.serializer.Encode(ticket)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getTicketKey(ticket.Ticket), encodeData, timeout); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// saveSharedToken serializes and persists a shared token with the supplied ttl. saveSharedToken 按指定有效期序列化并保存共享 Token。
func (s *Server) saveSharedToken(ctx context.Context, token *SharedToken, timeout time.Duration) error {
	encodeData, err := s.serializer.Encode(token)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getSharedTokenKey(token.Token), encodeData, timeout); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// saveRemoteSession serializes and persists a remote session with the supplied ttl. saveRemoteSession 按指定有效期序列化并保存远程会话。
func (s *Server) saveRemoteSession(ctx context.Context, session *RemoteSession, timeout time.Duration) error {
	encodeData, err := s.serializer.Encode(session)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getRemoteSessionKey(session.SessionID), encodeData, timeout); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// saveOAuth2Code serializes and persists an OAuth2 code with the supplied ttl. saveOAuth2Code 按指定有效期序列化并保存 OAuth2 授权码。
func (s *Server) saveOAuth2Code(ctx context.Context, code *OAuth2Code, timeout time.Duration) error {
	encodeData, err := s.serializer.Encode(code)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getOAuth2CodeKey(code.Code), encodeData, timeout); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// saveClientSession serializes and stores a client session in the login index. saveClientSession 将客户端会话序列化并保存到登录索引。
func (s *Server) saveClientSession(ctx context.Context, session *ClientSession) error {
	sessions, err := s.GetClientSessions(ctx, session.LoginID)
	if err != nil {
		return err
	}
	replaced := false
	for i := range sessions {
		if sessions[i].ClientID == session.ClientID {
			sessions[i] = *session
			replaced = true
			break
		}
	}
	if !replaced {
		sessions = append(sessions, *session)
	}
	encodeData, err := s.serializer.Encode(sessions)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	if err = s.storage.Set(ctx, s.getClientSessionIndexKey(session.LoginID), encodeData, 0); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// getClientSession loads one client session from the login index. getClientSession 从登录索引加载单个客户端会话。
func (s *Server) getClientSession(ctx context.Context, loginID, clientID string) (*ClientSession, error) {
	sessions, err := s.GetClientSessions(ctx, loginID)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		if sessions[i].ClientID == clientID {
			return &sessions[i], nil
		}
	}
	return nil, ErrClientSessionNotFound
}

// deleteClientSessionIndex removes all client session records for a login id. deleteClientSessionIndex 删除指定登录主体的全部客户端会话记录。
func (s *Server) deleteClientSessionIndex(ctx context.Context, loginID string) error {
	if err := s.storage.Delete(ctx, s.getClientSessionIndexKey(loginID)); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// getTicket loads and decodes a ticket by value. getTicket 根据票据值加载并解码 Ticket。
func (s *Server) getTicket(ctx context.Context, ticketValue string) (*Ticket, error) {
	if ticketValue == "" {
		return nil, ErrInvalidTicket
	}
	data, err := s.storage.Get(ctx, s.getTicketKey(ticketValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrInvalidTicket
	}
	return s.decodeTicket(data)
}

// getSharedToken loads and decodes a shared token by value. getSharedToken 根据 Token 值加载并解码共享 Token。
func (s *Server) getSharedToken(ctx context.Context, tokenValue string) (*SharedToken, error) {
	if tokenValue == "" {
		return nil, ErrInvalidSharedToken
	}
	data, err := s.storage.Get(ctx, s.getSharedTokenKey(tokenValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrInvalidSharedToken
	}
	return s.decodeSharedToken(data)
}

// getRemoteSession loads and decodes a remote session by id. getRemoteSession 根据会话 ID 加载并解码远程会话。
func (s *Server) getRemoteSession(ctx context.Context, sessionID string) (*RemoteSession, error) {
	if sessionID == "" {
		return nil, ErrInvalidRemoteSession
	}
	data, err := s.storage.Get(ctx, s.getRemoteSessionKey(sessionID))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrInvalidRemoteSession
	}
	return s.decodeRemoteSession(data)
}

// getOAuth2Code loads and decodes an OAuth2 code by value. getOAuth2Code 根据授权码值加载并解码 OAuth2 授权码。
func (s *Server) getOAuth2Code(ctx context.Context, codeValue string) (*OAuth2Code, error) {
	if codeValue == "" {
		return nil, ErrInvalidOAuth2Code
	}
	data, err := s.storage.Get(ctx, s.getOAuth2CodeKey(codeValue))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, ErrInvalidOAuth2Code
	}
	return s.decodeOAuth2Code(data)
}

// decodeTicket converts a stored payload into a Ticket object. decodeTicket 将存储载荷转换为 Ticket 对象。
func (s *Server) decodeTicket(value any) (*Ticket, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTypeConvert, err)
	}
	var ticket Ticket
	if err = s.serializer.Decode(rawData, &ticket); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	return &ticket, nil
}

// decodeSharedToken converts a stored payload into a SharedToken object. decodeSharedToken 将存储载荷转换为 SharedToken 对象。
func (s *Server) decodeSharedToken(value any) (*SharedToken, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTypeConvert, err)
	}
	var token SharedToken
	if err = s.serializer.Decode(rawData, &token); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	return &token, nil
}

// decodeRemoteSession converts a stored payload into a RemoteSession object. decodeRemoteSession 将存储载荷转换为 RemoteSession 对象。
func (s *Server) decodeRemoteSession(value any) (*RemoteSession, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTypeConvert, err)
	}
	var session RemoteSession
	if err = s.serializer.Decode(rawData, &session); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	return &session, nil
}

// decodeOAuth2Code converts a stored payload into an OAuth2Code object. decodeOAuth2Code 将存储载荷转换为 OAuth2Code 对象。
func (s *Server) decodeOAuth2Code(value any) (*OAuth2Code, error) {
	rawData, err := toBytes(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTypeConvert, err)
	}
	var code OAuth2Code
	if err = s.serializer.Decode(rawData, &code); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSerializeFailed, err)
	}
	return &code, nil
}

// toBytes converts stored payloads to bytes. toBytes 将存储载荷转换为字节切片。
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

// checkTicketAlive verifies basic ticket state independent of client policy. checkTicketAlive 校验与客户端策略无关的 Ticket 基础状态。
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

// checkSharedTokenAlive verifies basic shared token state. checkSharedTokenAlive 校验共享 Token 基础状态。
func (s *Server) checkSharedTokenAlive(token *SharedToken) error {
	if token == nil || token.Token == "" {
		return ErrInvalidSharedToken
	}
	if token.ExpiresIn > 0 && time.Now().Unix() > token.CreateTime+token.ExpiresIn {
		return ErrSharedTokenExpired
	}
	return nil
}

// checkRemoteSessionAlive verifies basic remote session state. checkRemoteSessionAlive 校验远程会话基础状态。
func (s *Server) checkRemoteSessionAlive(session *RemoteSession) error {
	if session == nil || session.SessionID == "" {
		return ErrInvalidRemoteSession
	}
	if session.ExpiresIn > 0 && time.Now().Unix() > session.CreateTime+session.ExpiresIn {
		return ErrRemoteSessionExpired
	}
	return nil
}

// checkOAuth2CodeAlive verifies basic OAuth2 code state. checkOAuth2CodeAlive 校验 OAuth2 授权码基础状态。
func (s *Server) checkOAuth2CodeAlive(code *OAuth2Code) error {
	if code == nil || code.Code == "" {
		return ErrInvalidOAuth2Code
	}
	if code.Used {
		return ErrOAuth2CodeUsed
	}
	if code.ExpiresIn > 0 && time.Now().Unix() > code.CreateTime+code.ExpiresIn {
		return ErrOAuth2CodeExpired
	}
	return nil
}

// generateTicketValue creates an opaque random ticket string. generateTicketValue 创建不透明随机 Ticket 字符串。
func (s *Server) generateTicketValue() (string, error) {
	return s.generateRandomValue(TicketLength, "SSO ticket")
}

// generateRandomValue creates an opaque random hex string. generateRandomValue 创建不透明随机十六进制字符串。
func (s *Server) generateRandomValue(length int, label string) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate %s: %w", label, err)
	}
	return hex.EncodeToString(bytes), nil
}

// getTTLSeconds normalizes storage TTL sentinel values to seconds. getTTLSeconds 将存储 TTL 哨兵值统一转换为秒。
func (s *Server) getTTLSeconds(ctx context.Context, key, value string) (int64, error) {
	if value == "" {
		return -2, nil
	}
	ttl, err := s.storage.TTL(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
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

// isValidRedirectURI checks whether the callback URI is registered for the client. isValidRedirectURI 检查回调地址是否已在客户端中登记。
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

// isValidLogoutCallbackURL checks whether logout callback URL belongs to the registered client. isValidLogoutCallbackURL 检查注销回调地址是否属于已注册客户端。
func (s *Server) isValidLogoutCallbackURL(client *Client, callbackURL string) bool {
	if client == nil || callbackURL == "" {
		return false
	}
	if s.isValidRedirectURI(client, callbackURL) {
		return true
	}
	callback, err := url.Parse(callbackURL)
	if err != nil || callback.Scheme == "" || callback.Host == "" {
		return false
	}
	for _, origin := range client.AllowOrigins {
		if originMatchesURL(origin, callback) {
			return true
		}
	}
	for _, redirectURI := range client.RedirectURIs {
		redirect, err := url.Parse(redirectURI)
		if err == nil && sameURLOrigin(redirect, callback) {
			return true
		}
	}
	return false
}

func originMatchesURL(origin string, target *url.URL) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return sameURLOrigin(parsed, target)
}

func sameURLOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Scheme != "" && b.Scheme != "" && a.Host != "" && b.Host != "" && a.Scheme == b.Scheme && a.Host == b.Host
}

// isValidScopes checks requested scopes against the client's allow-list. isValidScopes 根据客户端允许范围校验请求 scopes。
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

// isModeAllowed checks whether the client can use the requested SSO mode. isModeAllowed 检查客户端是否允许使用指定 SSO 模式。
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
