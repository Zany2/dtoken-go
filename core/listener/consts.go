// @Author daixk 2025/12/22 15:56:00
package listener

// Event defines authentication event type. Event 定义认证事件类型。
type Event string

const (
	// EventLogin indicates login event. EventLogin 表示用户登录事件。
	EventLogin Event = "login"
	// EventLogout indicates logout event. EventLogout 表示用户登出事件。
	EventLogout Event = "logout"
	// EventKickout indicates kickout event. EventKickout 表示用户被踢下线事件。
	EventKickout Event = "kickout"
	// EventReplace indicates replace event. EventReplace 表示用户被顶下线事件。
	EventReplace Event = "replace"
	// EventDisable indicates account disable event. EventDisable 表示账号被禁用事件。
	EventDisable Event = "disable"
	// EventUntie indicates account untie event. EventUntie 表示账号解禁事件。
	EventUntie Event = "untie"
	// EventRenew indicates token renew event. EventRenew 表示 Token 续期事件。
	EventRenew Event = "renew"
	// EventCreateSession indicates session creation event. EventCreateSession 表示 Session 创建事件。
	EventCreateSession Event = "createSession"
	// EventDestroySession indicates session destroy event. EventDestroySession 表示 Session 销毁事件。
	EventDestroySession Event = "destroySession"
	// EventPermissionCheck indicates permission check event. EventPermissionCheck 表示权限检查事件。
	EventPermissionCheck Event = "permissionCheck"
	// EventPermissionChange indicates permission mutation event. EventPermissionChange 表示权限变更事件。
	EventPermissionChange Event = "permissionChange"
	// EventRoleCheck indicates role check event. EventRoleCheck 表示角色检查事件。
	EventRoleCheck Event = "roleCheck"
	// EventRoleChange indicates role mutation event. EventRoleChange 表示角色变更事件。
	EventRoleChange Event = "roleChange"
	// EventDisableService indicates service disable event. EventDisableService 表示账号服务被封禁事件。
	EventDisableService Event = "disableService"
	// EventUntieService indicates service untie event. EventUntieService 表示账号服务解禁事件。
	EventUntieService Event = "untieService"
	// EventDisableDevice indicates device disable event. EventDisableDevice 表示账号设备被封禁事件。
	EventDisableDevice Event = "disableDevice"
	// EventUntieDevice indicates device untie event. EventUntieDevice 表示账号设备解禁事件。
	EventUntieDevice Event = "untieDevice"
	// EventRefreshTokenCreate indicates refresh token creation event. EventRefreshTokenCreate 表示刷新令牌创建事件。
	EventRefreshTokenCreate Event = "refreshTokenCreate"
	// EventRefreshTokenRotate indicates refresh token rotation event. EventRefreshTokenRotate 表示刷新令牌轮换事件。
	EventRefreshTokenRotate Event = "refreshTokenRotate"
	// EventRefreshTokenRevoke indicates refresh token revoke event. EventRefreshTokenRevoke 表示刷新令牌撤销事件。
	EventRefreshTokenRevoke Event = "refreshTokenRevoke"
	// EventNonceGenerate indicates nonce generation event. EventNonceGenerate 表示 Nonce 生成事件。
	EventNonceGenerate Event = "nonceGenerate"
	// EventNonceVerify indicates nonce verification event. EventNonceVerify 表示 Nonce 校验事件。
	EventNonceVerify Event = "nonceVerify"
	// EventTicketCreate indicates ticket creation event. EventTicketCreate 表示 Ticket 创建事件。
	EventTicketCreate Event = "ticketCreate"
	// EventTicketValidate indicates ticket validation event. EventTicketValidate 表示 Ticket 校验事件。
	EventTicketValidate Event = "ticketValidate"
	// EventTicketConsume indicates ticket consumption event. EventTicketConsume 表示 Ticket 消费事件。
	EventTicketConsume Event = "ticketConsume"
	// EventTicketRevoke indicates ticket revoke event. EventTicketRevoke 表示 Ticket 撤销事件。
	EventTicketRevoke Event = "ticketRevoke"
	// EventShortKeyCreate indicates short key creation event. EventShortKeyCreate 表示短 Key 创建事件。
	EventShortKeyCreate Event = "shortKeyCreate"
	// EventShortKeyConfirm indicates short key confirmation event. EventShortKeyConfirm 表示短 Key 确认事件。
	EventShortKeyConfirm Event = "shortKeyConfirm"
	// EventShortKeyValidate indicates short key validation event. EventShortKeyValidate 表示短 Key 校验事件。
	EventShortKeyValidate Event = "shortKeyValidate"
	// EventShortKeyConsume indicates short key consumption event. EventShortKeyConsume 表示短 Key 消费事件。
	EventShortKeyConsume Event = "shortKeyConsume"
	// EventShortKeyRevoke indicates short key revoke event. EventShortKeyRevoke 表示短 Key 撤销事件。
	EventShortKeyRevoke Event = "shortKeyRevoke"
	// EventOAuth2ClientRegister indicates OAuth2 client registration event. EventOAuth2ClientRegister 表示 OAuth2 客户端注册事件。
	EventOAuth2ClientRegister Event = "oauth2ClientRegister"
	// EventOAuth2ClientUnregister indicates OAuth2 client unregister event. EventOAuth2ClientUnregister 表示 OAuth2 客户端注销事件。
	EventOAuth2ClientUnregister Event = "oauth2ClientUnregister"
	// EventOAuth2CodeGenerate indicates OAuth2 authorization code generation event. EventOAuth2CodeGenerate 表示 OAuth2 授权码生成事件。
	EventOAuth2CodeGenerate Event = "oauth2CodeGenerate"
	// EventOAuth2TokenIssue indicates OAuth2 token issue event. EventOAuth2TokenIssue 表示 OAuth2 令牌签发事件。
	EventOAuth2TokenIssue Event = "oauth2TokenIssue"
	// EventOAuth2TokenRefresh indicates OAuth2 token refresh event. EventOAuth2TokenRefresh 表示 OAuth2 令牌刷新事件。
	EventOAuth2TokenRefresh Event = "oauth2TokenRefresh"
	// EventOAuth2TokenValidate indicates OAuth2 token validation event. EventOAuth2TokenValidate 表示 OAuth2 令牌校验事件。
	EventOAuth2TokenValidate Event = "oauth2TokenValidate"
	// EventOAuth2TokenRevoke indicates OAuth2 token revoke event. EventOAuth2TokenRevoke 表示 OAuth2 令牌撤销事件。
	EventOAuth2TokenRevoke Event = "oauth2TokenRevoke"
	// EventAll indicates wildcard event. EventAll 表示匹配所有事件的通配符事件。
	EventAll Event = "*"
)

