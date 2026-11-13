// Package nonce defines capability contracts without runtime implementations. 包 nonce 定义能力契约，不依赖运行时实现。
package nonce

import (
	"context"
	"time"
)

// Service is the request-facing capability contract; implementations retain resource ownership. Service 是面向请求的能力契约，资源所有权由具体实现约定。
type Service interface {
	// GenerateNonce generates nonce with default timeout. GenerateNonce 使用默认有效期生成 nonce。
	GenerateNonce(ctx context.Context) (string, error)
	// GenerateNonceWithTimeout generates nonce with custom timeout. GenerateNonceWithTimeout 使用指定有效期生成 nonce。
	GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration) (string, error)
	// VerifyNonce verifies and consumes  VerifyNonce 验证并消费一次 nonce。
	VerifyNonce(ctx context.Context, nonce string) bool
	// VerifyAndConsumeNonce verifies and consumes nonce with error detail. VerifyAndConsumeNonce 验证并消费 nonce，失败时返回错误。
	VerifyAndConsumeNonce(ctx context.Context, nonce string) error
	// IsNonceValid checks nonce validity without consuming it. IsNonceValid 检查 nonce 是否有效且不消费。
	IsNonceValid(ctx context.Context, nonce string) bool
	// GetNonceTTL gets nonce ttl in seconds. GetNonceTTL 获取 nonce 剩余有效秒数。
	GetNonceTTL(ctx context.Context, nonce string) (int64, error)
}
