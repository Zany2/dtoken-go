// @Author daixk 2026/05/15
package dtoken

import (
	"testing"

	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// TestNewBuilderReturnsDefaultBuilder verifies the facade exposes a ready builder TestNewBuilderReturnsDefaultBuilder 验证门面入口会返回可用的 Builder
func TestNewBuilderReturnsDefaultBuilder(t *testing.T) {
	if NewBuilder() == nil {
		t.Fatal("NewBuilder() returned nil")
	}
}

// TestZeroValueBuilderInitializesDefaults verifies a zero-value builder lazily initializes every config. TestZeroValueBuilderInitializesDefaults 验证零值 Builder 会惰性初始化全部配置。

// TestBuilderNilConfigsRestoreDefaults verifies nil config setters restore independent defaults. TestBuilderNilConfigsRestoreDefaults 验证传入 nil 会恢复独立的默认配置。

// TestBuilderBuildsCoreOnlyByDefault verifies optional modules are opt-in TestBuilderBuildsCoreOnlyByDefault 验证默认只装配核心能力。

// TestBuilderBuildsWithModuleConfig verifies high-level module config chain TestBuilderBuildsWithModuleConfig 验证高层模块配置链路

// TestBuilderRejectsInvalidModuleConfig verifies module validation runs before assembly TestBuilderRejectsInvalidModuleConfig 验证装配前会执行模块校验

// TestBuilderKeepsEnabledModulesWithExtraOption verifies generic options do not disable enabled modules TestBuilderKeepsEnabledModulesWithExtraOption 验证通用选项不会关闭已启用模块

// TestBuilderAppliesUserOptionsAfterEnabledModules verifies user options can still override enabled defaults TestBuilderAppliesUserOptionsAfterEnabledModules 验证用户选项仍可覆盖已启用模块

// mustBuildManager constructs the internal engine for builder contract assertions. mustBuildManager 为构建器契约断言创建内部引擎。
func (b *Builder) mustBuildManager() *manager.Manager {
	mgr, err := b.buildManager()
	if err != nil {
		panic(err)
	}
	return mgr
}
