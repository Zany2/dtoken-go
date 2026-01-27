// @Author daixk 2026/1/22 13:38:00
package errors

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

	// ErrManagerNotFound 指定认证类型的 Manager 实例未找到
	ErrManagerNotFound = errors.New("auth manager not found")
)

// ---------------------------- 账号错误 ----------------------------

var (
	// ErrIDIsEmpty ID为空
	ErrIDIsEmpty = errors.New("ID is required and cannot be empty")

	// ErrAccountDisabled 账号已被禁用
	ErrAccountDisabled = errors.New("account disabled: this account has been temporarily or permanently disabled")

	// ErrLoginLimitExceeded 超出最大登录数量限制
	ErrLoginLimitExceeded = errors.New("account error: login count exceeds the maximum limit")
)

var (
	// ErrInvalidToken 令牌无效
	ErrInvalidToken = errors.New("invalid or malformed authentication token")

	// ErrTokenKickout Token已被踢下线
	ErrTokenKickout = fmt.Errorf("authentication required: token has been kicked out")

	// ErrTokenReplaced Token已被顶下线
	ErrTokenReplaced = fmt.Errorf("authentication required: token has been replaced")
)
