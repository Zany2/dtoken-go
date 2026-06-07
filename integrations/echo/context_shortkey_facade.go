// @Author daixk 2026/06/05
package echo

import (
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
	echo4 "github.com/labstack/echo/v4"
)

// CreateShortKeyByContext creates short key CreateShortKeyByContext 鍒涘缓鐭?Key
func CreateShortKeyByContext(c echo4.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Create(requestContext(c), opts)
}

// CreateShortKeyWithTimeoutByContext creates short key with timeout CreateShortKeyWithTimeoutByContext 浣跨敤鎸囧畾鏈夋晥鏈熷垱寤虹煭 Key
func CreateShortKeyWithTimeoutByContext(c echo4.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().CreateWithTimeout(requestContext(c), opts, timeout)
}

// ConfirmShortKeyByContext confirms short key ConfirmShortKeyByContext 纭鐭?Key
func ConfirmShortKeyByContext(c echo4.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Confirm(requestContext(c), key, opts)
}

// ConfirmShortKeyForCurrentLoginByContext confirms short key for current user ConfirmShortKeyForCurrentLoginByContext 使用当前用户确认 ShortKey
func ConfirmShortKeyForCurrentLoginByContext(c echo4.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().ConfirmForCurrentLogin(requestContext(c), key, opts)
}

// ValidateShortKeyByContext validates short key ValidateShortKeyByContext 鏍￠獙鐭?Key
func ValidateShortKeyByContext(c echo4.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Validate(requestContext(c), key, opts...)
}

// ConsumeShortKeyByContext consumes short key ConsumeShortKeyByContext 娑堣垂鐭?Key
func ConsumeShortKeyByContext(c echo4.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.ShortKey().Consume(requestContext(c), key, opts...)
}

// RevokeShortKeyByContext revokes short key RevokeShortKeyByContext 鎾ら攢鐭?Key
func RevokeShortKeyByContext(c echo4.Context, key string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.ShortKey().Revoke(requestContext(c), key)
}

// GetShortKeyStatusByContext gets short key status GetShortKeyStatusByContext 鑾峰彇鐭?Key 鐘舵€?
func GetShortKeyStatusByContext(c echo4.Context, key string) (shortkey.Status, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.ShortKey().GetStatus(requestContext(c), key)
}

// GetShortKeyTTLByContext gets short key TTL GetShortKeyTTLByContext 鑾峰彇鐭?Key 鍓╀綑鏈夋晥鏈?
func GetShortKeyTTLByContext(c echo4.Context, key string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.ShortKey().GetTTL(requestContext(c), key)
}
