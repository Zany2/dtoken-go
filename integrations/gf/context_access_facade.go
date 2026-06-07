// @Author daixk 2026/06/05
package gf

import (
	"context"
)

// CheckRoleByCtx checks current user role CheckRoleByCtx 鏍￠獙褰撳墠鐢ㄦ埛瑙掕壊
func CheckRoleByCtx(ctx context.Context, role string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(ctx, role)
}

// CheckRolesAndByCtx checks all current user roles CheckRolesAndByCtx 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁鍏ㄩ儴瑙掕壊
func CheckRolesAndByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(ctx, roles)
}

// CheckRolesOrByCtx checks any current user role CheckRolesOrByCtx 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁浠讳竴瑙掕壊
func CheckRolesOrByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(ctx, roles)
}

// CheckPermissionByCtx checks current user permission CheckPermissionByCtx 鏍￠獙褰撳墠鐢ㄦ埛鏉冮檺
func CheckPermissionByCtx(ctx context.Context, permission string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(ctx, permission)
}

// CheckPermissionsAndByCtx checks all current user permissions CheckPermissionsAndByCtx 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁鍏ㄩ儴鏉冮檺
func CheckPermissionsAndByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(ctx, permissions)
}

// CheckPermissionsOrByCtx checks any current user permission CheckPermissionsOrByCtx 鏍￠獙褰撳墠鐢ㄦ埛鏄惁鎷ユ湁浠讳竴鏉冮檺
func CheckPermissionsOrByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(ctx, permissions)
}
