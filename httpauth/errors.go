package httpauth

import (
	"errors"

	derror "github.com/Zany2/dtoken-go/errors"
)

// GetErrorCodeAndMessage hides wrapped causes and infrastructure details, treats other custom DTokenError messages as public, and leaves original errors available to callbacks. GetErrorCodeAndMessage 隐藏包装原因及基础设施详情，将其他自定义 DTokenError 消息视为公开信息，并保留原始错误供回调使用。
func GetErrorCodeAndMessage(err error) (int, string) {
	if err == nil {
		return derror.CodeSuccess, "success"
	}

	var dErr *derror.DTokenError
	if errors.As(err, &dErr) && dErr != nil {
		if dErr.Code == derror.CodeServerError || dErr.Code == derror.CodeStorageError {
			return dErr.Code, "internal server error"
		}
		return dErr.Code, dErr.Message
	}

	// Preserve business codes and canonical messages while discarding wrapper details. 保留业务错误码和标准消息，丢弃包装层详情。
	for _, group := range []struct {
		code   int
		causes []error
	}{
		{derror.CodeNotLogin, []error{derror.ErrNotLogin}},
		{derror.CodeTokenInvalid, []error{
			derror.ErrInvalidToken,
			derror.ErrInvalidAccessToken,
			derror.ErrInvalidRefreshToken,
			derror.ErrInvalidNonce,
			derror.ErrInvalidTicket,
			derror.ErrTicketConsumed,
			derror.ErrTicketRevoked,
			derror.ErrInvalidShortKey,
			derror.ErrShortKeyConsumed,
			derror.ErrShortKeyRevoked,
		}},
		{derror.CodeTokenExpired, []error{
			derror.ErrTokenExpired,
			derror.ErrAuthCodeExpired,
			derror.ErrTicketExpired,
			derror.ErrShortKeyExpired,
		}},
		{derror.CodeActiveTimeout, []error{derror.ErrActiveTimeout}},
		{derror.CodeKickedOut, []error{derror.ErrTokenKickout, derror.ErrTokenReplaced}},
		{derror.CodePermissionDenied, []error{derror.ErrPermissionDenied, derror.ErrRoleDenied}},
		{derror.CodeAccountDisabled, []error{derror.ErrAccountDisabled, derror.ErrDeviceDisabled, derror.ErrServiceDisabled}},
		{derror.CodeStorageError, []error{derror.ErrStorageUnavailable, derror.ErrSerializeFailed, derror.ErrTypeConvert}},
		{derror.CodeNotFound, []error{
			derror.ErrKeyNotFound,
			derror.ErrManagerNotFound,
			derror.ErrClientNotFound,
			derror.ErrSessionNotFound,
		}},
		{derror.CodeBadRequest, []error{
			derror.ErrInvalidParam,
			derror.ErrIDIsEmpty,
			derror.ErrEmptyLoginID,
			derror.ErrUserIDEmpty,
			derror.ErrInvalidDevice,
			derror.ErrClientOrClientIDEmpty,
			derror.ErrInvalidClientCredentials,
			derror.ErrInvalidGrantType,
			derror.ErrInvalidRedirectURI,
			derror.ErrInvalidScope,
			derror.ErrInvalidAuthCode,
			derror.ErrInvalidCodeVerifier,
			derror.ErrAuthCodeUsed,
			derror.ErrClientMismatch,
			derror.ErrRedirectURIMismatch,
			derror.ErrInvalidUserCredentials,
			derror.ErrTicketMismatch,
			derror.ErrShortKeyPending,
			derror.ErrShortKeyMismatch,
			derror.ErrAccountNotDisabled,
			derror.ErrDeviceNotDisabled,
			derror.ErrServiceNotDisabled,
			derror.ErrDisableLevelNotReached,
		}},
	} {
		for _, cause := range group.causes {
			if errors.Is(err, cause) {
				if group.code == derror.CodeStorageError {
					return group.code, "internal server error"
				}
				return group.code, cause.Error()
			}
		}
	}
	return derror.CodeServerError, "internal server error"
}
