package dtoken

import "github.com/Zany2/dtoken-go/config"

// Config describes authentication behavior. Config 描述认证行为。
type Config = config.Config

// DefaultConfig creates the default authentication configuration. DefaultConfig 创建默认认证配置。
func DefaultConfig() *Config { return config.DefaultConfig() }
