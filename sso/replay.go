package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

const (
	// DefaultReplayMaxAge bounds accepted request clock skew. DefaultReplayMaxAge 限制请求时间戳与当前时间的差值。
	DefaultReplayMaxAge = 5 * time.Minute
	// ActionAuthorize binds signatures to credential issuance. ActionAuthorize 将签名绑定到凭证签发。
	ActionAuthorize = "sso.authorize"
	// ActionToken binds signatures to credential exchange. ActionToken 将签名绑定到凭证兑换。
	ActionToken = "sso.token"
	// ActionIntrospect binds signatures to credential inspection. ActionIntrospect 将签名绑定到凭证检查。
	ActionIntrospect = "sso.introspect"
	// ActionUserInfo binds signatures to identity lookup. ActionUserInfo 将签名绑定到身份查询。
	ActionUserInfo = "sso.userinfo"
	// ActionRevoke binds signatures to credential revocation. ActionRevoke 将签名绑定到凭证撤销。
	ActionRevoke = "sso.revoke"
	// ActionLogout binds signatures to center logout. ActionLogout 将签名绑定到中心注销。
	ActionLogout = "sso.logout"
	// ActionLogoutCallback binds signatures to client logout callbacks. ActionLogoutCallback 将签名绑定到客户端注销回调。
	ActionLogoutCallback = "sso.logout_callback"
)

var (
	// ErrReplayProtectionConfig reports invalid replay protection wiring. ErrReplayProtectionConfig 表示防重放配置无效。
	ErrReplayProtectionConfig = errors.New("invalid sso replay protection configuration")
	// ErrRequestExpired reports an invalid or out-of-window timestamp. ErrRequestExpired 表示时间戳无效或超出允许窗口。
	ErrRequestExpired = errors.New("sso request timestamp is outside the accepted window")
	// ErrRequestReplayed reports a previously accepted nonce. ErrRequestReplayed 表示请求随机值已经使用。
	ErrRequestReplayed = errors.New("sso request has already been accepted")
)

// ReplayProtectionOptions configures borrowed storage and the accepted clock window. ReplayProtectionOptions 配置借用的存储与允许的时间窗口。
type ReplayProtectionOptions struct {
	// Storage retains accepted nonces; its lifecycle belongs to the caller. Storage 保存已使用随机值，生命周期由调用方管理。
	Storage contract.Storage
	// KeyPrefix isolates a trust domain; replicas must use the same prefix and storage. KeyPrefix 隔离信任域，同一服务的副本必须使用相同前缀和存储。
	KeyPrefix string
	// MaxAge allows past and future skew; zero defaults to five minutes. MaxAge 允许过去与未来的时间偏差，零值默认五分钟。
	MaxAge time.Duration
}

// ReplayProtection claims authenticated nonces using optional atomic storage or an instance-local lock. ReplayProtection 使用可选原子存储或实例内锁占用已认证的随机值。
type ReplayProtection struct {
	storage contract.Storage
	prefix  string
	maxAge  time.Duration
	mu      sync.Mutex
}

// NewReplayProtection creates opt-in protection without taking ownership of storage. NewReplayProtection 创建显式启用的防重放组件，不接管存储生命周期。
func NewReplayProtection(options ReplayProtectionOptions) (*ReplayProtection, error) {
	if options.MaxAge == 0 {
		options.MaxAge = DefaultReplayMaxAge
	}
	if options.Storage == nil || options.MaxAge < 0 || options.MaxAge > time.Duration(1<<63-1)/2 {
		return nil, ErrReplayProtectionConfig
	}
	if options.KeyPrefix == "" {
		options.KeyPrefix = "dtoken:sso:replay:"
	}
	return &ReplayProtection{storage: options.Storage, prefix: options.KeyPrefix, maxAge: options.MaxAge}, nil
}

// validateConfig rejects unsigned protection and ambiguous protocol parameter mappings. validateConfig 拒绝未签名的保护配置与歧义协议参数映射。
func (p *ReplayProtection) validateConfig(checkSign bool, names ParamNames) error {
	if p == nil {
		return nil
	}
	if !checkSign || p.storage == nil || p.maxAge <= 0 {
		return ErrReplayProtectionConfig
	}
	if !distinctParamNames(names) {
		return ErrReplayProtectionConfig
	}
	return nil
}

// stamp replaces reserved metadata for each new signing attempt. stamp 为每次新的签名尝试替换保留元数据。
func (p *ReplayProtection) stamp(values url.Values, names ParamNames, action string) url.Values {
	values = cloneValues(values)
	values.Set(names.Action, action)
	values.Set(names.Timestamp, time.Now().UTC().Format(time.RFC3339Nano))
	values.Set(names.Nonce, rand.Text())
	return values
}

// validate checks signed metadata without consuming the browser login round trip. validate 校验已签名的元数据，不消耗浏览器登录往返请求。
func (p *ReplayProtection) validate(values url.Values, names ParamNames, action string, now time.Time) error {
	for _, name := range []string{names.Client, names.Action, names.Timestamp, names.Nonce} {
		if len(values[name]) != 1 || values.Get(name) == "" {
			return ErrInvalidRequest
		}
	}
	if values.Get(names.Action) != action {
		return ErrInvalidRequest
	}
	nonce := values.Get(names.Nonce)
	if len(nonce) < 16 || len(nonce) > 128 {
		return ErrInvalidRequest
	}
	for _, ch := range nonce {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return ErrInvalidRequest
		}
	}
	timestamp, err := time.Parse(time.RFC3339Nano, values.Get(names.Timestamp))
	if err != nil || !timestamp.After(now.Add(-p.maxAge)) || timestamp.After(now.Add(p.maxAge)) {
		return ErrRequestExpired
	}
	return nil
}

// claim records a nonce only after signature verification and before business effects. claim 仅在验签后、业务操作前记录随机值。
func (p *ReplayProtection) claim(ctx context.Context, values url.Values, names ParamNames, action string) error {
	if p == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	atomicStorage, atomic := p.storage.(contract.AtomicStorage)
	if !atomic {
		p.mu.Lock()
		defer p.mu.Unlock()
	}
	if err := p.validate(values, names, action, time.Now()); err != nil {
		return err
	}

	// Encode tuple boundaries before hashing; future timestamps need a full double window. 编码元组边界后再哈希；未来时间戳需要保留完整的双倍窗口。
	payload := url.Values{"client": {values.Get(names.Client)}, "action": {action}, "nonce": {values.Get(names.Nonce)}}.Encode()
	digest := sha256.Sum256([]byte(payload))
	key := p.prefix + hex.EncodeToString(digest[:])
	ttl := 2 * p.maxAge
	if atomic {
		ok, err := atomicStorage.SetIfAbsent(ctx, key, true, ttl)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
		}
		if !ok {
			return ErrRequestReplayed
		}
		return nil
	}
	existing, err := p.storage.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	if existing != nil {
		return ErrRequestReplayed
	}
	if err := p.storage.Set(ctx, key, true, ttl); err != nil {
		return fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	return nil
}
