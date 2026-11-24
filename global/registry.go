// @Author daixk 2025/12/22 15:56:00
package global

import (
	"strings"

	"github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/config"
	listener "github.com/Zany2/dtoken-go/event"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// BuildAndSetManager overrides auth type on supported builders and stores the manager in the global registry. BuildAndSetManager 为支持的 Builder 覆盖认证类型并将管理器注册到全局注册表。
func BuildAndSetManager(b any, authType ...string) (*manager.Manager, error) {
	return dtoken.BuildAndSetManager(b, authType...)
}

// SetAuth registers an open Auth without cloning its shared engine and closes that engine on replacement or deletion; nil and closed instances are ignored. SetAuth 注册未关闭的 Auth，与已有门面共享引擎并在替换或删除时关闭引擎；忽略空实例和已关闭实例。
func SetAuth(auth *Auth) {
	dtoken.SetAuth(auth)
}

// SetManager stores a valid open manager and closes the manager it replaces; invalid or closed managers are ignored. SetManager 存入有效且未关闭的 Manager，并关闭被替换实例；无效或已关闭的 Manager 会被忽略。
func SetManager(mgr *manager.Manager) {
	dtoken.SetManager(mgr)
}

// GetManager retrieves an open manager by auth type or returns ErrManagerNotFound when missing or closed. GetManager 按认证类型获取未关闭的管理器，不存在或已关闭时返回 ErrManagerNotFound。
func GetManager(authType ...string) (*manager.Manager, error) {
	return dtoken.GetManager(authType...)
}

// GetEventManager retrieves event manager by auth type. GetEventManager 根据认证类型获取事件监听管理器。
func GetEventManager(authType ...string) (*listener.Manager, error) {
	return dtoken.GetEventManager(authType...)
}

// DeleteManager deletes the manager for the specified auth type and releases resources. DeleteManager 删除指定认证类型的管理器并释放资源。
func DeleteManager(authType ...string) error {
	return dtoken.DeleteManager(authType...)
}

// DeleteAllManager closes and deletes all managers in the global registry. DeleteAllManager 关闭并删除全局注册表中的全部管理器。
func DeleteAllManager() {
	dtoken.DeleteAllManager()
}

// getManagerAuto resolves the auth type string (empty falls back to default). getManagerAuto 解析认证类型字符串，为空时回退到默认。
func getManagerAuto(authType string) (*manager.Manager, error) {
	if strings.TrimSpace(authType) != "" {
		return GetManager(authType)
	}
	return GetManager()
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
