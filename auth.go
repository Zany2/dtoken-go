// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"sync/atomic"

	"github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// Auth is an instance-oriented facade over Manager. Auth 是基于 Manager 的实例化门面。
type Auth struct {
	manager atomic.Pointer[manager.Manager] // manager stores the underlying auth manager. manager 存储底层鉴权管理器。
}

// New wraps an existing engine and shares its lifetime; it does not clone resources. New 包装现有引擎并共享其生命周期，不复制资源。
func New(mgr *Manager) *Auth {
	auth := &Auth{}
	auth.manager.Store(mgr)
	return auth
}

// Manager exposes the shared engine for custom integrations; applications should prefer Auth methods or WithAuth. Manager 暴露用于自定义集成的共享引擎，应用优先使用 Auth 方法或 WithAuth。
func (a *Auth) Manager() *Manager {
	if a == nil {
		return nil
	}
	return a.manager.Load()
}

// IsClosed reports whether this facade is detached or its shared engine is closed. IsClosed 判断门面已分离或共享引擎已关闭。
func (a *Auth) IsClosed() bool {
	mgr := a.Manager()
	return mgr == nil || mgr.IsClosed()
}

// EventManager returns the event manager, or nil when the auth manager is unavailable. EventManager 返回事件监听管理器，鉴权管理器不可用时返回 nil。
func (a *Auth) EventManager() *listener.Manager {
	mgr, err := a.requireManager()
	if err != nil {
		return nil
	}
	return mgr.GetEventManager()
}

// Close detaches this facade and closes the shared engine, invalidating other wrappers and middleware; drain requests before releasing shared resources. Close 分离当前门面并关闭共享引擎，使其他门面和中间件不可用；释放共享资源前须排空请求。
func (a *Auth) Close() {
	if a == nil {
		return
	}

	mgr := a.manager.Swap(nil)
	if mgr == nil {
		return
	}

	mgr.CloseManager()
}

// requireManager returns the underlying manager or an explicit error. requireManager 返回底层管理器或明确错误。
func (a *Auth) requireManager() (*manager.Manager, error) {
	mgr := a.Manager()
	if mgr == nil || mgr.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}
	return mgr, nil
}

// Emit publishes an extension event through this instance lifecycle. Emit 通过当前实例生命周期发布扩展事件。
func (a *Auth) Emit(event listener.Event, loginID, device, deviceID, token string, extra map[string]any) {
	mgr, err := a.requireManager()
	if err != nil {
		return
	}
	mgr.Emit(event, loginID, device, deviceID, token, extra)
}

// Config returns an isolated configuration snapshot. Config 返回独立的配置快照。
func (a *Auth) Config() *Config {
	m, err := a.requireManager()
	if err != nil {
		return nil
	}
	return m.GetConfig().Clone()
}

// Storage borrows the configured storage while Auth is open; callers must not close it or use it beyond its owner's lifetime. Storage 在 Auth 可用时借用其存储，调用方不得关闭它或超出其所有者的生命周期使用。
func (a *Auth) Storage() contract.Storage {
	m, err := a.requireManager()
	if err != nil {
		return nil
	}
	return m.GetStorage()
}

// Codec returns the configured codec for composing capabilities. Codec 返回用于组合独立能力的编解码器。
func (a *Auth) Codec() contract.Codec {
	m, err := a.requireManager()
	if err != nil {
		return nil
	}
	return m.GetSerializer()
}
