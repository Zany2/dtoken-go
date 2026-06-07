// @Author daixk 2026/06/05
package gf

import (
	"context"
)

// SetSessionValueByCtx sets current session value SetSessionValueByCtx 璁剧疆褰撳墠浼氳瘽鍊?
func SetSessionValueByCtx(ctx context.Context, key string, value any) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Session().SetValue(ctx, key, value)
}

// GetSessionValueByCtx gets current session value GetSessionValueByCtx 鑾峰彇褰撳墠浼氳瘽鍊?
func GetSessionValueByCtx(ctx context.Context, key string) (any, bool, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, false, err
	}
	return dCtx.Session().GetValue(ctx, key)
}

// DeleteSessionValueByCtx deletes current session value DeleteSessionValueByCtx 鍒犻櫎褰撳墠浼氳瘽鍊?
func DeleteSessionValueByCtx(ctx context.Context, key string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Session().DeleteValue(ctx, key)
}