// KnownEvents stores built-in event names. KnownEvents 存储内置事件名称。
var KnownEvents = []Event{
	EventLogin,
	EventLogout,
	EventKickout,
	EventReplace,
	EventDisable,
	EventUntie,
	EventRenew,
	EventCreateSession,
	EventDestroySession,
	EventPermissionCheck,
	EventPermissionChange,
	EventRoleCheck,
	EventRoleChange,
	EventDisableService,
	EventUntieService,
	EventDisableDevice,
	EventUntieDevice,
	EventRefreshTokenCreate,
	EventRefreshTokenRotate,
	EventRefreshTokenRevoke,
	EventNonceGenerate,
	EventNonceVerify,
	EventTicketCreate,
	EventTicketValidate,
	EventTicketConsume,
	EventTicketRevoke,
	EventShortKeyCreate,
	EventShortKeyConfirm,
	EventShortKeyValidate,
	EventShortKeyConsume,
	EventShortKeyRevoke,
	EventOAuth2ClientRegister,
	EventOAuth2ClientUnregister,
	EventOAuth2CodeGenerate,
	EventOAuth2TokenIssue,
	EventOAuth2TokenRefresh,
	EventOAuth2TokenValidate,
	EventOAuth2TokenRevoke,
}

const (
	// ExtraKeyPermission stores permission key. ExtraKeyPermission 存储单个权限字段。
	ExtraKeyPermission = "permission"
	// ExtraKeyPermissions stores permissions key. ExtraKeyPermissions 存储多个权限字段。
	ExtraKeyPermissions = "permissions"
	// ExtraKeyRole stores role key. ExtraKeyRole 存储单个角色字段。
	ExtraKeyRole = "role"
	// ExtraKeyRoles stores roles key. ExtraKeyRoles 存储多个角色字段。
	ExtraKeyRoles = "roles"
	// ExtraKeyLogic stores logic key. ExtraKeyLogic 存储逻辑类型字段。
	ExtraKeyLogic = "logic"
	// ExtraKeyResult stores result key. ExtraKeyResult 存储检查结果字段。
	ExtraKeyResult = "result"
	// ExtraKeyAction stores mutation action key. ExtraKeyAction 存储变更动作字段。
	ExtraKeyAction = "action"
	// ExtraKeyShared stores shared token flag key. ExtraKeyShared 存储共享 Token 标记字段。
	ExtraKeyShared = "shared"
	// ExtraKeyService stores service key. ExtraKeyService 存储服务模块字段。
	ExtraKeyService = "service"
	// ExtraKeyLevel stores level key. ExtraKeyLevel 存储封禁等级字段。
	ExtraKeyLevel = "level"
	// ExtraKeyTokenType stores token type key. ExtraKeyTokenType 存储令牌类型字段。
	ExtraKeyTokenType = "tokenType"
	// ExtraKeyClientID stores OAuth2 client id key. ExtraKeyClientID 存储 OAuth2 客户端 ID 字段。
	ExtraKeyClientID = "clientId"
	// ExtraKeyUserID stores OAuth2 user id key. ExtraKeyUserID 存储 OAuth2 用户 ID 字段。
	ExtraKeyUserID = "userId"
	// ExtraKeyScopes stores scopes key. ExtraKeyScopes 存储授权范围字段。
	ExtraKeyScopes = "scopes"
	// ExtraKeySource stores source key. ExtraKeySource 存储业务来源字段。
	ExtraKeySource = "source"
	// ExtraKeySourceApp stores source app key. ExtraKeySourceApp 存储来源应用字段。
	ExtraKeySourceApp = "sourceApp"
	// ExtraKeyTargetApp stores target app key. ExtraKeyTargetApp 存储目标应用字段。
	ExtraKeyTargetApp = "targetApp"
	// ExtraKeyScene stores scene key. ExtraKeyScene 存储业务场景字段。
	ExtraKeyScene = "scene"
	// ExtraKeyStatus stores lifecycle status key. ExtraKeyStatus 存储生命周期状态字段。
	ExtraKeyStatus = "status"
	// ExtraKeyTTL stores ttl key. ExtraKeyTTL 存储剩余有效期字段。
	ExtraKeyTTL = "ttl"
	// ExtraKeyRefreshToken stores refresh token key. ExtraKeyRefreshToken 存储刷新令牌字段。
	ExtraKeyRefreshToken = "refreshToken"
	// ExtraKeyGrantType stores OAuth2 grant type key. ExtraKeyGrantType 存储 OAuth2 授权类型字段。
	ExtraKeyGrantType = "grantType"
)

