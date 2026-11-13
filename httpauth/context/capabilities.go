package context

import (
	"reflect"

	"github.com/Zany2/dtoken-go/contract/nonce"
	"github.com/Zany2/dtoken-go/contract/oauth2"
	"github.com/Zany2/dtoken-go/contract/shortkey"
	"github.com/Zany2/dtoken-go/contract/ticket"
)

// Capabilities groups explicitly composed services for a request's authentication instance. Capabilities 聚合为请求认证实例显式组合的服务。
type Capabilities struct {
	// Nonce handles replay protection. Nonce 处理防重放凭证。
	Nonce nonce.Service
	// OAuth2 handles authorization flows. OAuth2 处理授权流程。
	OAuth2 oauth2.Service
	// Ticket handles temporary credentials. Ticket 处理临时凭证。
	Ticket ticket.Service
	// ShortKey handles confirmation credentials. ShortKey 处理确认凭证。
	ShortKey shortkey.Service
}

// WithCapabilities copies explicit service bindings and preserves them when no override is supplied. WithCapabilities 复制显式服务绑定，未提供覆盖值时保留现有绑定。
func (c *DTokenContext) WithCapabilities(capabilities *Capabilities) *DTokenContext {
	if capabilities != nil {
		c.capabilities = *capabilities

		// Normalize typed nils introduced by interface bindings. 归一化接口绑定中的带类型空值。
		if isNilCapability(c.capabilities.Nonce) {
			c.capabilities.Nonce = nil
		}
		if isNilCapability(c.capabilities.OAuth2) {
			c.capabilities.OAuth2 = nil
		}
		if isNilCapability(c.capabilities.Ticket) {
			c.capabilities.Ticket = nil
		}
		if isNilCapability(c.capabilities.ShortKey) {
			c.capabilities.ShortKey = nil
		}
	}
	return c
}

func (c *DTokenContext) nonceService() nonce.Service {
	return c.capabilities.Nonce
}

func (c *DTokenContext) oauth2Service() oauth2.Service {
	return c.capabilities.OAuth2
}

func (c *DTokenContext) ticketService() ticket.Service {
	return c.capabilities.Ticket
}

func (c *DTokenContext) shortKeyService() shortkey.Service {
	return c.capabilities.ShortKey
}

// isNilCapability detects absent services without invoking methods on typed nil values. isNilCapability 检测缺失服务，避免调用带类型空值的方法。
func isNilCapability(service any) bool {
	if service == nil {
		return true
	}
	value := reflect.ValueOf(service)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
