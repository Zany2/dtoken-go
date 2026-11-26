package sso

import (
	"context"
	"time"
)

// SessionBackend manages authenticated sessions without HTTP or business credential verification. SessionBackend 管理已认证会话，不处理 HTTP 或业务凭据验证。
type SessionBackend interface {
	// Resolve returns nil without error for an inactive token, and an error for backend failures. Resolve 对无效 Token 返回 nil 且无错误，对后端故障返回错误。
	Resolve(ctx context.Context, token string) (*SessionInfo, error)
	// Create accepts a trusted subject and grant, never unverified browser identity fields. Create 接受可信主体及授权，禁止直接传入未校验的浏览器身份字段。
	Create(ctx context.Context, request SessionCreate) (*SessionInfo, error)
	// Revoke idempotently revokes the specified local target; it does not coordinate global logout. Revoke 幂等撤销指定本地目标，不负责协调全局退出。
	Revoke(ctx context.Context, request SessionRevoke) error
}

// SessionGrant binds a local session to a trusted center login lifecycle, not a bearer credential. SessionGrant 将本地会话关联到可信中心登录生命周期，自身不是登录凭据。
type SessionGrant struct {
	// CenterID identifies the subject at the center. CenterID 标识中心主体。
	CenterID string
	// CenterSessionID identifies one center login lifecycle and changes after re-login. CenterSessionID 标识一次中心登录生命周期，重新登录后变化。
	CenterSessionID string
	// ClientID identifies the authorized application. ClientID 标识获授权应用。
	ClientID string
	// ExpiresAt bounds this grant; zero means no fixed center deadline, not unlimited local lifetime. ExpiresAt 限制本次授权期限，零值表示中心无固定截止时间，不表示本地无限期。
	ExpiresAt time.Time
}

// SessionInfo describes a resolved local session; private tokens must never enter protocol JSON. SessionInfo 描述已解析的本地会话，私有 Token 不得进入协议 JSON。
type SessionInfo struct {
	// LoginID identifies the local authenticated subject. LoginID 标识本地已认证主体。
	LoginID string
	// Token is a private backend credential, excluded from JSON. Token 是后端私有凭据，不参与 JSON 序列化。
	Token string `json:"-"`
	// Namespace identifies the authentication data scope. Namespace 标识认证数据范围。
	Namespace string
	// Device identifies the local terminal category. Device 标识本地终端类别。
	Device string
	// DeviceID identifies the local terminal, not the center lifecycle. DeviceID 标识本地终端，不代替中心生命周期。
	DeviceID string
	// ExpiresAt is the absolute local deadline; zero denotes no fixed deadline. ExpiresAt 是本地绝对截止时间，零值表示无固定期限。
	ExpiresAt time.Time
	// ActiveExpiresAt is the idle deadline; zero denotes no idle deadline. ActiveExpiresAt 是活跃截止时间，零值表示无活跃期限。
	ActiveExpiresAt time.Time
	// Grant contains the trusted center association when this is a child session. Grant 在子会话中保存可信中心关联。
	Grant *SessionGrant
}

// SessionCreate requests a session for an already verified subject; adapters must honor both deadlines. SessionCreate 为已验证主体请求会话，适配器必须同时遵守本地与授权期限。
type SessionCreate struct {
	// LoginID is the trusted local subject after any permitted mapping. LoginID 是经过允许的身份映射后的可信本地主体。
	LoginID string
	// ClientID separates application terminals; empty is reserved for center login. ClientID 区分应用终端，空值保留给中心登录。
	ClientID string
	// ExpiresAt bounds local lifetime; zero uses the backend's configured limit. ExpiresAt 限制本地期限，零值使用后端配置上限。
	ExpiresAt time.Time
	// Grant is required for child sessions and nil for a new center login. Grant 在子会话中必填，在新中心登录时为 nil。
	Grant *SessionGrant
}

// SessionRevokeScope distinguishes local backend operations from center-coordinated logout. SessionRevokeScope 区分本地后端操作与中心协调的全局退出。
type SessionRevokeScope string

const (
	// RevokeSessionToken revokes only the supplied local token. RevokeSessionToken 仅撤销指定本地 Token。
	RevokeSessionToken SessionRevokeScope = "token"
	// RevokeSessionAccount revokes all local sessions of the supplied subject. RevokeSessionAccount 撤销指定主体的全部本地会话。
	RevokeSessionAccount SessionRevokeScope = "account"
)

// SessionRevoke names a verified local target; browser-global logout resolves links into token operations. SessionRevoke 指定已验证的本地目标，浏览器全局退出需先将关联解析为 Token 操作。
type SessionRevoke struct {
	// Scope must be explicit; unknown values must fail without side effects. Scope 必须显式指定，未知值须无副作用地失败。
	Scope SessionRevokeScope
	// Token is required for RevokeSessionToken and excluded from JSON. Token 在 RevokeSessionToken 时必填且不参与 JSON 序列化。
	Token string `json:"-"`
	// LoginID is required for RevokeSessionAccount. LoginID 在 RevokeSessionAccount 时必填。
	LoginID string
}
