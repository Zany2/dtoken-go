package sso

// Deployment describes session topology independently of the credential Mode. Deployment 描述会话部署结构，与凭证 Mode 相互独立。
type Deployment string

const (
	// DeploymentSharedCookie shares both browser cookies and authentication storage. DeploymentSharedCookie 同时共享浏览器 Cookie 与认证存储。
	DeploymentSharedCookie Deployment = "shared-cookie"
	// DeploymentSharedStore uses separate cookies with shared authentication and protocol storage. DeploymentSharedStore 使用独立 Cookie，共享认证及协议存储。
	DeploymentSharedStore Deployment = "shared-store"
	// DeploymentHTTP exchanges over HTTP and maintains independent local authentication sessions. DeploymentHTTP 通过 HTTP 兑换并维护独立本地认证会话。
	DeploymentHTTP Deployment = "http"
)

// DeploymentConfig validates planned standard session wiring; it does not install browser handlers. DeploymentConfig 校验标准会话装配配置，不安装浏览器处理器。
type DeploymentConfig struct {
	// Deployment selects one of the three standard session topologies. Deployment 选择三种标准会话部署结构之一。
	Deployment Deployment
	// Client supplies application identity and exchange dependencies. Client 提供应用身份及兑换依赖。
	Client ClientConfig
	// Sessions supplies the application's session backend, borrowed without ownership. Sessions 提供应用会话后端，仅借用且不接管所有权。
	Sessions SessionBackend
}

// Validate checks visible wiring only; it cannot prove shared storage, cookies, or a remote center's capabilities. Validate 仅检查可见装配，无法证明存储或 Cookie 共享及远端中心能力。
func (c DeploymentConfig) Validate() error {
	client := NewClientApp(c.Client).config
	switch c.Deployment {
	case DeploymentSharedCookie:
		if client.ExchangeService != nil || client.SharedStoreServer != nil || client.UseHTTPCheck {
			return configError("Deployment", "shared-cookie does not exchange credentials", nil)
		}
	case DeploymentSharedStore:
		if client.UseHTTPCheck || client.ExchangeService == nil && client.SharedStoreServer == nil {
			return configError("Deployment", "shared-store requires local exchange without UseHTTPCheck", nil)
		}
	case DeploymentHTTP:
		if client.ExchangeService != nil || client.SharedStoreServer != nil {
			return configError("Deployment", "HTTP deployment cannot configure local exchange", nil)
		}
	default:
		return configError("Deployment", "select a supported deployment explicitly", ErrModeUnsupported)
	}
	if client.Mode != ModeTicket {
		return configError("Deployment.Client.Mode", "standard deployment requires Ticket mode", ErrModeUnsupported)
	}
	if c.Sessions == nil {
		return configError("Deployment.Sessions", "a session backend is required", nil)
	}
	return client.Validate()
}
