// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/manager"
)

// Auth is an instance-oriented facade over Manager. Auth 是基于 Manager 的实例化门面。
type Auth struct {
	manager *manager.Manager // manager stores the underlying auth manager. manager 存储底层鉴权管理器。
}

// New creates an instance-oriented auth facade. New 创建实例化鉴权门面。
func New(mgr *manager.Manager) *Auth {
	return &Auth{manager: mgr}
}

// Manager returns the underlying manager. Manager 返回底层管理器。
func (a *Auth) Manager() *manager.Manager {
	if a == nil {
		return nil
	}
	return a.manager
}

// EventManager returns the underlying event manager. EventManager 返回底层事件监听管理器。
func (a *Auth) EventManager() *listener.Manager {
	if a == nil || a.manager == nil {
		return nil
	}
	return a.manager.GetEventManager()
}

// Close releases resources held by the underlying manager. Close 释放底层管理器持有的资源。
func (a *Auth) Close() {
	if a == nil || a.manager == nil {
		return
	}
	a.manager.CloseManager()
}

// requireManager returns the underlying manager or an explicit error. requireManager 返回底层管理器或明确错误。
func (a *Auth) requireManager() (*manager.Manager, error) {
	if a == nil || a.manager == nil {
		return nil, derror.ErrManagerNotFound
	}
	return a.manager, nil
}
