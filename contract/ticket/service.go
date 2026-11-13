// Package ticket defines capability contracts without runtime implementations. 包 ticket 定义能力契约，不依赖运行时实现。
package ticket

import (
	"context"
	"time"
)

// Ticket stores a temporary credential payload. Ticket 存储临时凭证载荷。
type Ticket struct {
	// Ticket stores the random credential value. Ticket 存储随机凭证值。
	Ticket string `json:"ticket"`
	// AuthType stores auth namespace. AuthType 存储认证命名空间。
	AuthType string `json:"authType,omitempty"`
	// LoginID stores subject id. LoginID 存储登录主体 ID。
	LoginID string `json:"loginId,omitempty"`
	// Device stores device type. Device 存储设备类型。
	Device string `json:"device,omitempty"`
	// DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	DeviceID string `json:"deviceId,omitempty"`
	// Source stores business source. Source 存储业务来源。
	Source string `json:"source,omitempty"`
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
	// ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	ExpiresIn int64 `json:"expiresIn"`
	// ExpiresAt stores the precise deadline; zero denotes a legacy ticket. ExpiresAt 存储精确截止时间，零值表示旧版 Ticket。
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	// Status stores lifecycle state. Status 存储生命周期状态。
	Status Status `json:"status"`
}

// CreateOptions defines ticket creation options. CreateOptions 定义 Ticket 创建选项。
type CreateOptions struct {
	// LoginID stores subject id. LoginID 存储登录主体 ID。
	LoginID string
	// Device stores device type. Device 存储设备类型。
	Device string
	// DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	DeviceID string
	// Source stores business source. Source 存储业务来源。
	Source string
	// SourceApp stores issuing application. SourceApp 存储签发应用。
	SourceApp string
	// TargetApp stores consuming application. TargetApp 存储目标应用。
	TargetApp string
	// Scopes stores granted scopes. Scopes 存储授权范围。
	Scopes []string
	// Extra stores extension data. Extra 存储扩展数据。
	Extra map[string]any
	// Timeout overrides default ticket ttl. Timeout 覆盖默认 Ticket 有效期。
	Timeout time.Duration
}

// ValidateOptions defines ticket validation constraints. ValidateOptions 定义 Ticket 校验约束。
type ValidateOptions struct {
	// LoginID requires matching subject id when set. LoginID 非空时要求主体 ID 匹配。
	LoginID string
	// Device requires matching device type when set. Device 非空时要求设备类型匹配。
	Device string
	// DeviceID requires matching concrete device id when set. DeviceID 非空时要求具体设备 ID 匹配。
	DeviceID string
	// Source requires matching business source when set. Source 非空时要求业务来源匹配。
	Source string
	// SourceApp requires matching issuing application when set. SourceApp 非空时要求签发应用匹配。
	SourceApp string
	// TargetApp requires matching consuming application when set. TargetApp 非空时要求目标应用匹配。
	TargetApp string
}

// ConsumeResult stores consumed ticket data. ConsumeResult 存储 Ticket 消费结果。
type ConsumeResult struct {
	// Ticket stores consumed ticket payload. Ticket 存储已消费的 Ticket 载荷。
	Ticket *Ticket
}

// Status defines ticket lifecycle state. Status 定义 Ticket 生命周期状态。
type Status string

const (
	// StatusValid indicates the ticket can still be used. StatusValid 表示 Ticket 当前可用。
	StatusValid Status = "valid"
	// StatusConsumed indicates the ticket has been consumed. StatusConsumed 表示 Ticket 已消费。
	StatusConsumed Status = "consumed"
	// StatusRevoked indicates the ticket has been revoked. StatusRevoked 表示 Ticket 已撤销。
	StatusRevoked Status = "revoked"
	// StatusExpired indicates the ticket has expired. StatusExpired 表示 Ticket 已过期。
	StatusExpired Status = "expired"
	// StatusInvalid indicates the ticket is missing or malformed. StatusInvalid 表示 Ticket 无效或不存在。
	StatusInvalid Status = "invalid"
)

// Service is the request-facing capability contract; implementations retain resource ownership. Service 是面向请求的能力契约，资源所有权由具体实现约定。
type Service interface {
	// CreateTicket creates a temporary  CreateTicket 创建临时 Ticket。
	CreateTicket(ctx context.Context, opts CreateOptions) (*Ticket, error)
	// CreateTicketWithTimeout creates a temporary ticket with timeout. CreateTicketWithTimeout 使用指定有效期创建临时 Ticket。
	CreateTicketWithTimeout(ctx context.Context, opts CreateOptions, timeout time.Duration) (*Ticket, error)
	// ValidateTicket validates a ticket without consuming it. ValidateTicket 校验 Ticket 但不消费。
	ValidateTicket(ctx context.Context, ticketValue string, opts ...ValidateOptions) (*Ticket, error)
	// ConsumeTicket validates and consumes a  ConsumeTicket 校验并消费 Ticket。
	ConsumeTicket(ctx context.Context, ticketValue string, opts ...ValidateOptions) (*ConsumeResult, error)
	// RevokeTicket revokes a  RevokeTicket 撤销 Ticket。
	RevokeTicket(ctx context.Context, ticketValue string) error
	// GetTicketStatus returns ticket lifecycle status. GetTicketStatus 返回 Ticket 生命周期状态。
	GetTicketStatus(ctx context.Context, ticketValue string) (Status, error)
	// GetTicketTTL returns ticket ttl in seconds. GetTicketTTL 获取 Ticket 剩余有效秒数。
	GetTicketTTL(ctx context.Context, ticketValue string) (int64, error)
}
