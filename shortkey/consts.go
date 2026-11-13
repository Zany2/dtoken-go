// @Author daixk 2026/06/01
package shortkey

import (
	"time"

	capability "github.com/Zany2/dtoken-go/contract/shortkey"
)

const (
	// DefaultTTL stores default short key ttl. DefaultTTL 存储默认短 Key 有效期。
	DefaultTTL = 5 * time.Minute
	// DefaultLength stores default generated key length. DefaultLength 存储默认短 Key 长度。
	DefaultLength = 8
	// KeySuffix stores short key storage suffix. KeySuffix 存储短 Key 存储键后缀。
	KeySuffix = "shortkey:"
)

const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Status aliases the public capability state. Status 引用公开能力状态。
type Status = capability.Status

const (
	// StatusPending identifies the capability state. StatusPending 标识能力状态。
	StatusPending = capability.StatusPending
	// StatusConfirmed identifies the capability state. StatusConfirmed 标识能力状态。
	StatusConfirmed = capability.StatusConfirmed
	// StatusConsumed identifies the capability state. StatusConsumed 标识能力状态。
	StatusConsumed = capability.StatusConsumed
	// StatusRevoked identifies the capability state. StatusRevoked 标识能力状态。
	StatusRevoked = capability.StatusRevoked
	// StatusExpired identifies the capability state. StatusExpired 标识能力状态。
	StatusExpired = capability.StatusExpired
	// StatusInvalid identifies the capability state. StatusInvalid 标识能力状态。
	StatusInvalid = capability.StatusInvalid
)
