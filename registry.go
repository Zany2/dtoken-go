// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

var (
	// defaultAuth resolves the current default manager on each call and is never reassigned. defaultAuth 每次调用查找当前默认 Manager，初始化后不再修改。
	defaultAuth = Auth{authType: config.DefaultAuthType}

	globalManagerMap sync.Map // globalManagerMap stores managers by normalized auth type. globalManagerMap 按规范化认证类型存储管理器。

	globalManagerMu sync.Mutex // globalManagerMu serializes registry lifecycle writes. globalManagerMu 串行化注册表生命周期写操作。
)

// SetAuth registers an open Auth and closes the replaced engine; unavailable instances return an error. SetAuth 注册未关闭的 Auth 并关闭被替换引擎；实例不可用时返回错误。
func SetAuth(auth *Auth) error {
	return SetManager(auth.Manager())
}

// SetManager registers a valid open manager and closes the replaced engine; invalid managers return an error. SetManager 注册有效且未关闭的 Manager 并关闭被替换引擎；无效实例返回错误。
func SetManager(mgr *manager.Manager) error {
	if mgr == nil {
		return derror.ErrManagerNotFound
	}
	cfg := manager.RuntimeConfig(mgr)
	if cfg == nil {
		return derror.ErrManagerInvalidType
	}

	validAutoType := getAutoType(cfg.AuthType)

	globalManagerMu.Lock()
	if mgr.IsClosed() {
		globalManagerMu.Unlock()
		return derror.ErrManagerNotFound
	}

	previous, loaded := globalManagerMap.Swap(validAutoType, mgr)
	previousManager, validPrevious := previous.(*manager.Manager)
	if loaded && validPrevious && previousManager != mgr {
		// Reject reuse before releasing the registry lock; resource cleanup may wait outside it. 释放注册表锁前拒绝旧实例复用，资源清理在锁外等待。
		manager.BeginClose(previousManager)
	}
	globalManagerMu.Unlock()

	if loaded && validPrevious && previousManager != mgr {
		previousManager.CloseManager()
	}
	return nil
}

// GetManager retrieves an open manager by auth type or returns ErrManagerNotFound when missing or closed. GetManager 按认证类型获取未关闭的管理器，不存在或已关闭时返回 ErrManagerNotFound。
func GetManager(authType ...string) (*manager.Manager, error) {
	validAutoType := getAutoType(authType...)
	return loadManager(validAutoType)
}

// GetEventManager retrieves event manager by auth type. GetEventManager 根据认证类型获取事件监听管理器。
func GetEventManager(authType ...string) (*listener.Manager, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetEventManager(), nil
}

// DeleteManager deletes the manager for the specified auth type and releases resources. DeleteManager 删除指定认证类型的管理器并释放资源。
func DeleteManager(authType ...string) error {
	validAutoType := getAutoType(authType...)

	globalManagerMu.Lock()
	value, loaded := globalManagerMap.LoadAndDelete(validAutoType)
	if !loaded {
		globalManagerMu.Unlock()
		return derror.ErrManagerNotFound
	}
	mgr, ok := value.(*manager.Manager)
	if !ok {
		globalManagerMu.Unlock()
		return derror.ErrManagerInvalidType
	}
	manager.BeginClose(mgr)
	globalManagerMu.Unlock()

	mgr.CloseManager()
	return nil
}

// DeleteAllManager closes and deletes all managers in the global registry. DeleteAllManager 关闭并删除全局注册表中的全部管理器。
func DeleteAllManager() {
	managers := make([]*manager.Manager, 0)

	globalManagerMu.Lock()
	globalManagerMap.Range(func(key, value interface{}) bool {
		if mgr, ok := value.(*manager.Manager); ok {
			manager.BeginClose(mgr)
			managers = append(managers, mgr)
		}
		globalManagerMap.Delete(key)
		return true
	})
	globalManagerMu.Unlock()

	for _, mgr := range managers {
		mgr.CloseManager()
	}
}

// closeAndUnregisterManager removes the stable registry entry only if it still points to mgr, then closes it. closeAndUnregisterManager 仅移除仍指向 mgr 的固定注册项，再关闭该实例。
func closeAndUnregisterManager(mgr *manager.Manager) {
	if mgr == nil {
		return
	}

	globalManagerMu.Lock()
	manager.BeginClose(mgr)
	if cfg := manager.RuntimeConfig(mgr); cfg != nil {
		globalManagerMap.CompareAndDelete(getAutoType(cfg.AuthType), mgr)
	}
	globalManagerMu.Unlock()

	mgr.CloseManager()
}

// getAutoType normalizes auth type and falls back to the default auth type. getAutoType 规范化认证类型并在为空时使用默认类型。
func getAutoType(authType ...string) string {
	if len(authType) > 0 && strings.TrimSpace(authType[0]) != "" {
		trimmed := strings.TrimSpace(authType[0])
		if !strings.HasSuffix(trimmed, ":") {
			trimmed += ":"
		}
		return trimmed
	}
	return config.DefaultAuthType
}

// loadManager loads the manager for the normalized auth type. loadManager 加载已规范化认证类型对应的管理器。
func loadManager(authType string) (*manager.Manager, error) {
	value, ok := globalManagerMap.Load(authType)
	if !ok {
		return nil, derror.ErrManagerNotFound
	}
	mgr, ok := value.(*manager.Manager)
	if !ok || mgr == nil {
		return nil, derror.ErrManagerInvalidType
	}
	if mgr.IsClosed() {
		globalManagerMap.CompareAndDelete(authType, mgr)
		return nil, derror.ErrManagerNotFound
	}
	return mgr, nil
}

// registerManager publishes a newly built manager without replacing an existing instance. registerManager 发布新构建的管理器，不替换已有实例。
func registerManager(mgr *manager.Manager) error {
	globalManagerMu.Lock()
	defer globalManagerMu.Unlock()

	authType := getAutoType(manager.RuntimeConfig(mgr).AuthType)
	if _, err := loadManager(authType); err == nil {
		return fmt.Errorf("%w: %s", derror.ErrManagerAlreadyRegistered, authType)
	} else if err != derror.ErrManagerNotFound {
		return err
	}
	if mgr.IsClosed() {
		return derror.ErrManagerNotFound
	}
	globalManagerMap.Store(authType, mgr)
	return nil
}

// AuthType selects a registry namespace on every operation; an empty name selects the default. AuthType 每次操作查找指定注册体系，空名称选择默认体系。
func AuthType(name string) *Auth {
	return &Auth{authType: getAutoType(name)}
}

// Close removes and closes the current default instance; repeated calls are harmless. Close 注销并关闭当前默认实例，可重复调用。
func Close() {
	defaultAuth.Close()
}

// CloseAll unregisters and closes all current registered instances after application requests are drained. CloseAll 在应用排空请求后注销并关闭所有当前已注册实例。
func CloseAll() {
	DeleteAllManager()
}
