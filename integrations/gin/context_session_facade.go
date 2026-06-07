// @Author daixk 2026/06/05
package gin

import (
	"github.com/gin-gonic/gin"
)

// SetSessionValueByContext sets current session value SetSessionValueByContext 璁剧疆褰撳墠浼氳瘽鍊?
func SetSessionValueByContext(c *gin.Context, key string, value any) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Session().SetValue(requestContext(c), key, value)
}

// GetSessionValueByContext gets current session value GetSessionValueByContext 鑾峰彇褰撳墠浼氳瘽鍊?
func GetSessionValueByContext(c *gin.Context, key string) (any, bool, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, false, err
	}
	return dCtx.Session().GetValue(requestContext(c), key)
}

// DeleteSessionValueByContext deletes current session value DeleteSessionValueByContext 鍒犻櫎褰撳墠浼氳瘽鍊?
func DeleteSessionValueByContext(c *gin.Context, key string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Session().DeleteValue(requestContext(c), key)
}
