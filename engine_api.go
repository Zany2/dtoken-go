package dtoken

import (
	"github.com/Zany2/dtoken-go/internal/builder"
	"github.com/Zany2/dtoken-go/internal/engine"
)

// Engine extension contracts remain nameable outside the implementation packages. 引擎扩展契约可由实现包之外的调用方直接引用。
type (
	// Manager is the supported low-level API for custom integrations, obtained from a built Auth; zero-value construction and configuration or component mutation after serving requests are unsupported. Manager 是用于自定义集成的受支持底层 API，应从构建完成的 Auth 获取；不支持零值构造，开始处理请求后不得修改配置或组件。
	Manager = engine.Manager
	// ManagerOption configures engine assembly. ManagerOption 配置引擎装配。
	ManagerOption = engine.Option
	// ComponentOwnership declares component shutdown ownership. ComponentOwnership 声明组件关闭所有权。
	ComponentOwnership = engine.ComponentOwnership
	// Strategy defines replaceable authentication algorithms. Strategy 定义可替换认证算法。
	Strategy = engine.Strategy
	// ManagerLoginOptions describes a low-level login request. ManagerLoginOptions 描述底层登录请求。
	ManagerLoginOptions = engine.LoginOptions
	// ManagerRefreshTokenOptions describes low-level refresh-token issuance. ManagerRefreshTokenOptions 描述底层刷新令牌签发请求。
	ManagerRefreshTokenOptions = engine.RefreshTokenOptions
	// ManagerTerminateOptions describes a low-level terminal operation. ManagerTerminateOptions 描述底层终端操作。
	ManagerTerminateOptions = engine.TerminateOptions
	// TerminateAction selects terminal removal behavior. TerminateAction 选择终端移除行为。
	TerminateAction = engine.TerminateAction
	// TokenState describes a stored token lifecycle state. TokenState 描述存储中的令牌生命周期状态。
	TokenState = engine.TokenState
	// TokenIntrospection describes token validation results. TokenIntrospection 描述令牌校验结果。
	TokenIntrospection = engine.TokenIntrospection
	// RefreshTokenPair contains access and refresh credentials. RefreshTokenPair 包含访问令牌与刷新凭证。
	RefreshTokenPair = engine.RefreshTokenPair
	// RefreshTokenInfo describes refresh credential metadata. RefreshTokenInfo 描述刷新凭证元数据。
	RefreshTokenInfo = engine.RefreshTokenInfo
	// TerminalRemovalFunc selects terminals to remove. TerminalRemovalFunc 选择待移除终端。
	TerminalRemovalFunc = engine.TerminalRemovalFunc
	// TerminalVisitor visits active terminals. TerminalVisitor 遍历活跃终端。
	TerminalVisitor = engine.TerminalVisitor
	// GeneratorFactory creates token generators. GeneratorFactory 创建令牌生成器。
	GeneratorFactory = builder.GeneratorFactory
	// StorageFactory creates storage adapters. StorageFactory 创建存储适配器。
	StorageFactory = builder.StorageFactory
	// CodecFactory creates codecs. CodecFactory 创建编解码器。
	CodecFactory = builder.CodecFactory
	// LogFactory creates log adapters. LogFactory 创建日志适配器。
	LogFactory = builder.LogFactory
	// PoolFactory creates task pools. PoolFactory 创建任务池。
	PoolFactory = builder.PoolFactory
	// Components groups explicit runtime components. Components 聚合显式运行时组件。
	Components = builder.Components
	// ComponentFactories groups runtime component factories. ComponentFactories 聚合运行时组件工厂。
	ComponentFactories = builder.ComponentFactories
)

// Terminal operation constants select the engine's termination behavior. 终端操作常量选择引擎的下线行为。
const (
	// TerminateActionLogout removes a terminal credential. TerminateActionLogout 移除终端凭证。
	TerminateActionLogout = engine.TerminateActionLogout
	// TerminateActionKickout marks a terminal as kicked out. TerminateActionKickout 标记终端被踢下线。
	TerminateActionKickout = engine.TerminateActionKickout
	// TerminateActionReplace marks a terminal as replaced. TerminateActionReplace 标记终端被顶下线。
	TerminateActionReplace = engine.TerminateActionReplace
	// TokenStateLogout reports an explicitly logged-out token. TokenStateLogout 表示主动登出的令牌。
	TokenStateLogout = engine.TokenStateLogout
	// TokenStateKickOut reports a kicked-out token. TokenStateKickOut 表示被踢下线的令牌。
	TokenStateKickOut = engine.TokenStateKickOut
	// TokenStateReplaced reports a replaced token. TokenStateReplaced 表示被顶下线的令牌。
	TokenStateReplaced = engine.TokenStateReplaced
	// TokenStateActiveTimeout reports an inactive token. TokenStateActiveTimeout 表示不活跃超时的令牌。
	TokenStateActiveTimeout = engine.TokenStateActiveTimeout
)

// DefaultStrategy returns the built-in authentication algorithms. DefaultStrategy 返回内置认证算法。
var DefaultStrategy = engine.DefaultStrategy

// WithStrategy selects custom authentication algorithms. WithStrategy 选择自定义认证算法。
var WithStrategy = engine.WithStrategy

// WithComponentOwnership selects which components the manager closes. WithComponentOwnership 选择由管理器关闭的组件。
var WithComponentOwnership = engine.WithComponentOwnership
