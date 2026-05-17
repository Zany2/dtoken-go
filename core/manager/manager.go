// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"sync"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// Manager is the core auth facade. Manager 是鉴权核心门面。
type Manager struct {
	config *config.Config // config stores runtime configuration. config 存储运行时配置。

	generator  adapter.Generator // generator creates token values. generator 负责生成 Token 值。
	storage    adapter.Storage   // storage persists auth data. storage 持久化鉴权数据。
	serializer adapter.Codec     // serializer encodes and decodes storage payloads. serializer 编解码存储数据。
	logger     adapter.Log       // logger writes framework logs. logger 写入框架日志。
	pool       adapter.Pool      // pool runs asynchronous tasks. pool 执行异步任务。

	nonceManager   *nonce.NonceManager        // nonceManager handles one-time nonce values. nonceManager 管理一次性 nonce。
	oauth2Manager  *oauth2.OAuth2Server       // oauth2Manager handles OAuth2 flows. oauth2Manager 处理 OAuth2 流程。
	eventManager   *listener.Manager          // eventManager dispatches auth events. eventManager 分发鉴权事件。
	loginLocksMu   sync.Mutex                 // loginLocksMu protects the login lock registry. loginLocksMu 保护登录锁注册表。
	loginLocks     map[string]*loginLockEntry // loginLocks serializes writes per login ID. loginLocks 按登录 ID 串行化写操作。
	accessProvider AccessProvider             // accessProvider resolves roles and permissions. accessProvider 解析角色和权限。
}

// TokenInfo defines token info. TokenInfo 定义 Token 信息。
type TokenInfo struct {
	AuthType   string `json:"authType"`   // AuthType stores auth namespace. AuthType 存储认证命名空间。
	LoginID    string `json:"loginId"`    // LoginID stores subject identifier. LoginID 存储主体标识。
	Device     string `json:"device"`     // Device stores device type. Device 存储设备类型。
	DeviceId   string `json:"deviceId"`   // DeviceId stores concrete device ID. DeviceId 存储具体设备 ID。
	CreateTime int64  `json:"createTime"` // CreateTime stores token creation time. CreateTime 存储 Token 创建时间。
	Timeout    int64  `json:"timeout"`    // Timeout stores token timeout seconds. Timeout 存储 Token 过期秒数。
}

// Session defines session object. Session 定义会话对象。
type Session struct {
	AuthType             string         `json:"authType"`                       // AuthType stores auth namespace. AuthType 存储认证命名空间。
	LoginID              string         `json:"loginId"`                        // LoginID stores subject identifier. LoginID 存储主体标识。
	CreateTime           int64          `json:"createTime"`                     // CreateTime stores session creation time. CreateTime 存储会话创建时间。
	TerminalInfos        []TerminalInfo `json:"terminalInfos,omitempty"`        // TerminalInfos stores online terminals. TerminalInfos 存储在线终端。
	Permissions          []string       `json:"permissions,omitempty"`          // Permissions stores cached permissions. Permissions 存储缓存权限。
	Roles                []string       `json:"roles,omitempty"`                // Roles stores cached roles. Roles 存储缓存角色。
	HistoryTerminalCount int64          `json:"historyTerminalCount,omitempty"` // HistoryTerminalCount stores terminal sequence count. HistoryTerminalCount 存储终端历史序号。
}

// TerminalInfo defines terminal info. TerminalInfo 定义终端信息。
type TerminalInfo struct {
	Token      string `json:"token"`      // Token stores terminal token. Token 存储终端 Token。
	LoginID    string `json:"loginId"`    // LoginID stores subject identifier. LoginID 存储主体标识。
	Device     string `json:"device"`     // Device stores device type. Device 存储设备类型。
	DeviceId   string `json:"deviceId"`   // DeviceId stores concrete device ID. DeviceId 存储具体设备 ID。
	CreateTime int64  `json:"createTime"` // CreateTime stores terminal creation time. CreateTime 存储终端创建时间。
	Index      int64  `json:"index"`      // Index stores terminal sequence. Index 存储终端序号。
}

// DisableInfo defines account disable info. DisableInfo 定义账号封禁信息。
type DisableInfo struct {
	DisableTime   int64  `json:"disableTime"`   // DisableTime stores disable timestamp. DisableTime 存储封禁时间戳。
	DisableReason string `json:"disableReason"` // DisableReason stores disable reason. DisableReason 存储封禁原因。
}

// ServiceDisableInfo defines service disable info. ServiceDisableInfo 定义服务封禁信息。
type ServiceDisableInfo struct {
	Service       string `json:"service"`       // Service stores disabled service name. Service 存储被封禁服务名。
	Level         int    `json:"level"`         // Level stores disable level. Level 存储封禁等级。
	DisableTime   int64  `json:"disableTime"`   // DisableTime stores disable timestamp. DisableTime 存储封禁时间戳。
	DisableReason string `json:"disableReason"` // DisableReason stores disable reason. DisableReason 存储封禁原因。
}

// DeviceDisableInfo defines device disable info. DeviceDisableInfo 定义设备封禁信息。
type DeviceDisableInfo struct {
	Device        string `json:"device"`        // Device stores disabled device type. Device 存储被封禁设备类型。
	DeviceId      string `json:"deviceId"`      // DeviceId stores disabled concrete device ID. DeviceId 存储被封禁具体设备 ID。
	DisableTime   int64  `json:"disableTime"`   // DisableTime stores disable timestamp. DisableTime 存储封禁时间戳。
	DisableReason string `json:"disableReason"` // DisableReason stores disable reason. DisableReason 存储封禁原因。
}
