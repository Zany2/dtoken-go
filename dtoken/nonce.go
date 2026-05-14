package dtoken

import (
	"context"
	"time"
)

// GenerateNonce generates a new nonce with the default ttl. GenerateNonce 使用默认有效期生成 nonce。
func GenerateNonce(ctx context.Context, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GenerateNonce(ctx)
}

// GenerateNonceWithTimeout generates a new nonce with a custom ttl. GenerateNonceWithTimeout 使用自定义有效期生成 nonce。
func GenerateNonceWithTimeout(ctx context.Context, timeout time.Duration, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GenerateNonceWithTimeout(ctx, timeout)
}

// VerifyNonce verifies and consumes a nonce. VerifyNonce 验证并消费 nonce。
func VerifyNonce(ctx context.Context, nonce string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.VerifyNonce(ctx, nonce)
}

// VerifyAndConsumeNonce verifies and consumes a nonce with error detail. VerifyAndConsumeNonce 验证并消费 nonce，失败时返回错误。
func VerifyAndConsumeNonce(ctx context.Context, nonce string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.VerifyAndConsumeNonce(ctx, nonce)
}

// IsNonceValid checks whether a nonce is valid without consuming it. IsNonceValid 检查 nonce 是否有效且不消费。
func IsNonceValid(ctx context.Context, nonce string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsNonceValid(ctx, nonce)
}

// GetNonceTTL returns the remaining ttl of a nonce in seconds. GetNonceTTL 返回 nonce 剩余有效秒数。
func GetNonceTTL(ctx context.Context, nonce string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetNonceTTL(ctx, nonce)
}
