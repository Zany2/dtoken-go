// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"github.com/Zany2/dtoken-go/core/builder"
	"strings"
	"sync"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

var globalManagerMap sync.Map

// BuildAndSetManager overrides auth type before building and stores the manager in the global registry. BuildAndSetManager 在构建前覆盖认证类型并将管理器注册到全局注册表。
func BuildAndSetManager(b *builder.Builder, authType ...string) (*manager.Manager, error) {
	// Override auth type before build 在构建前覆盖认证类型。
	if len(authType) > 0 && strings.TrimSpace(authType[0]) != "" {
		b.AuthType(authType[0])
	}

	// Build manager with final config 使用最终配置构建管理器。
	mgr, err := b.Build()
	if err != nil {
		return nil, err
	}

	// Store manager by its final auth type 按最终认证类型注册管理器。
	SetManager(mgr)
	return mgr, nil
}

// SetManager stores a manager in the global registry. SetManager 将管理器存入全局注册表。
func SetManager(mgr *manager.Manager) {
	validAutoType := getAutoType(mgr.GetConfig().AuthType)
	globalManagerMap.Store(validAutoType, mgr)
}

// GetManager retrieves a manager from the global registry by auth type. GetManager 根据认证类型从全局注册表获取管理器。
func GetManager(authType ...string) (*manager.Manager, error) {
	validAutoType := getAutoType(authType...)
	return loadManager(validAutoType)
}

// DeleteManager deletes the manager for the specified auth type and releases resources. DeleteManager 删除指定认证类型的管理器并释放资源。
func DeleteManager(authType ...string) error {
	validAutoType := getAutoType(authType...)
	mgr, err := loadManager(validAutoType)
	if err != nil {
		return err
	}
	mgr.CloseManager()
	globalManagerMap.Delete(validAutoType)
	return nil
}

// DeleteAllManager closes and deletes all managers in the global registry. DeleteAllManager 关闭并删除全局注册表中的全部管理器。
func DeleteAllManager() {
	globalManagerMap.Range(func(key, value interface{}) bool {
		if mgr, ok := value.(*manager.Manager); ok {
			mgr.CloseManager()
		}
		globalManagerMap.Delete(key)
		return true
	})
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

// parseDeviceAndAuthType parses optional legacy parameters: device, deviceId, authType. parseDeviceAndAuthType 解析旧版可选参数：device、deviceId、authType。
func parseDeviceAndAuthType(params ...string) (device, deviceId, authType string) {
	if len(params) > 0 {
		device = params[0]
	}
	if len(params) > 1 {
		deviceId = params[1]
	}
	if len(params) > 2 {
		authType = params[2]
	}
	return
}
