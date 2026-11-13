package contract

// TokenInfo defines token info. TokenInfo 定义 Token 信息。
type TokenInfo struct {
	// AuthType stores auth namespace. AuthType 存储认证命名空间。
	AuthType string `json:"authType"`
	// LoginID stores subject identifier. LoginID 存储主体标识。
	LoginID string `json:"loginId"`
	// Device stores device type. Device 存储设备类型。
	Device string `json:"device"`
	// DeviceID stores concrete device ID. DeviceID 存储具体设备 ID。
	DeviceID string `json:"deviceId"`
	// CreateTime stores token creation time. CreateTime 存储 Token 创建时间。
	CreateTime int64 `json:"createTime"`
	// Timeout stores token timeout seconds. Timeout 存储 Token 过期秒数。
	Timeout int64 `json:"timeout"`
	// ActiveTimeout stores inactive timeout seconds. ActiveTimeout 存储不活跃超时秒数。
	ActiveTimeout int64 `json:"activeTimeout,omitempty"`
	// Extra stores token extension data. Extra 存储 Token 扩展数据。
	Extra map[string]any `json:"extra,omitempty"`
}

// Session defines session object. Session 定义会话对象。
type Session struct {
	// AuthType stores auth namespace. AuthType 存储认证命名空间。
	AuthType string `json:"authType"`
	// LoginID stores subject identifier. LoginID 存储主体标识。
	LoginID string `json:"loginId"`
	// CreateTime stores session creation time. CreateTime 存储会话创建时间。
	CreateTime int64 `json:"createTime"`
	// TerminalInfos stores online terminals. TerminalInfos 存储在线终端。
	TerminalInfos []TerminalInfo `json:"terminalInfos,omitempty"`
	// Permissions stores cached permissions. Permissions 存储缓存权限。
	Permissions []string `json:"permissions,omitempty"`
	// Roles stores cached roles. Roles 存储缓存角色。
	Roles []string `json:"roles,omitempty"`
	// Data stores user session extension data. Data 存储用户会话扩展数据。
	Data map[string]any `json:"data,omitempty"`
	// HistoryTerminalCount stores terminal sequence count. HistoryTerminalCount 存储终端历史序号。
	HistoryTerminalCount int64 `json:"historyTerminalCount,omitempty"`
}

// TerminalInfo defines terminal info. TerminalInfo 定义终端信息。
type TerminalInfo struct {
	// Token stores terminal token. Token 存储终端 Token。
	Token string `json:"token"`
	// LoginID stores subject identifier. LoginID 存储主体标识。
	LoginID string `json:"loginId"`
	// Device stores device type. Device 存储设备类型。
	Device string `json:"device"`
	// DeviceID stores concrete device ID. DeviceID 存储具体设备 ID。
	DeviceID string `json:"deviceId"`
	// CreateTime stores terminal creation time. CreateTime 存储终端创建时间。
	CreateTime int64 `json:"createTime"`
	// Index stores terminal sequence. Index 存储终端序号。
	Index int64 `json:"index"`
	// Extra stores terminal extension data. Extra 存储终端扩展数据。
	Extra map[string]any `json:"extra,omitempty"`
}

// DisableInfo defines account disable info. DisableInfo 定义账号封禁信息。
type DisableInfo struct {
	// DisableTime stores disable timestamp. DisableTime 存储封禁时间戳。
	DisableTime int64 `json:"disableTime"`
	// DisableReason stores disable reason. DisableReason 存储封禁原因。
	DisableReason string `json:"disableReason"`
}

// ServiceDisableInfo defines service disable info. ServiceDisableInfo 定义服务封禁信息。
type ServiceDisableInfo struct {
	// Service stores disabled service name. Service 存储被封禁服务名。
	Service string `json:"service"`
	// Level stores disable level. Level 存储封禁等级。
	Level int `json:"level"`
	// DisableTime stores disable timestamp. DisableTime 存储封禁时间戳。
	DisableTime int64 `json:"disableTime"`
	// DisableReason stores disable reason. DisableReason 存储封禁原因。
	DisableReason string `json:"disableReason"`
}

// DeviceDisableInfo defines device disable info. DeviceDisableInfo 定义设备封禁信息。
type DeviceDisableInfo struct {
	// Device stores disabled device type. Device 存储被封禁设备类型。
	Device string `json:"device"`
	// DeviceID stores disabled concrete device ID. DeviceID 存储被封禁具体设备 ID。
	DeviceID string `json:"deviceId"`
	// DisableTime stores disable timestamp. DisableTime 存储封禁时间戳。
	DisableTime int64 `json:"disableTime"`
	// DisableReason stores disable reason. DisableReason 存储封禁原因。
	DisableReason string `json:"disableReason"`
}
