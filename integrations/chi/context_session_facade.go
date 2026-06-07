// @Author daixk 2026/06/05
package chi

import (
	"context"
)

// SetSessionValueByCtx sets current session value SetSessionValueByCtx ?
func SetSessionValueByCtx(ctx context.Context, key string, value any) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Session().SetValue(ctx, key, value)
}

// GetSessionValueByCtx gets current session value GetSessionValueByCtx ?
func GetSessionValueByCtx(ctx context.Context, key string) (any, bool, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, false, err
	}
	return dCtx.Session().GetValue(ctx, key)
}

// DeleteSessionValueByCtx deletes current session value DeleteSessionValueByCtx ?
func DeleteSessionValueByCtx(ctx context.Context, key string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Session().DeleteValue(ctx, key)
}
