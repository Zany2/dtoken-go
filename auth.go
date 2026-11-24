// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"sync/atomic"

	"github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// Auth wraps a fixed Manager or resolves a registry namespace through AuthType. Auth 包装固定 Manager，或通过 AuthType 动态查找注册体系。
type Auth struct {
	authType string                          // authType selects a registry namespace when nonempty. authType 非空时指定动态查找的注册体系。
	manager  atomic.Pointer[manager.Manager] // manager stores the underlying auth manager. manager 存储底层鉴权管理器。
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
	if a.authType != "" {
		mgr, _ := GetManager(a.authType)
		return mgr
	}
	return a.manager.Load()
}

// IsClosed reports whether the fixed or currently selected engine is unavailable. IsClosed 判断固定引擎或当前选择的引擎是否不可用。
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

// Close unregisters and closes the shared engine; selectors close the currently registered instance. Drain requests first. Close 注销并关闭共享引擎；选择器关闭当前注册实例，调用前应排空请求。
func (a *Auth) Close() {
	if a == nil {
		return
	}

	if a.authType != "" {
		_ = DeleteManager(a.authType)
		return
	}

	mgr := a.manager.Swap(nil)
	if mgr == nil {
		return
	}

	closeAndUnregisterManager(mgr)
}

// requireManager returns the underlying manager or an explicit error. requireManager 返回底层管理器或明确错误。
func (a *Auth) requireManager() (*manager.Manager, error) {
	if a != nil && a.authType != "" {
		return GetManager(a.authType)
	}
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