const (
	// LogicAnd indicates AND logic. LogicAnd 表示需要满足所有条件的 AND 逻辑。
	LogicAnd = "AND"
	// LogicOr indicates OR logic. LogicOr 表示满足任一条件即可的 OR 逻辑。
	LogicOr = "OR"
)

const (
	// ActionAdd indicates add mutation. ActionAdd 表示添加动作。
	ActionAdd = "add"
	// ActionRemove indicates remove mutation. ActionRemove 表示移除动作。
	ActionRemove = "remove"
	// ActionCreate indicates create action. ActionCreate 表示创建动作。
	ActionCreate = "create"
	// ActionValidate indicates validate action. ActionValidate 表示校验动作。
	ActionValidate = "validate"
	// ActionConsume indicates consume action. ActionConsume 表示消费动作。
	ActionConsume = "consume"
	// ActionRevoke indicates revoke action. ActionRevoke 表示撤销动作。
	ActionRevoke = "revoke"
	// ActionConfirm indicates confirm action. ActionConfirm 表示确认动作。
	ActionConfirm = "confirm"
	// ActionRotate indicates rotate action. ActionRotate 表示轮换动作。
	ActionRotate = "rotate"
	// ActionRegister indicates register action. ActionRegister 表示注册动作。
	ActionRegister = "register"
	// ActionUnregister indicates unregister action. ActionUnregister 表示注销动作。
	ActionUnregister = "unregister"
	// ActionIssue indicates issue action. ActionIssue 表示签发动作。
	ActionIssue = "issue"
	// ActionRefresh indicates refresh action. ActionRefresh 表示刷新动作。
	ActionRefresh = "refresh"
)
