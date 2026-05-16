package authcheck

import (
	"errors"

	"github.com/Zany2/dtoken-go/core/derror"
)

// GetErrorCodeAndMessage maps core errors to stable integration responses. GetErrorCodeAndMessage 将核心错误映射为稳定的集成层响应。
func GetErrorCodeAndMessage(err error) (int, string) {
	var dErr *derror.DTokenError
	if errors.As(err, &dErr) {
		return dErr.Code, dErr.Message
	}

	switch {
	case errors.Is(err, derror.ErrNotLogin):
		return derror.CodeNotLogin, err.Error()
	case errors.Is(err, derror.ErrInvalidToken):
		return derror.CodeTokenInvalid, err.Error()
	case errors.Is(err, derror.ErrTokenExpired):
		return derror.CodeTokenExpired, err.Error()
	case errors.Is(err, derror.ErrActiveTimeout):
		return derror.CodeActiveTimeout, err.Error()
	case errors.Is(err, derror.ErrTokenKickout), errors.Is(err, derror.ErrTokenReplaced):
		return derror.CodeKickedOut, err.Error()
	case errors.Is(err, derror.ErrPermissionDenied), errors.Is(err, derror.ErrRoleDenied):
		return derror.CodePermissionDenied, err.Error()
	case errors.Is(err, derror.ErrAccountDisabled), errors.Is(err, derror.ErrDeviceDisabled), errors.Is(err, derror.ErrServiceDisabled):
		return derror.CodeAccountDisabled, err.Error()
	case errors.Is(err, derror.ErrInvalidParam), errors.Is(err, derror.ErrIDIsEmpty):
		return derror.CodeBadRequest, err.Error()
	default:
		return derror.CodeServerError, err.Error()
	}
}
