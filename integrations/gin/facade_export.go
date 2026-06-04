// @Author daixk 2025/12/22 15:56:00
package gin

import "github.com/Zany2/dtoken-go/dtoken"

// Additional facade operations keep framework imports self-contained. Additional facade 操作保持框架包自包含。
var (
	CheckPermissionByToken      = dtoken.CheckPermissionByToken
	CheckPermissionAndByToken   = dtoken.CheckPermissionAndByToken
	CheckPermissionOrByToken    = dtoken.CheckPermissionOrByToken
	CheckRoleByToken            = dtoken.CheckRoleByToken
	CheckRoleAndByToken         = dtoken.CheckRoleAndByToken
	CheckRoleOrByToken          = dtoken.CheckRoleOrByToken
	SetSessionValue             = dtoken.SetSessionValue
	GetSessionValue             = dtoken.GetSessionValue
	DeleteSessionValue          = dtoken.DeleteSessionValue
	CreateTicket                = dtoken.CreateTicket
	CreateTicketWithOptions     = dtoken.CreateTicketWithOptions
	ValidateTicket              = dtoken.ValidateTicket
	ValidateTicketWithOptions   = dtoken.ValidateTicketWithOptions
	ConsumeTicket               = dtoken.ConsumeTicket
	ConsumeTicketWithOptions    = dtoken.ConsumeTicketWithOptions
	RevokeTicket                = dtoken.RevokeTicket
	GetTicketStatus             = dtoken.GetTicketStatus
	GetTicketTTL                = dtoken.GetTicketTTL
	CreateShortKey              = dtoken.CreateShortKey
	CreateShortKeyWithOptions   = dtoken.CreateShortKeyWithOptions
	ConfirmShortKey             = dtoken.ConfirmShortKey
	ConfirmShortKeyWithOptions  = dtoken.ConfirmShortKeyWithOptions
	ValidateShortKey            = dtoken.ValidateShortKey
	ValidateShortKeyWithOptions = dtoken.ValidateShortKeyWithOptions
	ConsumeShortKey             = dtoken.ConsumeShortKey
	ConsumeShortKeyWithOptions  = dtoken.ConsumeShortKeyWithOptions
	RevokeShortKey              = dtoken.RevokeShortKey
	GetShortKeyStatus           = dtoken.GetShortKeyStatus
	GetShortKeyTTL              = dtoken.GetShortKeyTTL
)
