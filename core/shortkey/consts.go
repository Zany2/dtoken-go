// @Author daixk 2026/06/01
package shortkey

import "time"

const (
	// DefaultTTL stores default short key ttl. DefaultTTL 存储默认短 Key 有效期。
	DefaultTTL = 5 * time.Minute
	// DefaultLength stores default generated key length. DefaultLength 存储默认短 Key 长度。
	DefaultLength = 8
	// KeySuffix stores short key storage suffix. KeySuffix 存储短 Key 存储键后缀。
	KeySuffix = "shortkey:"
)

const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Status defines short key lifecycle state. Status 定义短 Key 生命周期状态。
type Status string

const (
	// StatusPending indicates the key is waiting for confirmation. StatusPending 表示短 Key 等待确认。
	StatusPending Status = "pending"
	// StatusConfirmed indicates the key has been confirmed and can be consumed. StatusConfirmed 表示短 Key 已确认且可消费。
	StatusConfirmed Status = "confirmed"
	// StatusConsumed indicates the key has been consumed. StatusConsumed 表示短 Key 已消费。
	StatusConsumed Status = "consumed"
	// StatusRevoked indicates the key has been revoked. StatusRevoked 表示短 Key 已撤销。
	StatusRevoked Status = "revoked"
	// StatusExpired indicates the key has expired. StatusExpired 表示短 Key 已过期。
	StatusExpired Status = "expired"
	// StatusInvalid indicates the key is missing or malformed. StatusInvalid 表示短 Key 无效或不存在。
	StatusInvalid Status = "invalid"
)
