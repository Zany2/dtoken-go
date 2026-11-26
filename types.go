package dtoken

import "github.com/Zany2/dtoken-go/contract"

// Public domain types are shared with component implementations. 公开领域类型与组件实现共享。
type (
	TokenInfo          = contract.TokenInfo
	Session            = contract.Session
	TerminalInfo       = contract.TerminalInfo
	DisableInfo        = contract.DisableInfo
	ServiceDisableInfo = contract.ServiceDisableInfo
	DeviceDisableInfo  = contract.DeviceDisableInfo
	AccessSubject      = contract.AccessSubject
	AccessProvider     = contract.AccessProvider
	AccessProviderFunc = contract.AccessProviderFunc
)
