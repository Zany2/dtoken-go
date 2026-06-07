// @Author daixk 2026/06/05
package hertz

import (
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// SetSessionValueByContext sets current session value SetSessionValueByContext ?
func SetSessionValueByContext(ctx *hertzapp.RequestContext, key string, value any) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Session().SetValue(requestContext(ctx), key, value)
}

// GetSessionValueByContext gets current session value GetSessionValueByContext ?
func GetSessionValueByContext(ctx *hertzapp.RequestContext, key string) (any, bool, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, false, err
	}
	return dCtx.Session().GetValue(requestContext(ctx), key)
}

// DeleteSessionValueByContext deletes current session value DeleteSessionValueByContext ?
func DeleteSessionValueByContext(ctx *hertzapp.RequestContext, key string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Session().DeleteValue(requestContext(ctx), key)
}
