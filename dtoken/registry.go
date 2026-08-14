// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/manager"
)

var (
	// globalManagerMap stores managers by normalized auth type. globalManagerMap 按规范化认证类型存储管理器。
	globalManagerMap sync.Map

	// globalManagerMu serializes registry lifecycle writes. globalManagerMu 串行化注册表生命周期写操作。
	globalManagerMu sync.Mutex

	// globalRetiringManagers prevents managers being registered again while their resources are closing. globalRetiringManagers 防止资源关闭期间重新注册 Manager。
	globalRetiringManagers = make(map[*manager.Manager]struct{})
)

// managerBuilder abstracts builders supported by global registration. managerBuilder 抽象全局注册支持的构建器。
type managerBuilder interface {
	// Build creates a manager from the current builder state. Build 根据当前构建器状态创建管理器。
	Build() (*manager.Manager, error)
}

// BuildAndSetManager overrides auth type on supported builders and stores the manager in the global registry. BuildAndSetManager 为支持的 Builder 覆盖认证类型并将管理器注册到全局注册表。
func BuildAndSetManager(b managerBuilder, authType ...string) (*manager.Manager, error) {
	if b == nil {
		return nil, derror.ErrManagerNotFound
	}

	// Reject typed nil custom builders before invoking their methods. 调用方法前拒绝带具体类型的空自定义 Builder。
	builderValue := reflect.ValueOf(b)
	switch builderValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if builderValue.IsNil() {
			return nil, derror.ErrManagerNotFound
		}
	}

	// Override auth type before build 在构建前覆盖认证类型。
	if len(authType) > 0 && strings.TrimSpace(authType[0]) != "" {
		switch typed := b.(type) {
		case *Builder:
			typed.AuthType(authType[0])
		case *builder.Builder:
			typed.AuthType(authType[0])
		default:
			return nil, fmt.Errorf("%w: auth type override is unsupported by builder %T", derror.ErrInvalidParam, b)
		}
	}

	// Build manager with final config 使用最终配置构建管理器。
	mgr, err := b.Build()
	if err != nil {
		return nil, err
	}
	if mgr == nil || mgr.GetConfig() == nil || mgr.IsClosed() {
		return nil, fmt.Errorf("%w: builder returned an invalid manager", derror.ErrManagerInvalidType)
	}

	// Store manager by its final auth type 按最终认证类型注册管理器。
	SetManager(mgr)
	return mgr, nil
}

// SetManager stores a valid open manager and closes the manager it replaces; invalid or closed managers are ignored. SetManager 存入有效且未关闭的 Manager，并关闭被替换实例；无效或已关闭的 Manager 会被忽略。
func SetManager(mgr *manager.Manager) {
	if mgr == nil || mgr.GetConfig() == nil {
		return
	}

	validAutoType := getAutoType(mgr.GetConfig().AuthType)

	globalManagerMu.Lock()
	if mgr.IsClosed() {
		globalManagerMu.Unlock()
		return
	}
	if _, retiring := globalRetiringManagers[mgr]; retiring {
		globalManagerMu.Unlock()
		return
	}

	// Keep one stable registry key per manager instance. 每个 Manager 实例仅保留一个稳定注册键。
	globalManagerMap.Range(func(key, value interface{}) bool {
		registered, ok := value.(*manager.Manager)
		if ok && registered == mgr && key != validAutoType {
			globalManagerMap.Delete(key)
		}
		return true
	})

	previous, loaded := globalManagerMap.Swap(validAutoType, mgr)
	previousManager, validPrevious := previous.(*manager.Manager)
	if loaded && validPrevious && previousManager != mgr {
		globalRetiringManagers[previousManager] = struct{}{}
	}
	globalManagerMu.Unlock()

	if !loaded || !validPrevious || previousManager == mgr {
		return
	}
	closeRetiringManager(previousManager)
}

// GetManager retrieves a manager from the global registry by auth type. GetManager 根据认证类型从全局注册表获取管理器。
func GetManager(authType ...string) (*manager.Manager, error) {
	validAutoType := getAutoType(authType...)
	return loadManager(validAutoType)
}

// getManagerAuto resolves the auth type string (empty falls back to default). getManagerAuto 解析认证类型字符串，为空时回退到默认。
func getManagerAuto(authType string) (*manager.Manager, error) {
	if strings.TrimSpace(authType) != "" {
		return GetManager(authType)
	}
	return GetManager()
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
	globalRetiringManagers[mgr] = struct{}{}
	globalManagerMu.Unlock()

	closeRetiringManager(mgr)
	return nil
}

// DeleteAllManager closes and deletes all managers in the global registry. DeleteAllManager 关闭并删除全局注册表中的全部管理器。
func DeleteAllManager() {
	managers := make([]*manager.Manager, 0)
	seen := make(map[*manager.Manager]struct{})

	globalManagerMu.Lock()
	globalManagerMap.Range(func(key, value interface{}) bool {
		if mgr, ok := value.(*manager.Manager); ok {
			if _, exists := seen[mgr]; !exists {
				seen[mgr] = struct{}{}
				globalRetiringManagers[mgr] = struct{}{}
				managers = append(managers, mgr)
			}
		}
		globalManagerMap.Delete(key)
		return true
	})
	globalManagerMu.Unlock()

	for _, mgr := range managers {
		closeRetiringManager(mgr)
	}
}

// closeAndUnregisterManager removes every registry entry that still points to mgr and closes it. closeAndUnregisterManager 移除仍指向 mgr 的全部注册项并将其关闭。
func closeAndUnregisterManager(mgr *manager.Manager) {
	if mgr == nil {
		return
	}

	globalManagerMu.Lock()
	globalRetiringManagers[mgr] = struct{}{}
	globalManagerMap.Range(func(key, value interface{}) bool {
		registered, ok := value.(*manager.Manager)
		if ok && registered == mgr {
			globalManagerMap.Delete(key)
		}
		return true
	})
	globalManagerMu.Unlock()

	closeRetiringManager(mgr)
}

// closeRetiringManager closes mgr outside the registry lock and clears its transient retirement marker. closeRetiringManager 在注册表锁外关闭 mgr，并清除临时退役标记。
func closeRetiringManager(mgr *manager.Manager) {
	if mgr == nil {
		return
	}

	defer func() {
		globalManagerMu.Lock()
		delete(globalRetiringManagers, mgr)
		globalManagerMu.Unlock()
	}()
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
	if !ok {
		return nil, derror.ErrManagerInvalidType
	}
	return mgr, nil
}

// parseDeviceAndAuthType parses optional legacy parameters: device, deviceID, authType. parseDeviceAndAuthType 解析旧版可选参数：device、deviceId、authType。
func parseDeviceAndAuthType(params ...string) (device, deviceID, authType string) {
	if len(params) > 0 {
		device = params[0]
	}
	if len(params) > 1 {
		deviceID = params[1]
	}
	if len(params) > 2 {
		authType = params[2]
	}
	return
}
