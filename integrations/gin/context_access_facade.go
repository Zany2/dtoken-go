// @Author daixk 2026/06/05
package gin

import (
	"github.com/gin-gonic/gin"
)

// CheckRoleByContext checks current user role CheckRoleByContext 鏍￠獙褰撳墠鐢ㄦ埛瑙掕壊
func CheckRoleByContext(c *gin.Context, role string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(requestContext(c), role)
}

// CheckRolesAndByContext checks all current user roles CheckRolesAndByContext 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁鍏ㄩ儴瑙掕壊
func CheckRolesAndByContext(c *gin.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(requestContext(c), roles)
}

// CheckRolesOrByContext checks any current user role CheckRolesOrByContext 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁浠讳竴瑙掕壊
func CheckRolesOrByContext(c *gin.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(requestContext(c), roles)
}

// CheckPermissionByContext checks current user permission CheckPermissionByContext 鏍￠獙褰撳墠鐢ㄦ埛鏉冮檺
func CheckPermissionByContext(c *gin.Context, permission string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(requestContext(c), permission)
}

// CheckPermissionsAndByContext checks all current user permissions CheckPermissionsAndByContext 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁鍏ㄩ儴鏉冮檺
func CheckPermissionsAndByContext(c *gin.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(requestContext(c), permissions)
}

// CheckPermissionsOrByContext checks any current user permission CheckPermissionsOrByContext 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁浠讳竴鏉冮檺
func CheckPermissionsOrByContext(c *gin.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(requestContext(c), permissions)
}
