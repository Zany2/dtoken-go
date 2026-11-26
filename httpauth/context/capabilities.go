package context

import (
	"reflect"

	"github.com/Zany2/dtoken-go/contract/nonce"
	"github.com/Zany2/dtoken-go/contract/oauth2"
	"github.com/Zany2/dtoken-go/contract/shortkey"
	"github.com/Zany2/dtoken-go/contract/ticket"
	derror "github.com/Zany2/dtoken-go/errors"
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
		if isNilBinding(c.capabilities.Nonce) {
			c.capabilities.Nonce = nil
		}
		if isNilBinding(c.capabilities.OAuth2) {
			c.capabilities.OAuth2 = nil
		}
		if isNilBinding(c.capabilities.Ticket) {
			c.capabilities.Ticket = nil
		}
		if isNilBinding(c.capabilities.ShortKey) {
			c.capabilities.ShortKey = nil
		}
	}
	return c
}

func (c *DTokenContext) nonceService() (nonce.Service, error) {
	if c == nil {
		return nil, derror.ErrManagerNotFound
	}
	return c.capabilities.Nonce, c.requireCapability(c.capabilities.Nonce)
}

func (c *DTokenContext) oauth2Service() (oauth2.Service, error) {
	if c == nil {
		return nil, derror.ErrManagerNotFound
	}
	return c.capabilities.OAuth2, c.requireCapability(c.capabilities.OAuth2)
}

func (c *DTokenContext) ticketService() (ticket.Service, error) {
	if c == nil {
		return nil, derror.ErrManagerNotFound
	}
	return c.capabilities.Ticket, c.requireCapability(c.capabilities.Ticket)
}

func (c *DTokenContext) shortKeyService() (shortkey.Service, error) {
	if c == nil {
		return nil, derror.ErrManagerNotFound
	}
	return c.capabilities.ShortKey, c.requireCapability(c.capabilities.ShortKey)
}

// requireCapability preserves standalone services but rejects operations tied to a closed engine. requireCapability 保留独立服务用法，但拒绝绑定已关闭引擎的操作。
func (c *DTokenContext) requireCapability(service any) error {
	if c.manager != nil && c.manager.IsClosed() {
		return derror.ErrManagerNotFound
	}
	if service == nil {
		return derror.ErrModuleNotEnabled
	}
	return nil
}

// isNilBinding detects absent bindings without invoking methods on typed nil values. isNilBinding 检测缺失绑定，避免调用带类型空值的方法。
func isNilBinding(service any) bool {
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
