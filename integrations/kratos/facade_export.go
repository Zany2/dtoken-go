// @Author daixk 2025/12/22 15:56:00
package kratos

import "github.com/Zany2/dtoken-go/dtoken"

// Additional facade operations keep framework imports self-contained Additional facade 操作保持框架包自包含
var (
	CheckPermissionByToken    = dtoken.CheckPermissionByToken
	CheckPermissionAndByToken = dtoken.CheckPermissionAndByToken
	CheckPermissionOrByToken  = dtoken.CheckPermissionOrByToken
	CheckRoleByToken          = dtoken.CheckRoleByToken
	CheckRoleAndByToken       = dtoken.CheckRoleAndByToken
	CheckRoleOrByToken        = dtoken.CheckRoleOrByToken
)
