// @Author daixk 2025/12/22 15:56:00
package kratos

import "github.com/Zany2/dtoken-go/core/derror"

// DTokenError exposes the core DToken error type.
type DTokenError = derror.DTokenError

// DToken error codes.
const (
	CodeSuccess          = derror.CodeSuccess
	CodeBadRequest       = derror.CodeBadRequest
	CodeNotLogin         = derror.CodeNotLogin
	CodePermissionDenied = derror.CodePermissionDenied
	CodeNotFound         = derror.CodeNotFound
	CodeServerError      = derror.CodeServerError
	CodeTokenInvalid     = derror.CodeTokenInvalid
	CodeTokenExpired     = derror.CodeTokenExpired
	CodeAccountDisabled  = derror.CodeAccountDisabled
	CodeKickedOut        = derror.CodeKickedOut
	CodeActiveTimeout    = derror.CodeActiveTimeout
	CodeMaxLoginCount    = derror.CodeMaxLoginCount
	CodeStorageError     = derror.CodeStorageError
	CodeInvalidParameter = derror.CodeInvalidParameter
)

// DToken error helpers and values.
var (
	NewDTokenError                  = derror.NewDTokenError
	ErrStorageUnavailable           = derror.ErrStorageUnavailable
	ErrKeyNotFound                  = derror.ErrKeyNotFound
	ErrSerializeFailed              = derror.ErrSerializeFailed
	ErrTypeConvert                  = derror.ErrTypeConvert
	ErrManagerNotFound              = derror.ErrManagerNotFound
	ErrManagerInvalidType           = derror.ErrManagerInvalidType
	ErrInvalidParam                 = derror.ErrInvalidParam
	ErrStorageCapabilityUnsupported = derror.ErrStorageCapabilityUnsupported
	ErrIDIsEmpty                    = derror.ErrIDIsEmpty
	ErrEmptyLoginID                 = derror.ErrEmptyLoginID
	ErrAccountDisabled              = derror.ErrAccountDisabled
	ErrAccountNotDisabled           = derror.ErrAccountNotDisabled
	ErrLoginLimitExceeded           = derror.ErrLoginLimitExceeded
	ErrNotLogin                     = derror.ErrNotLogin
	ErrInvalidToken                 = derror.ErrInvalidToken
	ErrTokenExpired                 = derror.ErrTokenExpired
	ErrActiveTimeout                = derror.ErrActiveTimeout
	ErrTokenKickout                 = derror.ErrTokenKickout
	ErrTokenReplaced                = derror.ErrTokenReplaced
	ErrInvalidDevice                = derror.ErrInvalidDevice
	ErrPermissionDenied             = derror.ErrPermissionDenied
	ErrRoleDenied                   = derror.ErrRoleDenied
	ErrServiceDisabled              = derror.ErrServiceDisabled
	ErrServiceNotDisabled           = derror.ErrServiceNotDisabled
	ErrDeviceDisabled               = derror.ErrDeviceDisabled
	ErrDeviceNotDisabled            = derror.ErrDeviceNotDisabled
	ErrDisableLevelNotReached       = derror.ErrDisableLevelNotReached
	ErrSessionNotFound              = derror.ErrSessionNotFound
	ErrInvalidNonce                 = derror.ErrInvalidNonce
	ErrClientOrClientIDEmpty        = derror.ErrClientOrClientIDEmpty
	ErrClientNotFound               = derror.ErrClientNotFound
	ErrInvalidClientCredentials     = derror.ErrInvalidClientCredentials
	ErrInvalidGrantType             = derror.ErrInvalidGrantType
	ErrInvalidRedirectURI           = derror.ErrInvalidRedirectURI
	ErrInvalidScope                 = derror.ErrInvalidScope
	ErrUserIDEmpty                  = derror.ErrUserIDEmpty
	ErrInvalidAuthCode              = derror.ErrInvalidAuthCode
	ErrAuthCodeUsed                 = derror.ErrAuthCodeUsed
	ErrAuthCodeExpired              = derror.ErrAuthCodeExpired
	ErrClientMismatch               = derror.ErrClientMismatch
	ErrRedirectURIMismatch          = derror.ErrRedirectURIMismatch
	ErrInvalidRefreshToken          = derror.ErrInvalidRefreshToken
	ErrInvalidAccessToken           = derror.ErrInvalidAccessToken
	ErrInvalidUserCredentials       = derror.ErrInvalidUserCredentials
	ErrInvalidCodeVerifier          = derror.ErrInvalidCodeVerifier
)
