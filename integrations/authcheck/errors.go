package authcheck

import (
	"errors"

	"github.com/Zany2/dtoken-go/core/derror"
)

// GetErrorCodeAndMessage maps core errors to stable integration responses GetErrorCodeAndMessage 灏嗘牳蹇冮敊璇槧灏勪负绋冲畾鐨勯泦鎴愬眰鍝嶅簲銆?
func GetErrorCodeAndMessage(err error) (int, string) {
	var dErr *derror.DTokenError
	if errors.As(err, &dErr) {
		return dErr.Code, dErr.Message
	}

	switch {
	case errors.Is(err, derror.ErrNotLogin):
		return derror.CodeNotLogin, err.Error()
	case errors.Is(err, derror.ErrInvalidToken),
		errors.Is(err, derror.ErrInvalidAccessToken),
		errors.Is(err, derror.ErrInvalidRefreshToken),
		errors.Is(err, derror.ErrInvalidNonce),
		errors.Is(err, derror.ErrInvalidTicket),
		errors.Is(err, derror.ErrTicketConsumed),
		errors.Is(err, derror.ErrTicketRevoked),
		errors.Is(err, derror.ErrInvalidShortKey),
		errors.Is(err, derror.ErrShortKeyConsumed),
		errors.Is(err, derror.ErrShortKeyRevoked):
		return derror.CodeTokenInvalid, err.Error()
	case errors.Is(err, derror.ErrTokenExpired),
		errors.Is(err, derror.ErrTicketExpired),
		errors.Is(err, derror.ErrShortKeyExpired):
		return derror.CodeTokenExpired, err.Error()
	case errors.Is(err, derror.ErrActiveTimeout):
		return derror.CodeActiveTimeout, err.Error()
	case errors.Is(err, derror.ErrTokenKickout), errors.Is(err, derror.ErrTokenReplaced):
		return derror.CodeKickedOut, err.Error()
	case errors.Is(err, derror.ErrPermissionDenied), errors.Is(err, derror.ErrRoleDenied):
		return derror.CodePermissionDenied, err.Error()
	case errors.Is(err, derror.ErrAccountDisabled), errors.Is(err, derror.ErrDeviceDisabled), errors.Is(err, derror.ErrServiceDisabled):
		return derror.CodeAccountDisabled, err.Error()
	case errors.Is(err, derror.ErrStorageUnavailable),
		errors.Is(err, derror.ErrSerializeFailed),
		errors.Is(err, derror.ErrTypeConvert):
		return derror.CodeStorageError, err.Error()
	case errors.Is(err, derror.ErrKeyNotFound),
		errors.Is(err, derror.ErrManagerNotFound),
		errors.Is(err, derror.ErrClientNotFound),
		errors.Is(err, derror.ErrSessionNotFound):
		return derror.CodeNotFound, err.Error()
	case errors.Is(err, derror.ErrInvalidParam),
		errors.Is(err, derror.ErrIDIsEmpty),
		errors.Is(err, derror.ErrEmptyLoginID),
		errors.Is(err, derror.ErrUserIDEmpty),
		errors.Is(err, derror.ErrInvalidDevice),
		errors.Is(err, derror.ErrClientOrClientIDEmpty),
		errors.Is(err, derror.ErrInvalidClientCredentials),
		errors.Is(err, derror.ErrInvalidGrantType),
		errors.Is(err, derror.ErrInvalidRedirectURI),
		errors.Is(err, derror.ErrInvalidScope),
		errors.Is(err, derror.ErrInvalidAuthCode),
		errors.Is(err, derror.ErrInvalidCodeVerifier),
		errors.Is(err, derror.ErrAuthCodeUsed),
		errors.Is(err, derror.ErrClientMismatch),
		errors.Is(err, derror.ErrRedirectURIMismatch),
		errors.Is(err, derror.ErrInvalidUserCredentials),
		errors.Is(err, derror.ErrTicketMismatch),
		errors.Is(err, derror.ErrShortKeyPending),
		errors.Is(err, derror.ErrShortKeyMismatch),
		errors.Is(err, derror.ErrAccountNotDisabled),
		errors.Is(err, derror.ErrDeviceNotDisabled),
		errors.Is(err, derror.ErrServiceNotDisabled),
		errors.Is(err, derror.ErrDisableLevelNotReached):
		return derror.CodeBadRequest, err.Error()
	default:
		return derror.CodeServerError, err.Error()
	}
}
