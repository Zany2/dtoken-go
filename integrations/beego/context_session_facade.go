// @Author daixk 2026/06/06
package beego

import beegocontext "github.com/beego/beego/v2/server/web/context"

// SetSessionValueByContext sets current session value SetSessionValueByContext 设置当前会话值
func SetSessionValueByContext(c *beegocontext.Context, key string, value any) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Session().SetValue(requestContext(c), key, value)
}

// GetSessionValueByContext gets current session value GetSessionValueByContext 获取当前会话值
func GetSessionValueByContext(c *beegocontext.Context, key string) (any, bool, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, false, err
	}
	return dCtx.Session().GetValue(requestContext(c), key)
}

// DeleteSessionValueByContext deletes current session value DeleteSessionValueByContext 删除当前会话值
func DeleteSessionValueByContext(c *beegocontext.Context, key string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Session().DeleteValue(requestContext(c), key)
}
