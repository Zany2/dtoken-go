// @Author daixk 2026/1/22 13:38:00
package derror

import (
	"errors"
	"fmt"
)

// ---------------------------- 系统错误 ----------------------------

var (
	// ErrStorageUnavailable 存储后端不可用
	ErrStorageUnavailable = errors.New("storage unavailable: unable to connect to storage backend")

	// ErrSerializeFailed 序列化失败
	ErrSerializeFailed = errors.New("serialize failed: unable to encode data")

	// ErrTypeConvert 类型转换失败
	ErrTypeConvert = errors.New("type conversion failed: unable to convert value to target type")

	// ErrManagerNotFound 表示未找到指定认证类型的管理器
	ErrManagerNotFound = errors.New("manager not found")

	// ErrManagerInvalidType 表示全局存储中的管理器类型不正确
	ErrManagerInvalidType = errors.New("manager has invalid type")
)

// ---------------------------- 账号错误 ----------------------------

var (
	// ErrIDIsEmpty ID为空
	ErrIDIsEmpty = errors.New("ID is required and cannot be empty")

	// ErrAccountDisabled 账号已被禁用
	ErrAccountDisabled = errors.New("account disabled: this account has been temporarily or permanently disabled")

	// ErrAccountNotDisabled 账号未被封禁
	ErrAccountNotDisabled = errors.New("account not disabled: this account is not currently disabled")

	// ErrLoginLimitExceeded 超出最大登录数量限制
	ErrLoginLimitExceeded = errors.New("account error: login count exceeds the maximum limit")
)

// ---------------------------- 验证错误 ----------------------------

var (
	// ErrInvalidToken 令牌无效
	ErrInvalidToken = errors.New("invalid or malformed authentication token")

	// ErrTokenKickout Token已被踢下线
	ErrTokenKickout = fmt.Errorf("authentication required: token has been kicked out")

	// ErrTokenReplaced Token已被顶下线
	ErrTokenReplaced = fmt.Errorf("authentication required: token has been replaced")
)

// ---------------------------- session错误 ----------------------------
var (
	// ErrSessionNotFound 会话不存在
	ErrSessionNotFound = errors.New("session not found")
)

// ---------------------------- nonce错误 ----------------------------
var (
	// ErrInvalidNonce nonce无效
	ErrInvalidNonce = errors.New("invalid or expired nonce")
)
