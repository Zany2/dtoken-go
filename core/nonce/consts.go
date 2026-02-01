// @Author daixk 2025/12/11 22:20:00
package security

import (
	"time"
)

// Nonce常量
const (
	DefaultNonceTTL = 5 * time.Minute // 默认nonce过期时间
	NonceLength     = 32              // Nonce字节长度
	NonceKeySuffix  = "nonce:"        // 前缀后的键后缀
)

// 刷新令牌常量
const (
	DefaultRefreshTTL  = 30 * 24 * time.Hour // 30天
	DefaultAccessTTL   = 2 * time.Hour       // 2小时
	RefreshTokenLength = 32                  // 刷新令牌字节长度
	RefreshKeySuffix   = "refresh:"          // 前缀后的键后缀
)
