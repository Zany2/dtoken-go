// Package shortkey defines capability contracts without runtime implementations. 包 shortkey 定义能力契约，不依赖运行时实现。
package shortkey

import (
	"context"
	"time"
)

// ShortKey stores an interactive short credential payload. ShortKey 存储交互式短凭证载荷。
type ShortKey struct {
	// Key stores short credential value. Key 存储短凭证值。
	Key string `json:"key"`
	// AuthType stores auth namespace. AuthType 存储认证命名空间。
	AuthType string `json:"authType,omitempty"`
	// LoginID stores confirmed subject id. LoginID 存储确认后的主体 ID。
	LoginID string `json:"loginId,omitempty"`
	// Device stores device type. Device 存储设备类型。
	Device string `json:"device,omitempty"`
	// DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	DeviceID string `json:"deviceId,omitempty"`
	// Scene stores business scene. Scene 存储业务场景。
	Scene string `json:"scene,omitempty"`
	// SourceApp stores issuing application. SourceApp 存储签发应用。
	SourceApp string `json:"sourceApp,omitempty"`
	// TargetApp stores consuming application. TargetApp 存储目标应用。
	TargetApp string `json:"targetApp,omitempty"`
	// Scopes stores granted scopes. Scopes 存储授权范围。
	Scopes []string `json:"scopes,omitempty"`
	// Extra stores extension data. Extra 存储扩展数据。
	Extra map[string]any `json:"extra,omitempty"`
	// CreateTime stores creation unix time. CreateTime 存储创建时间戳。
	CreateTime int64 `json:"createTime"`
	// UpdateTime stores update unix time. UpdateTime 存储更新时间戳。
	UpdateTime int64 `json:"updateTime"`
	// ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	ExpiresIn int64 `json:"expiresIn"`
	// ExpiresAt stores the precise deadline; zero denotes a legacy short key. ExpiresAt 存储精确截止时间，零值表示旧版短 Key。
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	// Status stores lifecycle state. Status 存储生命周期状态。
	Status Status `json:"status"`
}

// CreateOptions defines short key creation options. CreateOptions 定义短 Key 创建选项。
type CreateOptions struct {
	// LoginID stores subject id and confirms the key when set. LoginID 存储主体 ID，非空时直接确认短 Key。
	LoginID string
	// Device stores device type. Device 存储设备类型。
	Device string
	// DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	DeviceID string
	// Scene stores business scene. Scene 存储业务场景。
	Scene string
	// SourceApp stores issuing application. SourceApp 存储签发应用。
	SourceApp string
	// TargetApp stores consuming application. TargetApp 存储目标应用。
	TargetApp string
	// Scopes stores granted scopes. Scopes 存储授权范围。
	Scopes []string
	// Extra stores extension data. Extra 存储扩展数据。
	Extra map[string]any
	// Timeout overrides default short key ttl. Timeout 覆盖默认短 Key 有效期。
	Timeout time.Duration
}

// ConfirmOptions defines short key confirmation data. ConfirmOptions 定义短 Key 确认数据。
type ConfirmOptions struct {
	// LoginID stores confirmed subject id. LoginID 存储确认后的主体 ID。
	LoginID string
	// Device stores confirmed device type. Device 存储确认后的设备类型。
	Device string
	// DeviceID stores confirmed concrete device id. DeviceID 存储确认后的具体设备 ID。
	DeviceID string
	// Scopes replaces granted scopes when set. Scopes 非空时替换授权范围。
	Scopes []string
	// Extra replaces extension data when set. Extra 非空时替换扩展数据。
	Extra map[string]any
}

// ValidateOptions defines short key validation constraints. ValidateOptions 定义短 Key 校验约束。
type ValidateOptions struct {
	// LoginID requires matching subject id when set. LoginID 非空时要求主体 ID 匹配。
	LoginID string
	// Device requires matching device type when set. Device 非空时要求设备类型匹配。
	Device string
	// DeviceID requires matching concrete device id when set. DeviceID 非空时要求具体设备 ID 匹配。
	DeviceID string
	// Scene requires matching business scene when set. Scene 非空时要求业务场景匹配。
	Scene string
	// SourceApp requires matching issuing application when set. SourceApp 非空时要求签发应用匹配。
	SourceApp string
	// TargetApp requires matching consuming application when set. TargetApp 非空时要求目标应用匹配。
	TargetApp string
}

// ConsumeResult stores consumed short key data. ConsumeResult 存储短 Key 消费结果。
type ConsumeResult struct {
	// ShortKey stores consumed short key payload. ShortKey 存储已消费的短 Key 载荷。
	ShortKey *ShortKey
}

// Status defines short key lifecycle state. Status 定义短 Key 生命周期状态。
type Status string

const (
	// StatusPending indicates the key is waiting for confirmation. StatusPending 表示短 Key 等待确认。
	StatusPending Status = "pending"
	// StatusConfirmed indicates the key has been confirmed and can be consumed. StatusConfirmed 表示短 Key 已确认且可消费。
	StatusConfirmed Status = "confirmed"
	// StatusConsumed indicates the key has been consumed. StatusConsumed 表示短 Key 已消费。
	StatusConsumed Status = "consumed"
	// StatusRevoked indicates the key has been revoked. StatusRevoked 表示短 Key 已撤销。
	StatusRevoked Status = "revoked"
	// StatusExpired indicates the key has expired. StatusExpired 表示短 Key 已过期。
	StatusExpired Status = "expired"
	// StatusInvalid indicates the key is missing or malformed. StatusInvalid 表示短 Key 无效或不存在。
	StatusInvalid Status = "invalid"
)

// Service is the request-facing capability contract; implementations retain resource ownership. Service 是面向请求的能力契约，资源所有权由具体实现约定。
type Service interface {
	// CreateShortKey creates a pending short key. CreateShortKey 创建待确认短 Key。
	CreateShortKey(ctx context.Context, opts CreateOptions) (*ShortKey, error)
	// CreateShortKeyWithTimeout creates a pending short key with timeout. CreateShortKeyWithTimeout 使用指定有效期创建待确认短 Key。
	CreateShortKeyWithTimeout(ctx context.Context, opts CreateOptions, timeout time.Duration) (*ShortKey, error)
	// ConfirmShortKey confirms a pending short key. ConfirmShortKey 确认待处理短 Key。
	ConfirmShortKey(ctx context.Context, key string, opts ConfirmOptions) (*ShortKey, error)
	// ValidateShortKey validates a short key without consuming it. ValidateShortKey 校验短 Key 但不消费。
	ValidateShortKey(ctx context.Context, key string, opts ...ValidateOptions) (*ShortKey, error)
	// ConsumeShortKey validates and consumes a short key. ConsumeShortKey 校验并消费短 Key。
	ConsumeShortKey(ctx context.Context, key string, opts ...ValidateOptions) (*ConsumeResult, error)
	// RevokeShortKey revokes a short key. RevokeShortKey 撤销短 Key。
	RevokeShortKey(ctx context.Context, key string) error
	// GetShortKeyStatus returns short key lifecycle status. GetShortKeyStatus 返回短 Key 生命周期状态。
	GetShortKeyStatus(ctx context.Context, key string) (Status, error)
	// GetShortKeyTTL returns short key ttl in seconds. GetShortKeyTTL 获取短 Key 剩余有效秒数。
	GetShortKeyTTL(ctx context.Context, key string) (int64, error)
}
