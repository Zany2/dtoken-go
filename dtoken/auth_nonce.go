// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"
	"time"
)

// GenerateNonce creates a nonce. GenerateNonce 创建一次性 nonce。
func (a *Auth) GenerateNonce(ctx context.Context) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.GenerateNonce(ctx)
}

// GenerateNonceWithTimeout creates a nonce with custom timeout. GenerateNonceWithTimeout 使用自定义过期时间创建 nonce。
func (a *Auth) GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.GenerateNonceWithTimeout(ctx, timeout)
}

// VerifyNonce verifies and consumes nonce. VerifyNonce 验证并消费 nonce。
func (a *Auth) VerifyNonce(ctx context.Context, nonce string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.VerifyNonce(ctx, nonce)
}

// VerifyAndConsumeNonce verifies and consumes nonce with error detail. VerifyAndConsumeNonce 验证并消费 nonce，失败时返回错误。
func (a *Auth) VerifyAndConsumeNonce(ctx context.Context, nonce string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.VerifyAndConsumeNonce(ctx, nonce)
}

// IsNonceValid checks whether a nonce is valid without consuming it. IsNonceValid 检查 nonce 是否有效且不消费。
func (a *Auth) IsNonceValid(ctx context.Context, nonce string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.IsNonceValid(ctx, nonce)
}

// GetNonceTTL returns the remaining ttl of a nonce in seconds. GetNonceTTL 返回 nonce 剩余有效秒数。
func (a *Auth) GetNonceTTL(ctx context.Context, nonce string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetNonceTTL(ctx, nonce)
}
